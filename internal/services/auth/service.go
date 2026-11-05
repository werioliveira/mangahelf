package service

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/mangashelf/mangashelf/internal/ports"
)

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateOIDCUser(ctx context.Context, email, name, provider, providerID string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, name, oauth_provider, oauth_provider_id, is_active, password_hash)
		 VALUES ($1, $2, $3, $4, true, '')
		 RETURNING id`,
		email, name, provider, providerID,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *Service) GetUserByID(ctx context.Context, userID string) (*ports.User, error) {
	var u ports.User
	var name sql.NullString
	var passwordHash sql.NullString
	var roles sql.NullString
	var groups sql.NullString

	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash, is_active, roles, groups, created_at, updated_at
		 FROM users WHERE id = $1`,
		userID,
	).Scan(&u.ID, &u.Email, &name, &passwordHash, &u.IsActive, &roles, &groups, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("usuário não encontrado")
		}
		return nil, err
	}

	if name.Valid {
		u.Name = name.String
	}
	if passwordHash.Valid {
		u.PasswordHash = passwordHash.String
	}
	if roles.Valid {
		u.Roles = []string{roles.String}
	}
	if groups.Valid {
		u.Groups = []string{groups.String}
	}

	return &u, nil
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*ports.User, error) {
	var u ports.User
	var name sql.NullString
	var passwordHash sql.NullString
	var roles sql.NullString
	var groups sql.NullString

	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash, is_active, roles, groups, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &name, &passwordHash, &u.IsActive, &roles, &groups, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("usuário não encontrado")
		}
		return nil, err
	}

	if name.Valid {
		u.Name = name.String
	}
	if passwordHash.Valid {
		u.PasswordHash = passwordHash.String
	}
	if roles.Valid {
		u.Roles = []string{roles.String}
	}
	if groups.Valid {
		u.Groups = []string{groups.String}
	}

	return &u, nil
}

// SaveSession salva a sessão do usuário (cookie httpOnly).
// Nota: esta é uma implementação simplificada. A sessão real é gerenciada
// pelo middleware de sessão do aplicativo principal.
func (s *Service) SaveSession(w http.ResponseWriter, r *http.Request, userID string) error {
	_ = w
	_ = r
	_ = userID
	return nil
}
