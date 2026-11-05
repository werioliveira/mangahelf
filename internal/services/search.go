package services

import (
	"context"
	"database/sql"
	"errors"

	"github.com/mangashelf/mangashelf/internal/ports"
)

type SearchService struct {
	db          *sql.DB
	searchEngine ports.SearchEngine
}

type SearchResult struct {
	IDs     []string `json:"ids"`
	Total   int      `json:"total"`
	Scores  map[string]float64 `json:"scores"`
}

type SearchInput struct {
	Query      string `json:"query"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	PublicOnly bool   `json:"public_only"`
}

func NewSearchService(db *sql.DB, searchEngine ports.SearchEngine) *SearchService {
	return &SearchService{
		db:          db,
		searchEngine: searchEngine,
	}
}

// Search executa busca usando o motor configurado (Postgres FTS ou Meilisearch).
func (s *SearchService) Search(ctx context.Context, in SearchInput) (SearchResult, error) {
	if in.Query == "" {
		return SearchResult{}, nil
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

	result, err := s.searchEngine.Search(ctx, in.Query, ports.SearchOpts{
		Query:   in.Query,
		Limit:   in.Limit,
		Offset:  in.Offset,
		Filters: nil,
	})
	if err != nil {
		return SearchResult{}, err
	}

	return SearchResult{
		IDs:     result.IDs,
		Total:   result.Total,
		Scores:  result.Scores,
	}, nil
}

// IndexDocument indexa um documento no motor de busca.
func (s *SearchService) IndexDocument(ctx context.Context, id, title, body string) error {
	if s.searchEngine == nil {
		return errors.New("search engine não configurado")
	}
	return s.searchEngine.Index(ctx, ports.Document{
		ID:   id,
		Body: title + " " + body,
	})
}

// DeleteDocument remove um documento do motor de busca.
func (s *SearchService) DeleteDocument(ctx context.Context, id string) error {
	if s.searchEngine == nil {
		return errors.New("search engine não configurado")
	}
	return s.searchEngine.Delete(ctx, id)
}

var _ = errors.New
