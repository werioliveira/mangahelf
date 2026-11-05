package oidc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	authService "github.com/mangashelf/mangashelf/internal/services/auth"
	"github.com/mangashelf/mangashelf/internal/ports"
)

type AuthService struct {
	db          *sql.DB
	authService *authService.Service
}

func NewAuthService(db *sql.DB, authService *authService.Service) *AuthService {
	return &AuthService{
		db:          db,
		authService: authService,
	}
}

func (s *AuthService) FindOrCreateOIDCUser(ctx context.Context, provider string, userInfo map[string]any) (*ports.User, error) {
	var email string
	var name string
	var providerID string

	switch provider {
	case "google":
		if v, ok := userInfo["email"].(string); ok {
			email = v
		}
		if v, ok := userInfo["name"].(string); ok {
			name = v
		}
		if v, ok := userInfo["id"].(string); ok {
			providerID = v
		}
	case "github":
		if v, ok := userInfo["email"].(string); ok {
			email = v
		} else if v, ok := userInfo["login"].(string); ok {
			email = v + "@github.com"
		}
		if v, ok := userInfo["name"].(string); ok {
			name = v
		} else if v, ok := userInfo["login"].(string); ok {
			name = v
		}
		if v, ok := userInfo["id"].(float64); ok {
			providerID = fmt.Sprintf("%d", int(v))
		} else if v, ok := userInfo["id"].(string); ok {
			providerID = v
		}
	default:
		if v, ok := userInfo["email"].(string); ok {
			email = v
		}
		if v, ok := userInfo["name"].(string); ok {
			name = v
		}
		if v, ok := userInfo["id"].(string); ok {
			providerID = v
		}
	}

	if email == "" {
		return nil, errors.New("e-mail não disponível no provedor")
	}

	var existingUserID string
	err := s.db.QueryRowContext(ctx,
		"SELECT id FROM users WHERE email = $1",
		email,
	).Scan(&existingUserID)

	if err == nil {
		_, err = s.db.ExecContext(ctx,
			`UPDATE users SET oauth_provider = $1, oauth_provider_id = $2, updated_at = now()
			 WHERE id = $3`,
			provider, providerID, existingUserID,
		)
		if err != nil {
			return nil, err
		}
		return s.authService.GetUserByID(ctx, existingUserID)
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	userID, err := s.authService.CreateOIDCUser(ctx, email, name, provider, providerID)
	if err != nil {
		return nil, err
	}

	return s.authService.GetUserByID(ctx, userID)
}
