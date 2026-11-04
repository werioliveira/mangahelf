package postgres

import (
	"context"

	"github.com/mangashelf/mangashelf/internal/ports"
)

type Postgres struct{}

func New() *Postgres { return &Postgres{} }

func (p *Postgres) Index(ctx context.Context, doc ports.Document) error      { return errNotImplemented }
func (p *Postgres) Delete(ctx context.Context, id string) error              { return errNotImplemented }
func (p *Postgres) Search(ctx context.Context, q string, opts ports.SearchOpts) (ports.SearchResult, error) { return ports.SearchResult{}, errNotImplemented }
func (p *Postgres) Capabilities() []string                                   { return []string{"fts"} }

var _ ports.SearchEngine = (*Postgres)(nil)

var errNotImplemented = &errImpl{msg: "postgres search adapter not yet implemented"}

type errImpl struct{ msg string }

func (e *errImpl) Error() string { return e.msg }
