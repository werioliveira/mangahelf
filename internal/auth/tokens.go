package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

var (
)

type TokenPurpose string

const (
	PurposeVerifyEmail   TokenPurpose = "verify_email"
	PurposeResetPassword TokenPurpose = "reset_password"
	PurposeChangeEmail   TokenPurpose = "change_email"
)

var tokenTTLs = map[TokenPurpose]time.Duration{
	PurposeVerifyEmail:   24 * time.Hour,
	PurposeResetPassword: 30 * time.Minute,
	PurposeChangeEmail:   1 * time.Hour,
}

func NewToken() (plain string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(plain))
	return plain, sum[:], nil
}

type TokenMeta map[string]any

func (s *TokenService) CreateToken(ctx context.Context, userID uuid.UUID, purpose TokenPurpose, meta TokenMeta) (string, error) {
	if _, ok := tokenTTLs[purpose]; !ok {
		return "", ErrInvalidPurpose
	}

	if err := s.q.InvalidateAuthTokens(ctx, userID, string(purpose)); err != nil {
		return "", err
	}

	plain, hash, err := NewToken()
	if err != nil {
		return "", err
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().UTC().Add(tokenTTLs[purpose])
	if err := s.q.InsertAuthToken(ctx, userID, string(purpose), hash, metaJSON, expiresAt); err != nil {
		return "", err
	}

	return plain, nil
}

func (s *TokenService) ConsumeToken(ctx context.Context, plainToken string, purpose TokenPurpose) (uuid.UUID, TokenMeta, error) {
	sum := sha256.Sum256([]byte(plainToken))
	userID, metaJSON, err := s.q.ConsumeToken(ctx, sum[:], string(purpose))
	if err != nil {
		return uuid.Nil, nil, err
	}

	var meta TokenMeta
	if len(metaJSON) > 0 {
		if err := json.Unmarshal(metaJSON, &meta); err != nil {
			return uuid.Nil, nil, err
		}
	}

	return userID, meta, nil
}

type TokenService struct {
	q *Queries
	cfg Config
}