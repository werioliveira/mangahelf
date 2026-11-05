package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/netip"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/pgxstore"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
)

type SessionData struct {
	SID       string
	UID       string
	AuthTime  int64
	MFAAt     *int64
	CSRF      string
}

type UserSessionRow struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	CreatedAt     time.Time
	LastSeenAt    time.Time
	AuthTime      time.Time
	MFAAt         *time.Time
	IP            netip.Addr
	UserAgent     string
	RevokedAt     *time.Time
	RevokedReason string
}

type Identity struct {
	UserID   uuid.UUID
	AuthTime time.Time
	MFA      bool
}

type SessionService struct {
	sm  *scs.SessionManager
	cfg Config
	q   *Queries
}

func NewSessionManager(pool *pgxpool.Pool, cfg Config) *scs.SessionManager {
	sm := scs.New()
	sm.Store = newHashedStore(pgxstore.New(pool))

	sm.Lifetime    = cfg.SessionAbsoluteTTL
	sm.IdleTimeout = cfg.SessionIdleTTL

	sm.Cookie.Name     = "__Host-ms_session"
	sm.Cookie.Path     = "/"
	sm.Cookie.Secure   = cfg.CookieSecure
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Persist  = true
	return sm
}

type hashedStore struct {
	inner scs.Store
}

func newHashedStore(s scs.Store) hashedStore {
	return hashedStore{inner: s}
}

func keyOf(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (h hashedStore) Find(token string) ([]byte, bool, error) {
	return h.inner.Find(keyOf(token))
}

func (h hashedStore) Commit(token string, data []byte, expiry time.Time) error {
	return h.inner.Commit(keyOf(token), data, expiry)
}

func (h hashedStore) Delete(token string) error {
	return h.inner.Delete(keyOf(token))
}

func (s *SessionService) Establish(ctx context.Context, id Identity, ip netip.Addr, ua string) error {
	if err := s.sm.RenewToken(ctx); err != nil {
		return err
	}
	sid := uuid.New()
	csrf, _ := GenerateToken(s.cfg.CSRFTokenLen)

	var mfaAt *time.Time
	if id.MFA {
		t := id.AuthTime
		mfaAt = &t
	}
	if err := s.q.InsertUserSession(ctx, sid, id.UserID, id.AuthTime, mfaAt, ip, truncate(ua, 512)); err != nil {
		return err
	}

	if err := s.q.RevokeOldestBeyond(ctx, id.UserID, s.cfg.MaxSessionsPerUser, "session_limit"); err != nil {
		return err
	}

	s.sm.Put(ctx, "sid", sid.String())
	s.sm.Put(ctx, "uid", id.UserID.String())
	s.sm.Put(ctx, "auth_time", id.AuthTime.Unix())
	if mfaAt != nil {
		s.sm.Put(ctx, "mfa_at", mfaAt.Unix())
	}
	s.sm.Put(ctx, "csrf", csrf)
	return nil
}

func (s *SessionService) RevokeAll(ctx context.Context, userID uuid.UUID, exceptSID *uuid.UUID, reason string) error {
	return s.q.RevokeUserSessions(ctx, userID, exceptSID, reason)
}

func (s *SessionService) RevokeCurrent(ctx context.Context, reason string) error {
	sid := s.sm.GetString(ctx, "sid")
	if sid == "" {
		return nil
	}
	sidUUID, err := uuid.Parse(sid)
	if err != nil {
		return err
	}
	return s.q.RevokeSession(ctx, sidUUID, reason)
}

func (s *SessionService) TouchLastSeen(ctx context.Context) error {
	sid := s.sm.GetString(ctx, "sid")
	if sid == "" {
		return nil
	}
	sidUUID, err := uuid.Parse(sid)
	if err != nil {
		return err
	}
	return s.q.TouchSession(ctx, sidUUID)
}

func (s *SessionService) GetPrincipal(ctx context.Context) (*Principal, bool) {
	sid := s.sm.GetString(ctx, "sid")
	if sid == "" {
		return nil, false
	}
	sidUUID, err := uuid.Parse(sid)
	if err != nil {
		return nil, false
	}
	row, err := s.q.SessionPrincipal(ctx, sidUUID)
	if err != nil {
		return nil, false
	}
	if row.RevokedAt != nil || row.Status != "active" {
		return nil, false
	}
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
	return p, true
}

func (s *SessionService) UpdateMFAAt(ctx context.Context, mfaAt time.Time) error {
	sid := s.sm.GetString(ctx, "sid")
	if sid == "" {
		return ErrSessionNotFound
	}
	sidUUID, err := uuid.Parse(sid)
	if err != nil {
		return err
	}
	s.sm.Put(ctx, "mfa_at", mfaAt.Unix())
	return s.q.UpdateSessionMFA(ctx, sidUUID, mfaAt)
}

func (s *SessionService) GetCSRF(ctx context.Context) string {
	return s.sm.GetString(ctx, "csrf")
}

func (s *SessionService) RenewCSRF(ctx context.Context) (string, error) {
	csrf, err := GenerateToken(s.cfg.CSRFTokenLen)
	if err != nil {
		return "", err
	}
	s.sm.Put(ctx, "csrf", csrf)
	return csrf, nil
}

func (s *SessionService) Destroy(ctx context.Context) error {
	return s.sm.Destroy(ctx)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}