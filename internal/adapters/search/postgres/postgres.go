package postgres

import (
	"context"
	"database/sql"
	"strings"

	"github.com/mangashelf/mangashelf/internal/ports"
)

type Postgres struct {
	db *sql.DB
}

func New(db *sql.DB) *Postgres { return &Postgres{db: db} }

func (p *Postgres) Index(ctx context.Context, doc ports.Document) error {
	_, err := p.db.ExecContext(ctx,
		`UPDATE series SET search_vector = setweight(to_tsvector('portuguese', coalesce(title, '')), 'A') ||
		                              setweight(to_tsvector('portuguese', coalesce(synopsis, '')), 'B')
		 WHERE id = $1 OR slug = $2`,
		doc.ID, doc.ID,
	)
	return err
}

func (p *Postgres) Delete(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx,
		`UPDATE series SET search_vector = setweight(to_tsvector('portuguese', ''), 'A') ||
		                              setweight(to_tsvector('portuguese', ''), 'B')
		 WHERE id = $1 OR slug = $2`,
		id, id,
	)
	return err
}

func (p *Postgres) Search(ctx context.Context, q string, opts ports.SearchOpts) (ports.SearchResult, error) {
	if q == "" {
		return ports.SearchResult{}, nil
	}

	// Construir tsquery: juntar termos com ' | ' (OR) e adicionar ':*' para prefix match
	terms := strings.Fields(q)
	if len(terms) == 0 {
		return ports.SearchResult{}, nil
	}
	query := strings.Join(terms, " | ") + ":*"

	rows, err := p.db.QueryContext(ctx,
		`SELECT s.id, ts_rank(s.search_vector, to_tsquery('portuguese', $1)) as rank
		 FROM series s
		 WHERE s.search_vector @@ to_tsquery('portuguese', $1)
		 ORDER BY rank DESC, s.created_at DESC
		 LIMIT $2 OFFSET $3`,
		query, opts.Limit, opts.Offset,
	)
	if err != nil {
		return ports.SearchResult{}, err
	}
	defer rows.Close()

	var result ports.SearchResult
	var ids []string
	scores := make(map[string]float64)
	for rows.Next() {
		var id string
		var rank float64
		if err := rows.Scan(&id, &rank); err != nil {
			return ports.SearchResult{}, err
		}
		ids = append(ids, id)
		scores[id] = rank
	}
	if err := rows.Err(); err != nil {
		return ports.SearchResult{}, err
	}

	result.IDs = ids
	result.Total = len(ids)
	result.Scores = scores
	return result, nil
}

func (p *Postgres) Capabilities() []string {
	return []string{"fts", "prefix_match"}
}

var _ ports.SearchEngine = (*Postgres)(nil)
