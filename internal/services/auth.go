package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/mangashelf/mangashelf/internal/config"
	"github.com/mangashelf/mangashelf/internal/ports"
	"golang.org/x/crypto/argon2"
)

type AuthService struct {
	db         *sql.DB
	sessionKey []byte
	store      *sessions.CookieStore
}

const (
	argonTime    = 3
	argonMemory  = 32 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

func NewAuthService(db *sql.DB, cfg *config.Config) *AuthService {
	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   cfg.Env == "production",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	}
	return &AuthService{db: db, sessionKey: []byte(cfg.SessionSecret), store: store}
}

func (a *AuthService) SessionStore() *sessions.CookieStore { return a.store }

func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	encoded := fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

func (a *AuthService) verifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false
	}
	var memory, timeParam, threads int
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeParam, &threads); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	hash := argon2.IDKey([]byte(password), salt, uint32(timeParam), uint32(memory), uint8(threads), uint32(len(expectedHash)))
	return subtleConstantTimeCompare(hash, expectedHash)
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

type RegisterInput struct {
	Email    string
	Password string
}

func (a *AuthService) Register(ctx context.Context, in RegisterInput) (ports.User, error) {
	var exists bool
	err := a.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", in.Email).Scan(&exists)
	if err != nil {
		return ports.User{}, err
	}
	if exists {
		return ports.User{}, errors.New("email ja cadastrado")
	}
	hashed, err := hashPassword(in.Password)
	if err != nil {
		return ports.User{}, err
	}
	var u ports.User
	err = a.db.QueryRowContext(ctx,
		"INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, password_hash, is_active",
		in.Email, hashed,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsActive)
	return u, err
}

func (a *AuthService) Login(ctx context.Context, email, password string) (ports.User, error) {
	var u ports.User
	err := a.db.QueryRowContext(ctx,
		"SELECT id, email, password_hash, is_active FROM users WHERE email=$1",
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return ports.User{}, errors.New("credenciais invalidas")
		}
		return ports.User{}, err
	}
	if !u.IsActive {
		return ports.User{}, errors.New("usuario inativo")
	}
	if !a.verifyPassword(u.PasswordHash, password) {
		return ports.User{}, errors.New("credenciais invalidas")
	}
	return u, nil
}

func (a *AuthService) SaveSession(w http.ResponseWriter, r *http.Request, userID string) error {
	session, err := a.store.New(r, "mangashelf-session")
	if err != nil {
		return err
	}
	session.Values["user_id"] = userID
	return session.Save(r, w)
}

func (a *AuthService) ClearSession(w http.ResponseWriter, r *http.Request) error {
	session, err := a.store.New(r, "mangashelf-session")
	if err != nil {
		return err
	}
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

func (a *AuthService) UserIDFromSession(r *http.Request) (string, bool) {
	session, err := a.store.Get(r, "mangashelf-session")
	if err != nil {
		return "", false
	}
	v, ok := session.Values["user_id"]
	if !ok {
		return "", false
	}
	id, ok := v.(string)
	return id, ok
}
