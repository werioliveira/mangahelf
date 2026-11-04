package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Chapter struct {
	ID        string    `json:"id"`
	SeriesID  string    `json:"series_id"`
	Number    int       `json:"number"`
	Title     string    `json:"title,omitempty"`
	State     string    `json:"state"`
	Access    string    `json:"access"`
	PublicAt  *time.Time `json:"public_at,omitempty"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

type CreateChapterInput struct {
	SeriesID string  `json:"series_id"`
	Number   int     `json:"number"`
	Title    string  `json:"title,omitempty"`
	State    string  `json:"state"`
	Access   string  `json:"access"`
	PublicAt *string `json:"public_at,omitempty"`
}

type UpdateChapterInput struct {
	Title    *string `json:"title,omitempty"`
	State    *string `json:"state,omitempty"`
	Access   *string `json:"access,omitempty"`
	PublicAt *string `json:"public_at,omitempty"`
}

type ChapterService struct {
	db *sql.DB
}

func NewChapterService(db *sql.DB) *ChapterService {
	return &ChapterService{db: db}
}

func parseTimePtr(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *ChapterService) Create(ctx context.Context, in CreateChapterInput) (Chapter, error) {
	if in.SeriesID == "" {
		return Chapter{}, errors.New("series_id e obrigatorio")
	}
	if in.Number <= 0 {
		return Chapter{}, errors.New("numero deve ser maior que zero")
	}
	if in.State == "" {
		in.State = "draft"
	}
	if in.Access == "" {
		in.Access = "public"
	}
	var pubAt *time.Time
	if in.PublicAt != nil && *in.PublicAt != "" {
		pt, err := parseTimePtr(in.PublicAt)
		if err != nil {
			return Chapter{}, fmt.Errorf("public_at invalido: %w", err)
		}
		pubAt = pt
	}
	var id string
	var publicAtInterface any = pubAt
	err := s.db.QueryRowContext(ctx,
		"INSERT INTO chapters (series_id, number, title, state, access, public_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, series_id, number, title, state, access, public_at, created_at, updated_at",
		in.SeriesID, in.Number, in.Title, in.State, in.Access, publicAtInterface,
	).Scan(&id, &in.SeriesID, &in.Number, &in.Title, &in.State, &in.Access, &publicAtInterface, new(string), new(string))
	if err != nil {
		return Chapter{}, err
	}
	ch := Chapter{ID: id, SeriesID: in.SeriesID, Number: in.Number, Title: in.Title, State: in.State, Access: in.Access}
	if pubAt != nil {
		ch.PublicAt = pubAt
	}
	return ch, nil
}

func (s *ChapterService) List(ctx context.Context) ([]Chapter, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, series_id, number, title, state, access, public_at, created_at, updated_at FROM chapters ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Chapter
	for rows.Next() {
		var ch Chapter
		var pubAt sql.NullTime
		if err := rows.Scan(&ch.ID, &ch.SeriesID, &ch.Number, &ch.Title, &ch.State, &ch.Access, &pubAt, new(string), new(string)); err != nil {
			return nil, err
		}
		if pubAt.Valid {
			ch.PublicAt = &pubAt.Time
		}
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (s *ChapterService) Get(ctx context.Context, id string) (Chapter, error) {
	var ch Chapter
	var pubAt sql.NullTime
	err := s.db.QueryRowContext(ctx, "SELECT id, series_id, number, title, state, access, public_at, created_at, updated_at FROM chapters WHERE id=$1", id).Scan(&ch.ID, &ch.SeriesID, &ch.Number, &ch.Title, &ch.State, &ch.Access, &pubAt, new(string), new(string))
	if err != nil {
		return Chapter{}, err
	}
	if pubAt.Valid {
		ch.PublicAt = &pubAt.Time
	}
	return ch, nil
}

func (s *ChapterService) Update(ctx context.Context, id string, in UpdateChapterInput) (Chapter, error) {
	ch, err := s.Get(ctx, id)
	if err != nil {
		return Chapter{}, err
	}
	if in.Title != nil {
		ch.Title = *in.Title
	}
	if in.State != nil {
		ch.State = *in.State
	}
	if in.Access != nil {
		ch.Access = *in.Access
	}
	if in.PublicAt != nil {
		pt, err := parseTimePtr(in.PublicAt)
		if err != nil {
			return Chapter{}, fmt.Errorf("public_at invalido: %w", err)
		}
		ch.PublicAt = pt
	}
	var pubAtInterface any
	if ch.PublicAt != nil {
		pubAtInterface = *ch.PublicAt
	}
	_, err = s.db.ExecContext(ctx, "UPDATE chapters SET title=$1, state=$2, access=$3, public_at=$4, updated_at=now() WHERE id=$5", ch.Title, ch.State, ch.Access, pubAtInterface, id)
	return ch, err
}

func (s *ChapterService) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM chapters WHERE id=$1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("capitulo nao encontrado: %s", id)
	}
	return nil
}


