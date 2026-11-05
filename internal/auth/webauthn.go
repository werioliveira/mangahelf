package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

var (
)

type WebAuthnService struct {
	wa    *webauthn.WebAuthn
	q     *Queries
	cfg   Config
}

func NewWebAuthnService(cfg Config, q *Queries) (*WebAuthnService, error) {
	if cfg.RPID == "" || len(cfg.RPOrigins) == 0 {
		return nil, ErrWebAuthnNotConfigured
	}

	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "MangaShelf",
		RPID:          cfg.RPID,
		RPOrigins:     cfg.RPOrigins,
	})
	if err != nil {
		return nil, err
	}

	return &WebAuthnService{wa: wa, q: q, cfg: cfg}, nil
}

func (s *WebAuthnService) BeginRegistration(ctx context.Context, userID uuid.UUID, username string) (webauthn.SessionData, error) {
	user := &webauthnUser{id: userID, name: username}
	_, sessionData, err := s.wa.BeginRegistration(user)
	if err != nil {
		return webauthn.SessionData{}, err
	}
	return *sessionData, nil
}

func (s *WebAuthnService) FinishRegistration(ctx context.Context, userID uuid.UUID, sessionData webauthn.SessionData, r *http.Request) error {
	user := &webauthnUser{id: userID, name: ""}
	credential, err := s.wa.FinishRegistration(user, sessionData, r)
	if err != nil {
		return err
	}

	err = s.q.ExecContext(ctx,
		`INSERT INTO user_webauthn_credentials (id, user_id, public_key, sign_count, transports, aaguid, name, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, now())`,
		credential.ID, userID, credential.PublicKey, credential.Authenticator.SignCount,
		credential.Transport, credential.Authenticator.AAGUID,
		"Passkey")
	return err
}

func (s *WebAuthnService) BeginLogin(ctx context.Context) (webauthn.SessionData, error) {
	_, sessionData, err := s.wa.BeginDiscoverableLogin()
	if err != nil {
		return webauthn.SessionData{}, err
	}
	return *sessionData, nil
}

func (s *WebAuthnService) FinishLogin(ctx context.Context, sessionData webauthn.SessionData, r *http.Request) (uuid.UUID, error) {
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		userID, err := s.q.GetUserByCredentialID(ctx, rawID)
		if err != nil {
			return nil, err
		}
		return &webauthnUser{id: userID, name: ""}, nil
	}

	credential, err := s.wa.FinishDiscoverableLogin(handler, sessionData, r)
	if err != nil {
		return uuid.Nil, err
	}

	userID, err := s.q.GetUserByCredentialID(ctx, credential.ID)
	if err != nil {
		return uuid.Nil, err
	}

	if credential.Authenticator.SignCount <= 0 {
		return userID, nil
	}

	storedCount, err := s.q.GetCredentialSignCount(ctx, credential.ID)
	if err != nil {
		return uuid.Nil, err
	}

	if credential.Authenticator.SignCount <= storedCount {
		return uuid.Nil, ErrSignCountInvalid
	}

	if err := s.q.UpdateCredentialSignCount(ctx, credential.ID, credential.Authenticator.SignCount); err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func (s *WebAuthnService) ListCredentials(ctx context.Context, userID uuid.UUID) ([]CredentialInfo, error) {
	return s.q.ListCredentials(ctx, userID)
}

func (s *WebAuthnService) RemoveCredential(ctx context.Context, userID uuid.UUID, credentialID []byte) error {
	return s.q.DeleteCredential(ctx, userID, credentialID)
}

type webauthnUser struct {
	id   uuid.UUID
	name string
}

func (u *webauthnUser) WebAuthnID() []byte       { return u.id[:] }
func (u *webauthnUser) WebAuthnName() string     { return u.name }
func (u *webauthnUser) WebAuthnDisplayName() string { return u.name }
func (u *webauthnUser) WebAuthnIcon() string     { return "" }
func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential { return nil }

type CredentialInfo struct {
	ID              []byte
	Name            string
	Transports      []string
	BackupEligible  bool
	BackupState     bool
	LastUsedAt      *time.Time
	CreatedAt       time.Time
}