package ports

import (
	"context"
	"io"
	"time"
)

type Op string

const (
	OpPut  Op = "put"
	OpGet  Op = "get"
)

// Storage abstrai armazenamento de objetos (local ou S3-compatível).
type Storage interface {
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	Get(ctx context.Context, key string) (io.ReadSeekCloser, error)
	Delete(ctx context.Context, key string) error
	Presign(ctx context.Context, key string, op Op, ttl time.Duration) (string, error)
	List(ctx context.Context, prefix string) ([]ObjectInfo, error)
}

type ObjectInfo struct {
	Key    string
	Size   int64
	ETag   string
	ModTime time.Time
}

// SearchEngine abstrai busca (Postgres FTS ou Meilisearch).
type SearchEngine interface {
	Index(ctx context.Context, doc Document) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, q string, opts SearchOpts) (SearchResult, error)
	Capabilities() []string
}

type Document struct {
	ID   string
	Body string
}

type SearchOpts struct {
	Query   string
	Limit   int
	Offset  int
	Filters map[string]string
}

type SearchResult struct {
	IDs     []string
	Total   int
	Scores  map[string]float64
}

// ImageURLBuilder monta a URL pública/assinada de uma página em tempo de resposta.
type ImageURLBuilder interface {
	Build(ctx context.Context, page Page, variant string) (string, error)
}

type Page struct {
	StorageKey string
	Width      int
	Height     int
	Variant    string
	Access     string
}

// Cache abstrai cache (memória ou Redis).
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// Authenticator abstrai autenticação (builtin ou OIDC).
type Authenticator interface {
	Authenticate(ctx context.Context, creds Credentials) (User, error)
	VerifyPassword(ctx context.Context, userID, password string) (bool, error)
}

type Credentials struct {
	Email    string
	Password string
}

type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	Roles        []string
	Groups       []string
	IsActive     bool
	EmailVerified bool
	OAuthProvider     string
	OAuthProviderID    string
	CreatedAt     string
	UpdatedAt     string
	LastLoginAt   *string
	FailedLoginAttempts int
	LockedUntil   *string
}

// Authorizer resolve permissões atômicas (sempre implementação própria).
type Authorizer interface {
	Can(ctx context.Context, u User, perm string, scope Scope) bool
}

type Scope struct {
	GroupID string // vazio = global
}

type AuthorizeFunc func(ctx context.Context, u User, perm string, scope Scope) error
