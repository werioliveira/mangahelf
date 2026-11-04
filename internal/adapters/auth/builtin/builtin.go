package builtin

import (
	"context"
	"errors"

	"github.com/mangashelf/mangashelf/internal/ports"
)

type Builtin struct{}

func New() *Builtin { return &Builtin{} }

func (b *Builtin) Authenticate(ctx context.Context, creds ports.Credentials) (ports.User, error) {
	return ports.User{}, errors.New("builtin auth not yet implemented")
}
func (b *Builtin) VerifyPassword(ctx context.Context, userID, password string) (bool, error) {
	return false, errors.New("builtin auth not yet implemented")
}

var _ ports.Authenticator = (*Builtin)(nil)
