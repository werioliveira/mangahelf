package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/netip"
	"time"
)

type Config struct {
	SessionAbsoluteTTL time.Duration
	SessionIdleTTL     time.Duration
	CookieSecure       bool
	CookieDomain       string
	MaxSessionsPerUser int
	Argon2Memory       uint32
	Argon2Time         uint32
	Argon2Threads      uint8
	Argon2SaltLen      uint32
	Argon2KeyLen       uint32
	MaxConcurrentHash  int
	AllowedOrigins     map[string]struct{}
	TrustedProxies     []netip.Prefix
	ClientIPHeader     string
	EncryptionKeys     []EncryptionKey
	MFAWindow          time.Duration
	StepUpWindows      map[string]time.Duration
	HIBPCheck          bool
	CSRFTokenLen       int
	SessionTokenLen    int
	RPID               string
	RPOrigins          []string
}

type EncryptionKey struct {
	ID  int16
	Key []byte
}

func DefaultConfig() Config {
	return Config{
		SessionAbsoluteTTL: 30 * 24 * time.Hour,
		SessionIdleTTL:     7 * 24 * time.Hour,
		CookieSecure:       true,
		CookieDomain:       "",
		MaxSessionsPerUser: 10,
		Argon2Memory:       64 * 1024,
		Argon2Time:         3,
		Argon2Threads:      2,
		Argon2SaltLen:      16,
		Argon2KeyLen:       32,
		MaxConcurrentHash:  4,
		AllowedOrigins:     map[string]struct{}{},
		TrustedProxies:     []netip.Prefix{},
		ClientIPHeader:     "CF-Connecting-IP",
		EncryptionKeys:     []EncryptionKey{},
		MFAWindow:          30 * time.Minute,
		StepUpWindows: map[string]time.Duration{
			"role.manage":        30 * time.Minute,
			"instance.configure": 30 * time.Minute,
			"password.change":    10 * time.Minute,
			"email.change":       10 * time.Minute,
			"mfa.change":         10 * time.Minute,
			"owner.transfer":     5 * time.Minute,
		},
		HIBPCheck:       false,
		CSRFTokenLen:    32,
		SessionTokenLen: 32,
		RPID:            "",
		RPOrigins:       []string{},
	}
}

func (c *Config) Argon2Params() Argon2Params {
	return Argon2Params{
		Memory:  c.Argon2Memory,
		Time:    c.Argon2Time,
		Threads: c.Argon2Threads,
		SaltLen: c.Argon2SaltLen,
		KeyLen:  c.Argon2KeyLen,
	}
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func GenerateToken(n int) (string, error) {
	b, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func LoadEncryptionKeys(keyStrings []string) ([]EncryptionKey, error) {
	var keys []EncryptionKey
	for i, s := range keyStrings {
		if s == "" {
			continue
		}
		key, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("encryption key %d: %w", i, err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("encryption key %d must be 32 bytes", i)
		}
		keys = append(keys, EncryptionKey{ID: int16(i + 1), Key: key})
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no valid encryption keys provided")
	}
	return keys, nil
}

func (c *Config) CurrentEncryptionKey() EncryptionKey {
	if len(c.EncryptionKeys) == 0 {
		panic("no encryption keys configured")
	}
	return c.EncryptionKeys[len(c.EncryptionKeys)-1]
}