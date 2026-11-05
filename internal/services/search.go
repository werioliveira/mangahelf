package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type SearchResult struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Slug      string `json:"slug"`
	Synopsis  string `json:"synopsis"`
	Access    string `json:"access"`
	GroupID   string `json:"group_id,omitempty"`
	Cover     string `json:"cover,omitempty"`
	Rank      float64 `json:"rank"`
	Chapters  int    `json:"chapters"`
}

type SearchInput struct {
	Query   string `json:"query"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
	PublicOnly bool `json:"public_only"`
}

type SearchService struct {
	db *sql.DB
}

func NewSearchService(db *sql.DB) *SearchService {
	return &SearchService{db: db}
}

// Search executa busca full-text em séries usando tsvector + tsquery com ranking por relevância.
func (s *SearchService) Search(ctx context.Context, in SearchInput) ([]SearchResult, error) {
	if in.Query == "" {
		return nil, errors.New("query é obrigatória")
	}
	if in.Limit <= 0 {
		in.Limit = 20
	}
	if in.Limit > 100 {
		in.Limit = 100
	}
	if in.Offset < 0 {
		in.Offset = 0
	}

	// Converter a query do usuário para tsquery: juntar termos com ' | ' (OR)
	// e adicionar ':*' para prefix match em cada termo
	terms := strings.Fields(in.Query)
	if len(terms) == 0 {
		return nil, errors.New("query inválida")
	}
	query := strings.Join(terms, " | ") + ":*"

	// Construir a query SQL com filtro de acesso opcional
	baseQuery := `
		SELECT s.id, s.title, s.slug, s.synopsis, s.access, s.group_id,
		       ts_rank(s.search_vector, to_tsquery('portuguese', $1)) as rank
		FROM series s
		WHERE s.search_vector @@ to_tsquery('portuguese', $1)
	`
	args := []any{query}
	argIdx := 2

	if in.PublicOnly {
		baseQuery += fmt.Sprintf(" AND s.access = $%d", argIdx)
		args = append(args, "public")
		argIdx++
	}

	baseQuery += fmt.Sprintf(" ORDER BY rank DESC, s.created_at DESC")
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, in.Limit, in.Offset)

	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SearchResult
	for rows.Next() {
		var r SearchResult
		var groupID sql.NullString
		if err := rows.Scan(&r.ID, &r.Title, &r.Slug, &r.Synopsis, &r.Access, &groupID, &r.Rank); err != nil {
			return nil, err
		}
		if groupID.Valid {
			r.GroupID = groupID.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

var _ = fmt.Sprintf
