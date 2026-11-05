package auth

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
)

type contextKey string

const principalKey contextKey = "auth.principal"

var (
	ErrNoPrincipal = errors.New("auth: no principal in context")
)

type Principal struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Status    string
	AuthTime  time.Time
	MFAAt     *time.Time
}

func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalKey).(*Principal)
	return p, ok
}

type Middleware struct {
	sm  *scs.SessionManager
	cfg Config
	q   *Queries
}

func NewMiddleware(sm *scs.SessionManager, cfg Config, q *Queries) *Middleware {
	return &Middleware{sm: sm, cfg: cfg, q: q}
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sid := m.sm.GetString(ctx, "sid")
		if sid == "" {
			next.ServeHTTP(w, r)
			return
		}

		sidUUID, err := uuid.Parse(sid)
		if err != nil {
			m.sm.Destroy(ctx)
			next.ServeHTTP(w, r)
			return
		}

		row, err := m.q.SessionPrincipal(ctx, sidUUID)
		switch {
		case errors.Is(err, ErrSessionNotFound):
			m.sm.Destroy(ctx)
			next.ServeHTTP(w, r)
			return
		case err != nil:
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if row.RevokedAt != nil || row.Status != "active" || sessionExpired(row, m.cfg) {
			m.sm.Destroy(ctx)
			next.ServeHTTP(w, r)
			return
		}

		m.touchLastSeen(ctx, row)

		var mfaAt *time.Time
		if row.MFAAt != nil {
			mfaAt = row.MFAAt
		}

		p := &Principal{
			UserID:    row.UserID,
			SessionID: row.SID,
			Status:    row.Status,
			AuthTime:  row.AuthTime,
			MFAAt:     mfaAt,
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(ctx, p)))
	})
}

func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := PrincipalFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) RequireRecentAuth(maxAge time.Duration, needMFA bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ref := p.AuthTime
			if needMFA {
				if p.MFAAt == nil {
					http.Error(w, "mfa required", http.StatusForbidden)
					return
				}
				ref = *p.MFAAt
			}
			if time.Since(ref) > maxAge {
				http.Error(w, "reauth required", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (m *Middleware) touchLastSeen(ctx context.Context, row *SessionPrincipalRow) {
	if row.LastSeenAt == nil {
		return
	}
	if time.Since(*row.LastSeenAt) > time.Minute {
		m.q.TouchSession(ctx, row.SID)
	}
}

func sessionExpired(row *SessionPrincipalRow, cfg Config) bool {
	if time.Since(row.AuthTime) > cfg.SessionAbsoluteTTL {
		return true
	}
	if row.MFAAt != nil && time.Since(*row.MFAAt) > cfg.SessionIdleTTL {
		return true
	}
	if row.LastSeenAt != nil && time.Since(*row.LastSeenAt) > cfg.SessionIdleTTL {
		return true
	}
	return false
}

func ClientIP(r *http.Request, trusted []netip.Prefix, header string) netip.Addr {
	remote, _ := netip.ParseAddrPort(r.RemoteAddr)
	ip := remote.Addr()
	for _, p := range trusted {
		if p.Contains(ip) {
			if v := r.Header.Get(header); v != "" {
				if parsed, err := netip.ParseAddr(strings.TrimSpace(v)); err == nil {
					return parsed
				}
			}
			break
		}
	}
	return ip
}