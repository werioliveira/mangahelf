package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/text/unicode/norm"
)

type Argon2Params struct {
	Memory  uint32
	Time    uint32
	Threads uint8
	SaltLen uint32
	KeyLen  uint32
}

var DefaultArgon2 = Argon2Params{Memory: 64 * 1024, Time: 3, Threads: 2, SaltLen: 16, KeyLen: 32}

type Hasher struct {
	p   Argon2Params
	sem chan struct{}
}

func NewHasher(p Argon2Params, maxConcurrent int) *Hasher {
	return &Hasher{p: p, sem: make(chan struct{}, maxConcurrent)}
}

func (h *Hasher) acquire(ctx context.Context) error {
	select {
	case h.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ErrHashBusy
	}
}

func (h *Hasher) release() { <-h.sem }

var b64 = base64.RawStdEncoding

func (h *Hasher) Hash(ctx context.Context, password string) (string, error) {
	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer h.release()

	salt := make([]byte, h.p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, h.p.Time, h.p.Memory, h.p.Threads, h.p.KeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.p.Memory, h.p.Time, h.p.Threads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

func (h *Hasher) Verify(ctx context.Context, password, encoded string) (ok, needsRehash bool, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, false, ErrInvalidHashFormat
	}
	var v int
	var p Argon2Params
	if _, err := fmt.Sscanf(parts[2], "v=%d", &v); err != nil || v != argon2.Version {
		return false, false, ErrUnsupportedVersion
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Time, &p.Threads); err != nil {
		return false, false, err
	}
	if p.Memory > 256*1024 || p.Time > 10 || p.Threads > 16 {
		return false, false, ErrParamsOutOfBounds
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, false, err
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil {
		return false, false, err
	}

	if err := h.acquire(ctx); err != nil {
		return false, false, err
	}
	defer h.release()

	got := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, uint32(len(want)))
	ok = subtle.ConstantTimeCompare(got, want) == 1
	needsRehash = p.Memory != h.p.Memory || p.Time != h.p.Time || p.Threads != h.p.Threads
	return ok, needsRehash, nil
}

func NormalizePassword(password string) string {
	return norm.NFKC.String(password)
}

func ValidatePasswordPolicy(password, email, username string) error {
	normalized := NormalizePassword(password)
	if len(normalized) < 12 {
		return ErrPasswordTooShort
	}
	if len([]byte(normalized)) > 256 {
		return ErrPasswordTooLong
	}
	if email != "" && strings.Contains(strings.ToLower(normalized), strings.ToLower(email)) {
		return ErrPasswordContainsID
	}
	if username != "" && strings.Contains(strings.ToLower(normalized), strings.ToLower(username)) {
		return ErrPasswordContainsID
	}
	return nil
}

var commonPasswords = map[string]struct{}{
	"password": {}, "123456": {}, "123456789": {}, "qwerty": {}, "abc123": {},
	"password123": {}, "admin": {}, "letmein": {}, "welcome": {}, "monkey": {},
	"1234567890": {}, "dragon": {}, "master": {}, "hello": {}, "freedom": {},
	"whatever": {}, "qazwsx": {}, "trustno1": {}, "654321": {}, "jordan": {},
	"harley": {}, "robert": {}, "matthew": {}, "jennifer": {}, "joshua": {},
	"andrew": {}, "brandon": {}, "daniel": {}, "christian": {}, "michelle": {},
	"ashley": {}, "amanda": {}, "megan": {}, "nicole": {}, "jessica": {},
	"heather": {}, "melissa": {}, "angela": {}, "stephanie": {}, "samantha": {},
	"laura": {}, "emily": {}, "rebecca": {}, "kimberly": {},
	"katherine": {}, "crystal": {}, "tiffany": {}, "victoria": {}, "denise": {},
	"karen": {}, "lisa": {}, "nancy": {}, "betty": {},
	"sandra": {}, "margaret": {}, "donna": {}, "carol": {}, "ruth": {},
	"sharon": {}, "sarah": {},
	"deborah": {}, "dorothy": {}, "amy": {},
	"brenda": {}, "emma": {}, "olivia": {}, "cynthia": {}, "marie": {},
	"janet": {}, "catherine": {}, "frances": {}, "christine": {}, "sara": {},
	"janice": {}, "joyce": {}, "virginia": {}, "maria": {},
	"diane": {}, "alice": {}, "julie": {}, "kelly": {},
	"joan": {}, "evelyn": {}, "lauren": {}, "judith": {},
	"cheryl": {}, "andrea": {}, "hannah": {}, "martha": {}, "jacqueline": {},
	"gloria": {}, "teresa": {}, "katie": {},
	"elizabeth": {},
}

func CheckCommonPassword(password string) error {
	normalized := strings.ToLower(NormalizePassword(password))
	if _, exists := commonPasswords[normalized]; exists {
		return ErrCommonPassword
	}
	return nil
}