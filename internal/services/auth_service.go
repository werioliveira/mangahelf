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

	"github.com/mangashelf/mangashelf/internal/ports"
	"github.com/alexedwards/scs/v2"
	"golang.org/x/crypto/argon2"
)

type AuthService struct {
	db *sql.DB
	sm *scs.SessionManager
}

func NewAuthService(db *sql.DB, sm *scs.SessionManager) *AuthService {
	return &AuthService{db: db, sm: sm}
}

// HashPassword gera um hash argon2id da senha
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 3, 32*1024, 4, 32)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		32*1024, 3, 4,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword verifica se a senha corresponde ao hash
func VerifyPassword(password, hash string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false
	}

	var memory, timeParam, threads int
	fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeParam, &threads)

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	computedHash := argon2.IDKey([]byte(password), salt, uint32(timeParam), uint32(memory), uint8(threads), uint32(len(expectedHash)))
	return subtleConstantTimeCompare(computedHash, expectedHash)
}

// Login autentica um usuário por email e senha
func (s *AuthService) Login(ctx context.Context, email, password string, ipAddress, userAgent string) (*ports.User, error) {
	var userID string
	var passwordHash sql.NullString
	var status string

	err := s.db.QueryRowContext(ctx,
		`SELECT id, password_hash, status FROM users WHERE email = $1`,
		email,
	).Scan(&userID, &passwordHash, &status)

	if err == sql.ErrNoRows {
		s.recordLoginAttempt(ctx, email, ipAddress, userAgent, false, "usuario nao encontrado")
		return nil, errors.New("email ou senha invalidos")
	}
	if err != nil {
		return nil, err
	}

	// account lock check removed (no locked_until column)

	if status != "active" {
		return nil, errors.New("usuario inativo")
	}

	if passwordHash.Valid && !VerifyPassword(password, passwordHash.String) {
		s.incrementFailedLogin(ctx, email)
		s.recordLoginAttempt(ctx, email, ipAddress, userAgent, false, "senha invalida")
		return nil, errors.New("email ou senha invalidos")
	}

	s.resetFailedLogin(ctx, email)
	s.db.ExecContext(ctx, "UPDATE users SET last_login_at = now(), failed_login_attempts = 0 WHERE id = $1", userID)
	s.recordLoginAttempt(ctx, email, ipAddress, userAgent, true, "sucesso")

	return s.GetUserByID(ctx, userID)
}

// Register cria um novo usuário com senha
func (s *AuthService) Register(ctx context.Context, email, name, password string) (*ports.User, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)",
		email,
	).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("email ja cadastrado")
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	var id string
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, name, password_hash, status, email_verified_at)
		 VALUES ($1, $2, $3, 'active', NULL)
		 RETURNING id`,
		email, name, passwordHash,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	var userRoleID string
	err = s.db.QueryRowContext(ctx,
		"SELECT id FROM roles WHERE name = 'user'",
	).Scan(&userRoleID)
	if err == nil {
		s.db.ExecContext(ctx,
			`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`,
			id, userRoleID,
		)
	}

	return s.GetUserByID(ctx, id)
}

// GetUserByID busca um usuário por ID
func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*ports.User, error) {
	var u ports.User
	var passwordHash sql.NullString
	var status string

	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, username, password_hash, status FROM users WHERE id = $1`,
		userID,
	).Scan(&u.ID, &u.Email, &u.Name, &passwordHash, &status)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("usuario nao encontrado")
		}
		return nil, err
	}

	if passwordHash.Valid {
		u.PasswordHash = passwordHash.String
	}
	u.IsActive = (status == "active")

	return &u, nil
}

// GetUserByEmail busca um usuário por email
func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*ports.User, error) {
	var u ports.User
	var name sql.NullString
	var passwordHash sql.NullString
	var status string

	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, username, password_hash, status, oauth_provider,
		 oauth_provider_id, created_at, updated_at, last_login_at,
		 failed_login_attempts, locked_until
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &name, &passwordHash, &status, &u.OAuthProvider,
		&u.OAuthProviderID, &u.CreatedAt, &u.UpdatedAt,
		&u.LastLoginAt, &u.FailedLoginAttempts, &u.LockedUntil)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("usuario nao encontrado")
		}
		return nil, err
	}

	if name.Valid {
		u.Name = name.String
	}
	if passwordHash.Valid {
		u.PasswordHash = passwordHash.String
	}
	u.IsActive = (status == "active")

	return &u, nil
}

// HasPermission verifica se o usuário tem uma permissão específica
func (s *AuthService) HasPermission(ctx context.Context, userID string, permission string) (bool, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return false, err
	}

	if !user.IsActive {
		return false, nil
	}

	rolePermissions := map[string][]string{
		"admin": {
			"admin.access",
			"series.manage",
			"chapters.manage",
			"users.manage",
			"settings.manage",
			"vip.manage",
			"search.manage",
		},
		"editor": {
			"series.manage",
			"chapters.manage",
			"vip.manage",
		},
		"moderator": {
			"chapters.manage",
			"vip.manage",
		},
		"user": {},
	}

	for _, role := range user.Roles {
		perms, ok := rolePermissions[role]
		if !ok {
			continue
		}
		for _, p := range perms {
			if p == permission {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *AuthService) recordLoginAttempt(ctx context.Context, email, ipAddress, userAgent string, success bool, reason string) {
	s.db.ExecContext(ctx,
		`INSERT INTO login_attempts (email, ip_address, user_agent, success, reason)
		 VALUES ($1, $2, $3, $4, $5)`,
		email, ipAddress, userAgent, success, reason,
	)
}

func (s *AuthService) incrementFailedLogin(ctx context.Context, email string) {
	s.db.ExecContext(ctx,
		`UPDATE users SET failed_login_attempts = failed_login_attempts + 1
		 WHERE email = $1`,
		email,
	)
}

func (s *AuthService) resetFailedLogin(ctx context.Context, email string) {
	s.db.ExecContext(ctx,
		`UPDATE users SET failed_login_attempts = 0
		 WHERE email = $1`,
		email,
	)
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

// CreateOIDCUser cria um usuário a partir de autenticação OAuth/OIDC
func (s *AuthService) CreateOIDCUser(ctx context.Context, email, name, provider, providerID string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, name, oauth_provider, oauth_provider_id, status, email_verified_at)
		 VALUES ($1, $2, $3, $4, 'active', now())
		 RETURNING id`,
		email, name, provider, providerID,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	var userRoleID string
	err = s.db.QueryRowContext(ctx,
		"SELECT id FROM roles WHERE name = 'user'",
	).Scan(&userRoleID)
	if err == nil {
		s.db.ExecContext(ctx,
			`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`,
			id, userRoleID,
		)
	}

	return id, nil
}

// SaveSession salva a sessão do usuário (cookie httpOnly).
func (s *AuthService) SaveSession(w http.ResponseWriter, r *http.Request, userID string) error {
	s.sm.Put(r.Context(), "uid", userID)
	return nil
}

// UserIDFromSession extrai o ID do usuário da sessão.
func (s *AuthService) UserIDFromSession(r *http.Request) (string, bool) {
	uid := s.sm.GetString(r.Context(), "uid")
	return uid, uid != ""
}

// ClearSession limpa a sessão do usuário.
func (s *AuthService) ClearSession(w http.ResponseWriter, r *http.Request) error {
	s.sm.Destroy(r.Context())
	return nil
}
