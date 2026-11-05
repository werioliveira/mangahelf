package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/netip"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Rule struct {
	Limit  int
	Window time.Duration
}

type Decision struct {
	Allowed    bool
	RetryAfter time.Duration
}

type Limiter interface {
	Allow(ctx context.Context, key string, r Rule) (Decision, error)
	AllowAll(ctx context.Context, keys []KeyRule) (Decision, error)
}

type KeyRule struct {
	Key  string
	Rule Rule
}

type memoryLimiter struct {
	mu      sync.Mutex
	limiters map[string]*rateLimiterWrapper
	lastClean time.Time
}

func NewMemoryLimiter() *memoryLimiter {
	return &memoryLimiter{
		limiters:  make(map[string]*rateLimiterWrapper),
		lastClean: time.Now(),
	}
}

func (m *memoryLimiter) getLimiter(key string, r Rule) *rate.Limiter {
	m.mu.Lock()
	defer m.mu.Unlock()

	if time.Since(m.lastClean) > 5*time.Minute {
		for k, v := range m.limiters {
			if time.Since(v.lastAccess) > 10*time.Minute {
				delete(m.limiters, k)
			}
		}
		m.lastClean = time.Now()
	}

	if w, ok := m.limiters[key]; ok {
		w.lastAccess = time.Now()
		return w.limiter
	}

	rl := rate.NewLimiter(rate.Every(r.Window/time.Duration(r.Limit)), r.Limit)
	w := &rateLimiterWrapper{limiter: rl, lastAccess: time.Now()}
	m.limiters[key] = w
	return rl
}

type rateLimiterWrapper struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

func (m *memoryLimiter) Allow(ctx context.Context, key string, r Rule) (Decision, error) {
	rl := m.getLimiter(key, r)
	if rl.Allow() {
		return Decision{Allowed: true}, nil
	}
	retry := rl.Reserve().Delay()
	return Decision{Allowed: false, RetryAfter: retry}, nil
}

func (m *memoryLimiter) AllowAll(ctx context.Context, keys []KeyRule) (Decision, error) {
	var finalDecision Decision
	for _, kr := range keys {
		d, err := m.Allow(ctx, kr.Key, kr.Rule)
		if err != nil {
			return Decision{}, err
		}
		if !d.Allowed {
			if finalDecision.RetryAfter == 0 || d.RetryAfter > finalDecision.RetryAfter {
				finalDecision = d
			}
			finalDecision.Allowed = false
		}
	}
	if finalDecision.RetryAfter == 0 {
		finalDecision.Allowed = true
	}
	return finalDecision, nil
}

func EmailHash(key string, email string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(h.Sum(nil))
}

func LoginKeys(ip netip.Addr, emailHash string) []KeyRule {
	return []KeyRule{
		{Key: "ip:" + ip.String(), Rule: Rule{Limit: 20, Window: 15 * time.Minute}},
		{Key: "email:" + emailHash, Rule: Rule{Limit: 10, Window: 15 * time.Minute}},
		{Key: "ip_email:" + ip.String() + ":" + emailHash, Rule: Rule{Limit: 5, Window: 15 * time.Minute}},
	}
}

func RegisterKeys(ip netip.Addr) []KeyRule {
	return []KeyRule{
		{Key: "ip:" + ip.String(), Rule: Rule{Limit: 5, Window: 1 * time.Hour}},
	}
}

func ForgotKeys(ip netip.Addr, emailHash string) []KeyRule {
	return []KeyRule{
		{Key: "ip:" + ip.String(), Rule: Rule{Limit: 5, Window: 1 * time.Hour}},
		{Key: "email:" + emailHash, Rule: Rule{Limit: 3, Window: 1 * time.Hour}},
	}
}

func MFAKeys(sessionID, userID string) []KeyRule {
	return []KeyRule{
		{Key: "session:" + sessionID, Rule: Rule{Limit: 5, Window: 5 * time.Minute}},
		{Key: "user:" + userID, Rule: Rule{Limit: 10, Window: 15 * time.Minute}},
	}
}

func ResendEmailKeys(userID string, ip netip.Addr) []KeyRule {
	return []KeyRule{
		{Key: "user:" + userID, Rule: Rule{Limit: 3, Window: 1 * time.Hour}},
		{Key: "ip:" + ip.String(), Rule: Rule{Limit: 3, Window: 1 * time.Hour}},
	}
}