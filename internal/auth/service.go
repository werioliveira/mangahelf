package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mangashelf/mangashelf/internal/audit"
	"golang.org/x/crypto/argon2"
)

var (
)

type Service struct {
	cfg         Config
	q           *Queries
	hasher      *Hasher
	limiter     Limiter
	sessions    *SessionService
	tokens      *TokenService
	totp        *TOTPService
	webauthn    *WebAuthnService
	audit       *audit.Service
	sm          *scs.SessionManager
	dummyHash   string
	serverKey   []byte
}

type LoginResult struct {
	Done        bool
	MFARequired bool
	MFAMethods  []string
}

type RegisterInput struct {
	Email    string
	Username string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

func NewService(cfg Config, pool *pgxpool.Pool, limiter Limiter, audit *audit.Service) (*Service, error) {
	q := NewQueries(pool)
	hasher := NewHasher(cfg.Argon2Params(), cfg.MaxConcurrentHash)
	sm := NewSessionManager(pool, cfg)
	sessions := &SessionService{sm: sm, cfg: cfg, q: q}
	tokens := &TokenService{q: q, cfg: cfg}
	totp := &TOTPService{q: q, cfg: cfg}

	var webauthn *WebAuthnService
	if cfg.RPID != "" && len(cfg.RPOrigins) > 0 {
		wa, err := NewWebAuthnService(cfg, q)
		if err != nil {
			return nil, err
		}
		webauthn = wa
	}

	dummySalt := make([]byte, 16)
	rand.Read(dummySalt)
	dummyKey := argon2.IDKey([]byte("dummy"), dummySalt, cfg.Argon2Time, cfg.Argon2Memory, cfg.Argon2Threads, cfg.Argon2KeyLen)
	dummyHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, cfg.Argon2Memory, cfg.Argon2Time, cfg.Argon2Threads,
		base64.RawStdEncoding.EncodeToString(dummySalt),
		base64.RawStdEncoding.EncodeToString(dummyKey))

	serverKey := make([]byte, 32)
	rand.Read(serverKey)

	return &Service{
		cfg:       cfg,
		q:         q,
		hasher:    hasher,
		limiter:   limiter,
		sessions:  sessions,
		tokens:    tokens,
		totp:      totp,
		webauthn:  webauthn,
		audit:     audit,
		sm:        sm,
		dummyHash: dummyHash,
		serverKey: serverKey,
	}, nil
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (*User, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	username := strings.ToLower(strings.TrimSpace(in.Username))
	password := NormalizePassword(in.Password)

	if err := ValidatePasswordPolicy(password, email, username); err != nil {
		return nil, err
	}
	if err := CheckCommonPassword(password); err != nil {
		return nil, err
	}

	keys := RegisterKeys(netip.Addr{})
	if d, _ := s.limiter.AllowAll(ctx, keys); !d.Allowed {
		s.audit.Log(ctx, "auth.register.ratelimited", nil, audit.Meta{"ip": keys[0].Key})
		return nil, ErrRateLimited{RetryAfter: d.RetryAfter}
	}

	exists, err := s.q.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	passwordHash, err := s.hasher.Hash(ctx, password)
	if err != nil {
		return nil, err
	}

	var u *User
	if exists != nil {
		s.audit.Log(ctx, "auth.register.existing_email", nil, audit.Meta{"email_hash": s.hmacEmail(email)})
		return nil, ErrEmailExists
	}

	u, err = s.q.CreateUser(ctx, email, username, passwordHash)
	if err != nil {
		return nil, err
	}

	token, err := s.tokens.CreateToken(ctx, u.ID, PurposeVerifyEmail, TokenMeta{})
	if err != nil {
		return nil, err
	}

	s.audit.Log(ctx, "auth.register.success", &audit.Actor{ID: u.ID}, audit.Meta{"email_hash": s.hmacEmail(email)})

	go func() {
		s.sendEmail(context.Background(), email, "verify_email", token)
	}()

	return u, nil
}

func (s *Service) Login(ctx context.Context, in LoginInput, ip netip.Addr, ua string) (*LoginResult, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	password := NormalizePassword(in.Password)
	eh := s.hmacEmail(email)

	if d, _ := s.limiter.AllowAll(ctx, LoginKeys(ip, eh)); !d.Allowed {
		s.audit.Log(ctx, "auth.login.ratelimited", nil, audit.Meta{"email_hash": eh, "ip": ip.String()})
		return nil, ErrRateLimited{RetryAfter: d.RetryAfter}
	}

	u, err := s.q.GetUserByEmail(ctx, email)
	hash := s.dummyHash
	if err == nil && u.PasswordHash != nil {
		hash = *u.PasswordHash
	} else if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	ok, rehash, verr := s.hasher.Verify(ctx, password, hash)
	if verr != nil {
		return nil, verr
	}
	if !ok || u == nil || u.PasswordHash == nil {
		s.audit.Log(ctx, "auth.login.failure", nil, audit.Meta{"email_hash": eh, "ip": ip.String()})
		return nil, ErrInvalidCredentials
	}

	switch u.Status {
	case "banned":
		s.audit.Log(ctx, "auth.login.blocked", &audit.Actor{ID: u.ID}, audit.Meta{})
		return nil, ErrAccountBlocked
	case "suspended":
		s.audit.Log(ctx, "auth.login.blocked", &audit.Actor{ID: u.ID}, audit.Meta{})
		return nil, ErrAccountSuspended
	}

	if rehash {
		newHash, _ := s.hasher.Hash(ctx, password)
		s.q.UpdateUserPassword(ctx, u.ID, newHash)
	}

	mfaMethods := []string{}
	if s.totp.Enrolled(ctx, u.ID) {
		mfaMethods = append(mfaMethods, "totp")
	}
	if s.webauthn != nil {
		count, _ := s.q.GetCredentialCount(ctx, u.ID)
		if count > 0 {
			mfaMethods = append(mfaMethods, "passkey")
		}
	}

	if len(mfaMethods) > 0 {
		sid := uuid.New()
		if err := s.q.InsertUserSession(ctx, sid, u.ID, time.Now().UTC(), nil, ip, ua); err != nil {
			return nil, err
		}
		s.sm.Put(ctx, "sid", sid.String())
		s.sm.Put(ctx, "uid", u.ID.String())
		s.sm.Put(ctx, "auth_time", time.Now().UTC().Unix())
		s.sm.Put(ctx, "mfa_pending", true)
		s.sm.Put(ctx, "mfa_pending_uid", u.ID.String())
		s.sm.Put(ctx, "mfa_pending_until", time.Now().UTC().Add(5*time.Minute).Unix())
		csrf, _ := GenerateToken(s.cfg.CSRFTokenLen)
		s.sm.Put(ctx, "csrf", csrf)

		s.audit.Log(ctx, "auth.login.mfa_required", &audit.Actor{ID: u.ID}, audit.Meta{"methods": mfaMethods})
		return &LoginResult{MFARequired: true, MFAMethods: mfaMethods}, nil
	}

	if err := s.sessions.Establish(ctx, Identity{UserID: u.ID, AuthTime: time.Now().UTC()}, ip, ua); err != nil {
		return nil, err
	}

	s.audit.Log(ctx, "auth.login.success", &audit.Actor{ID: u.ID}, audit.Meta{"ip": ip.String()})
	return &LoginResult{Done: true}, nil
}

func (s *Service) CompleteMFA(ctx context.Context, method, code string, ip netip.Addr, ua string) (*LoginResult, error) {
	pendingUID := s.sm.GetString(ctx, "mfa_pending_uid")
	pendingUntil := s.sm.GetInt64(ctx, "mfa_pending_until")
	if pendingUID == "" || pendingUntil < time.Now().UTC().Unix() {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(pendingUID)
	if err != nil {
		return nil, err
	}

	var ok bool
	switch method {
	case "totp":
		ok, err = s.totp.Verify(ctx, userID, code)
	case "recovery":
		ok, err = s.VerifyRecoveryCode(ctx, userID, code)
	case "passkey":
		if s.webauthn == nil {
			return nil, ErrInvalidToken
		}
		return nil, ErrInvalidToken
	default:
		return nil, ErrInvalidToken
	}

	if err != nil {
		return nil, err
	}
	if !ok {
		s.audit.Log(ctx, "auth.mfa.failure", &audit.Actor{ID: userID}, audit.Meta{"method": method})
		return nil, ErrInvalidCredentials
	}

	if err := s.sessions.Establish(ctx, Identity{UserID: userID, AuthTime: time.Now().UTC(), MFA: true}, ip, ua); err != nil {
		return nil, err
	}

s.sm.Remove(ctx, "mfa_pending")
s.sm.Remove(ctx, "mfa_pending_uid")
s.sm.Remove(ctx, "mfa_pending_until")

	s.audit.Log(ctx, "auth.mfa.success", &audit.Actor{ID: userID}, audit.Meta{"method": method})
	return &LoginResult{Done: true}, nil
}

func (s *Service) Logout(ctx context.Context) error {
	sid := s.sm.GetString(ctx, "sid")
	if sid != "" {
		sidUUID, _ := uuid.Parse(sid)
		s.q.RevokeSession(ctx, sidUUID, "logout")
	}
	s.audit.Log(ctx, "auth.logout", nil, audit.Meta{})
	return s.sm.Destroy(ctx)
}

func (s *Service) ForgotPassword(ctx context.Context, email string, ip netip.Addr) error {
	email = strings.ToLower(strings.TrimSpace(email))
	eh := s.hmacEmail(email)

	if d, _ := s.limiter.AllowAll(ctx, ForgotKeys(ip, eh)); !d.Allowed {
		return ErrRateLimited{RetryAfter: d.RetryAfter}
	}

	u, err := s.q.GetUserByEmail(ctx, email)
	if err == nil {
		token, err := s.tokens.CreateToken(ctx, u.ID, PurposeResetPassword, TokenMeta{})
		if err != nil {
			return err
		}
		go func() {
			s.sendEmail(context.Background(), email, "reset_password", token)
		}()
	}

	s.audit.Log(ctx, "auth.forgot.requested", nil, audit.Meta{"email_hash": eh})
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	password := NormalizePassword(newPassword)
	if err := ValidatePasswordPolicy(password, "", ""); err != nil {
		return err
	}
	if err := CheckCommonPassword(password); err != nil {
		return err
	}

	userID, meta, err := s.tokens.ConsumeToken(ctx, token, PurposeResetPassword)
	if err != nil {
		return err
	}

	newHash, err := s.hasher.Hash(ctx, password)
	if err != nil {
		return err
	}

	if err := s.q.UpdateUserPassword(ctx, userID, newHash); err != nil {
		return err
	}

	if err := s.q.RevokeUserSessions(ctx, userID, nil, "password_reset"); err != nil {
		return err
	}

	var newEmail string
	if meta != nil {
		if e, ok := meta["new_email"].(string); ok {
			newEmail = e
		}
	}
	if newEmail != "" {
		s.q.UpdateUserEmail(ctx, userID, newEmail, false)
	}

	s.audit.Log(ctx, "auth.reset.success", &audit.Actor{ID: userID}, audit.Meta{})
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, current, new string, ip netip.Addr) error {
	u, err := s.q.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.PasswordHash == nil {
		return ErrInvalidCredentials
	}

	ok, _, err := s.hasher.Verify(ctx, NormalizePassword(current), *u.PasswordHash)
	if err != nil || !ok {
		s.audit.Log(ctx, "auth.password.change_failure", &audit.Actor{ID: userID}, audit.Meta{})
		return ErrInvalidCredentials
	}

	password := NormalizePassword(new)
	if err := ValidatePasswordPolicy(password, u.Email, u.Username); err != nil {
		return err
	}
	if err := CheckCommonPassword(password); err != nil {
		return err
	}

	newHash, err := s.hasher.Hash(ctx, password)
	if err != nil {
		return err
	}

	if err := s.q.UpdateUserPassword(ctx, userID, newHash); err != nil {
		return err
	}

	if err := s.q.RevokeUserSessions(ctx, userID, nil, "password_change"); err != nil {
		return err
	}

	s.audit.Log(ctx, "auth.password.changed", &audit.Actor{ID: userID}, audit.Meta{})
	return nil
}

func (s *Service) ChangeEmail(ctx context.Context, userID uuid.UUID, newEmail string) error {
	newEmail = strings.ToLower(strings.TrimSpace(newEmail))
	if newEmail == "" {
		return errors.New("auth: email required")
	}

	existing, err := s.q.GetUserByEmail(ctx, newEmail)
	if err == nil && existing.ID != userID {
		return ErrEmailExists
	}

	token, err := s.tokens.CreateToken(ctx, userID, PurposeChangeEmail, TokenMeta{"new_email": newEmail})
	if err != nil {
		return err
	}

	go func() {
		s.sendEmail(context.Background(), newEmail, "change_email", token)
	}()

	s.audit.Log(ctx, "auth.email.change_requested", &audit.Actor{ID: userID}, audit.Meta{"new_email": newEmail})
	return nil
}

func (s *Service) ConfirmEmailChange(ctx context.Context, token string) error {
	userID, meta, err := s.tokens.ConsumeToken(ctx, token, PurposeChangeEmail)
	if err != nil {
		return err
	}

	var newEmail string
	if meta != nil {
		if e, ok := meta["new_email"].(string); ok {
			newEmail = e
		}
	}
	if newEmail == "" {
		return ErrInvalidToken
	}

	if err := s.q.UpdateUserEmail(ctx, userID, newEmail, true); err != nil {
		return err
	}

	if err := s.q.RevokeUserSessions(ctx, userID, nil, "email_change"); err != nil {
		return err
	}

	s.audit.Log(ctx, "auth.email.changed", &audit.Actor{ID: userID}, audit.Meta{"new_email": newEmail})
	return nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	userID, _, err := s.tokens.ConsumeToken(ctx, token, PurposeVerifyEmail)
	if err != nil {
		return err
	}

	if err := s.q.ExecContext(ctx, `UPDATE users SET email_verified_at = now() WHERE id = $1`, userID); err != nil {
		return err
	}

	s.audit.Log(ctx, "auth.email.verified", &audit.Actor{ID: userID}, audit.Meta{})
	return nil
}

func (s *Service) ResendVerification(ctx context.Context, email string, ip netip.Addr) error {
	email = strings.ToLower(strings.TrimSpace(email))
	eh := s.hmacEmail(email)

	if d, _ := s.limiter.AllowAll(ctx, ResendEmailKeys("", ip)); !d.Allowed {
		return ErrRateLimited{RetryAfter: d.RetryAfter}
	}

	u, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil
	}

	if u.EmailVerifiedAt != nil {
		return nil
	}

	token, err := s.tokens.CreateToken(ctx, u.ID, PurposeVerifyEmail, TokenMeta{})
	if err != nil {
		return err
	}

	go func() {
		s.sendEmail(context.Background(), email, "verify_email", token)
	}()

	s.audit.Log(ctx, "auth.verification.resent", &audit.Actor{ID: u.ID}, audit.Meta{"email_hash": eh})
	return nil
}

func (s *Service) EnrollTOTP(ctx context.Context, userID uuid.UUID) (string, error) {
	return s.totp.BeginEnrollment(ctx, userID)
}

func (s *Service) ConfirmTOTP(ctx context.Context, userID uuid.UUID, code string) ([]string, error) {
	return s.totp.ConfirmEnrollment(ctx, userID, code)
}

func (s *Service) RemoveTOTP(ctx context.Context, userID uuid.UUID) error {
	return s.totp.RemoveTOTP(ctx, userID)
}

func (s *Service) GetTOTPStatus(ctx context.Context, userID uuid.UUID) (enrolled, confirmed bool, recoveryCount int, err error) {
	rec, err := s.q.GetTOTP(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return false, false, 0, nil
		}
		return false, false, 0, err
	}
	enrolled = true
	confirmed = rec.ConfirmedAt != nil
	recoveryCount, _ = s.q.GetRecoveryCodeCount(ctx, userID)
	return
}

