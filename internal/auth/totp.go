package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

var (
)

type TOTPService struct {
	q    *Queries
	cfg  Config
}

func (s *TOTPService) BeginEnrollment(ctx context.Context, userID uuid.UUID) (string, error) {
	key := s.cfg.CurrentEncryptionKey()
	if len(key.Key) != 32 {
		return "", ErrNoEncryptionKey
	}

	secret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "MangaShelf",
		AccountName: "",
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return "", err
	}

	encSecret, keyID, err := s.Encrypt([]byte(secret.Secret()))
	if err != nil {
		return "", err
	}

	if err := s.q.InsertTOTP(ctx, userID, encSecret, keyID); err != nil {
		return "", err
	}

	return secret.URL(), nil
}

func (s *TOTPService) ConfirmEnrollment(ctx context.Context, userID uuid.UUID, code string) ([]string, error) {
	rec, err := s.q.GetTOTP(ctx, userID)
	if err != nil {
		return nil, err
	}

	secret, err := s.Decrypt(rec.SecretEnc, rec.KeyID)
	if err != nil {
		return nil, err
	}

	if !s.VerifyCode(secret, code) {
		return nil, ErrInvalidTOTPCode
	}

	if err := s.q.ConfirmTOTP(ctx, userID); err != nil {
		return nil, err
	}

	codes := make([]string, 10)
	codeHMACs := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		code, err := GenerateToken(10)
		if err != nil {
			return nil, err
		}
		codes[i] = code
		h := hmacSHA256(s.cfg.CurrentEncryptionKey().Key, []byte(code))
		codeHMACs[i] = h
	}

	if err := s.q.InsertRecoveryCodes(ctx, userID, codeHMACs); err != nil {
		return nil, err
	}

	return codes, nil
}

func (s *TOTPService) Verify(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	rec, err := s.q.GetTOTP(ctx, userID)
	if err != nil {
		return false, err
	}

	if rec.ConfirmedAt == nil {
		return false, ErrTOTPNotConfirmed
	}

	secret, err := s.Decrypt(rec.SecretEnc, rec.KeyID)
	if err != nil {
		return false, err
	}

	if !s.VerifyCode(secret, code) {
		return false, ErrInvalidTOTPCode
	}

	now := time.Now().UTC()
	for _, off := range []int{-1, 0, 1} {
		t := now.Add(time.Duration(off) * 30 * time.Second)
		step := t.Unix() / 30
		n, err := s.q.AdvanceTOTPStep(ctx, userID, step)
		if err != nil {
			return false, err
		}
		if n == 1 {
			return true, nil
		}
	}

	return false, nil
}

func (s *TOTPService) VerifyCode(secret []byte, code string) bool {
	opts := totp.ValidateOpts{Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1}
	now := time.Now().UTC()
	for _, off := range []int{-1, 0, 1} {
		t := now.Add(time.Duration(off) * 30 * time.Second)
		want, err := totp.GenerateCodeCustom(string(secret), t, opts)
		if err != nil {
			continue
		}
		if subtleConstantTimeCompare([]byte(want), []byte(code)) {
			return true
		}
	}
	return false
}

func (s *TOTPService) RemoveTOTP(ctx context.Context, userID uuid.UUID) error {
	if err := s.q.DeleteTOTP(ctx, userID); err != nil {
		return err
	}
	err := s.q.ExecContext(ctx, `DELETE FROM user_recovery_codes WHERE user_id = $1`, userID)
	return err
}

func (s *TOTPService) GetRecoveryCodeCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.q.GetRecoveryCodeCount(ctx, userID)
}

func (s *TOTPService) Enrolled(ctx context.Context, userID uuid.UUID) bool {
	_, err := s.q.GetTOTP(ctx, userID)
	return err == nil
}

func (s *TOTPService) Encrypt(plain []byte) ([]byte, int16, error) {
	key := s.cfg.CurrentEncryptionKey()
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return nil, 0, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, 0, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, 0, err
	}
	ct := gcm.Seal(nonce, nonce, plain, []byte("totp:v1"))
	return ct, key.ID, nil
}

func (s *TOTPService) Decrypt(ct []byte, keyID int16) ([]byte, error) {
	var key EncryptionKey
	for _, k := range s.cfg.EncryptionKeys {
		if k.ID == keyID {
			key = k
			break
		}
	}
	if key.Key == nil {
		key = s.cfg.CurrentEncryptionKey()
	}

	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ct) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ct[:nonceSize], ct[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, []byte("totp:v1"))
	if err != nil {
		return nil, err
	}
	return plain, nil
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func subtleConstantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}