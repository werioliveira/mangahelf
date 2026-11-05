package meilisearch

import (
	"context"

	"github.com/meilisearch/meilisearch-go"
	"github.com/mangashelf/mangashelf/internal/ports"
)

type Meilisearch struct {
	client interface{}
	index  interface{}
}

type Config struct {
	Host   string
	APIKey string
	Index  string
}

func New(cfg Config) (*Meilisearch, error) {
	client := meilisearch.New(cfg.Host, meilisearch.WithAPIKey(cfg.APIKey))

	index := client.Index(cfg.Index)

	return &Meilisearch{
		client: client,
		index:  index,
	}, nil
}

func (m *Meilisearch) Index(ctx context.Context, doc ports.Document) error {
	idx := m.index.(interface {
		AddDocuments(documents []any) (any, error)
	})
	_, err := idx.AddDocuments([]any{
		map[string]any{
			"id":     doc.ID,
			"title":  doc.ID,
			"body":   doc.Body,
		},
	})
	return err
}

func (m *Meilisearch) Delete(ctx context.Context, id string) error {
	idx := m.index.(interface {
		DeleteDocument(documentID string) (any, error)
	})
	_, err := idx.DeleteDocument(id)
	return err
}

func (m *Meilisearch) Search(ctx context.Context, q string, opts ports.SearchOpts) (ports.SearchResult, error) {
	if q == "" {
		return ports.SearchResult{}, nil
	}

	idx := m.index.(interface {
		Search(q string, request *meilisearch.SearchRequest) (*meilisearch.SearchResponse, error)
	})

	limit := int64(opts.Limit)
	offset := int64(opts.Offset)

	res, err := idx.Search(q, &meilisearch.SearchRequest{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return ports.SearchResult{}, err
	}

	var ids []string
	scores := make(map[string]float64)
	for _, hitRaw := range res.Hits {
		var hit map[string]any
		if err := hitRaw.DecodeInto(&hit); err != nil {
			continue
		}
		id, _ := hit["id"].(string)
		if id == "" {
			continue
		}
		ids = append(ids, id)
		if score, ok := hit["_rankingScore"].(float64); ok {
			scores[id] = score
		}
	}

	return ports.SearchResult{
		IDs:     ids,
		Total:   int(res.EstimatedTotalHits),
		Scores:  scores,
	}, nil
}

func (m *Meilisearch) Capabilities() []string {
	return []string{"fulltext", "typo_tolerance", "facets"}
}

var _ ports.SearchEngine = (*Meilisearch)(nil)