func (s *Service) VerifyRecoveryCode(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	h := hmac.New(sha256.New, s.cfg.CurrentEncryptionKey().Key)
	h.Write([]byte(code))
	return s.q.ConsumeRecoveryCode(ctx, userID, h.Sum(nil))
}

func (s *Service) BeginWebAuthnRegistration(ctx context.Context, userID uuid.UUID, username string) (webauthn.SessionData, error) {
	if s.webauthn == nil {
		return webauthn.SessionData{}, ErrWebAuthnNotConfigured
	}
	return s.webauthn.BeginRegistration(ctx, userID, username)
}

func (s *Service) FinishWebAuthnRegistration(ctx context.Context, userID uuid.UUID, sessionData webauthn.SessionData, r *http.Request) error {
	if s.webauthn == nil {
		return ErrWebAuthnNotConfigured
	}
	return s.webauthn.FinishRegistration(ctx, userID, sessionData, r)
}

func (s *Service) BeginWebAuthnLogin(ctx context.Context) (webauthn.SessionData, error) {
	if s.webauthn == nil {
		return webauthn.SessionData{}, ErrWebAuthnNotConfigured
	}
	return s.webauthn.BeginLogin(ctx)
}

func (s *Service) FinishWebAuthnLogin(ctx context.Context, sessionData webauthn.SessionData, r *http.Request) (*LoginResult, error) {
	if s.webauthn == nil {
		return nil, ErrWebAuthnNotConfigured
	}

	userID, err := s.webauthn.FinishLogin(ctx, sessionData, r)
	if err != nil {
		return nil, err
	}

	if err := s.sessions.Establish(ctx, Identity{UserID: userID, AuthTime: time.Now().UTC(), MFA: true}, netip.Addr{}, ""); err != nil {
		return nil, err
	}

	s.audit.Log(ctx, "auth.webauthn.login.success", &audit.Actor{ID: userID}, audit.Meta{})
	return &LoginResult{Done: true}, nil
}

func (s *Service) ListWebAuthnCredentials(ctx context.Context, userID uuid.UUID) ([]CredentialInfo, error) {
	if s.webauthn == nil {
		return nil, ErrWebAuthnNotConfigured
	}
	return s.webauthn.ListCredentials(ctx, userID)
}

func (s *Service) RemoveWebAuthnCredential(ctx context.Context, userID uuid.UUID, credentialID []byte) error {
	if s.webauthn == nil {
		return ErrWebAuthnNotConfigured
	}
	return s.webauthn.RemoveCredential(ctx, userID, credentialID)
}

func (s *Service) hmacEmail(email string) string {
	h := hmac.New(sha256.New, s.serverKey)
	h.Write([]byte(email))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *Service) sendEmail(ctx context.Context, to, template, token string) {
	_ = ctx
	_ = to
	_ = template
	_ = token
}