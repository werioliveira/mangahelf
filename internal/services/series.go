package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mangashelf/mangashelf/internal/ports"
)

type Series struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Slug             string `json:"slug"`
	Synopsis         string `json:"synopsis"`
	Type             string `json:"type"`
	Status           string `json:"status"`
	Demography       string `json:"demography"`
	OriginalLanguage string `json:"original_language"`
	ReleaseYear      *int   `json:"release_year"`
	Access           string `json:"access"`
	PublicationStatus string `json:"publication_status"`
	PublishedAt      string `json:"published_at,omitempty"`
	PublicAt         string `json:"public_at,omitempty"`
	CoverStorageKey  string `json:"cover_storage_key,omitempty"`
	GroupID          string `json:"group_id,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type CreateSeriesInput struct {
	Title            string  `json:"title"`
	Slug             string  `json:"slug"`
	Synopsis         string  `json:"synopsis"`
	Type             string  `json:"type"`
	Status           string  `json:"status"`
	Demography       string  `json:"demography"`
	OriginalLanguage string  `json:"original_language"`
	ReleaseYear      *int    `json:"release_year"`
	Access           string  `json:"access"`
	CoverStorageKey  string  `json:"cover_storage_key"`
	GroupID          string  `json:"group_id"`
	PublicAt         *string `json:"public_at"`
}

type UpdateSeriesInput struct {
	Title            *string `json:"title,omitempty"`
	Slug             *string `json:"slug,omitempty"`
	Synopsis         *string `json:"synopsis,omitempty"`
	Type             *string `json:"type,omitempty"`
	Status           *string `json:"status,omitempty"`
	Demography       *string `json:"demography,omitempty"`
	OriginalLanguage *string `json:"original_language,omitempty"`
	ReleaseYear      *int    `json:"release_year,omitempty"`
	Access           *string `json:"access,omitempty"`
	CoverStorageKey  *string `json:"cover_storage_key,omitempty"`
	GroupID          *string `json:"group_id,omitempty"`
	PublicAt         *string `json:"public_at,omitempty"`
}

type SeriesService struct {
	db *sql.DB
}

func NewSeriesService(db *sql.DB) *SeriesService {
	return &SeriesService{db: db}
}

func (s *SeriesService) Create(ctx context.Context, in CreateSeriesInput) (Series, error) {
	if in.Title == "" {
		return Series{}, errors.New("titulo e obrigatorio")
	}
	if in.Slug == "" {
		return Series{}, errors.New("slug e obrigatorio")
	}
	if in.Type == "" {
		in.Type = "manga"
	}
	if in.Status == "" {
		in.Status = "ongoing"
	}
	if in.Demography == "" {
		in.Demography = "unspecified"
	}
	if in.OriginalLanguage == "" {
		in.OriginalLanguage = "ja"
	}
	if in.Access == "" {
		in.Access = "public"
	}
	var gid any
	if in.GroupID != "" {
		gid = in.GroupID
	}
	var pubAt any
	if in.PublicAt != nil && *in.PublicAt != "" {
		pubAt = *in.PublicAt
	}
	var coverKey any
	if in.CoverStorageKey != "" {
		coverKey = in.CoverStorageKey
	}
	var id string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO series (title, slug, synopsis, type, status, demography, original_language, release_year, access, cover_storage_key, group_id, public_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id, title, slug, synopsis, type, status, demography, original_language, release_year, access, publication_status, published_at, public_at, cover_storage_key, group_id, created_at, updated_at`,
		in.Title, in.Slug, in.Synopsis, in.Type, in.Status, in.Demography, in.OriginalLanguage, in.ReleaseYear, in.Access, coverKey, gid, pubAt,
	).Scan(&id, &in.Title, &in.Slug, &in.Synopsis, &in.Type, &in.Status, &in.Demography, &in.OriginalLanguage, &in.ReleaseYear, &in.Access, new(string), new(sql.NullTime), new(sql.NullTime), &coverKey, &gid, new(string), new(string))
	if err != nil {
		return Series{}, err
	}
	st := Series{ID: id, Title: in.Title, Slug: in.Slug, Synopsis: in.Synopsis, Type: in.Type, Status: in.Status, Demography: in.Demography, OriginalLanguage: in.OriginalLanguage, ReleaseYear: in.ReleaseYear, Access: in.Access}
	if coverKey != nil {
		st.CoverStorageKey = coverKey.(string)
	}
	if gid != nil {
		st.GroupID = gid.(string)
	}
	return st, nil
}

func (s *SeriesService) List(ctx context.Context) ([]Series, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, slug, synopsis, type, status, demography, original_language, release_year, access, publication_status, published_at, public_at, cover_storage_key, group_id, created_at, updated_at FROM series ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Series
	for rows.Next() {
		var st Series
		var gid, coverKey sql.NullString
		var pubAt, publishedAt sql.NullTime
		var releaseYear sql.NullInt32
		if err := rows.Scan(&st.ID, &st.Title, &st.Slug, &st.Synopsis, &st.Type, &st.Status, &st.Demography, &st.OriginalLanguage, &releaseYear, &st.Access, &st.PublicationStatus, &publishedAt, &pubAt, &coverKey, &gid, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, err
		}
		if releaseYear.Valid {
			v := int(releaseYear.Int32)
			st.ReleaseYear = &v
		}
		if pubAt.Valid {
			st.PublicAt = pubAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}
		if publishedAt.Valid {
			st.PublishedAt = publishedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}
		if coverKey.Valid {
			st.CoverStorageKey = coverKey.String
		}
		if gid.Valid {
			st.GroupID = gid.String
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func (s *SeriesService) Get(ctx context.Context, id string) (Series, error) {
	var st Series
	var gid, coverKey sql.NullString
	var pubAt, publishedAt sql.NullTime
	var releaseYear sql.NullInt32
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, slug, synopsis, type, status, demography, original_language, release_year, access, publication_status, published_at, public_at, cover_storage_key, group_id, created_at, updated_at FROM series WHERE id=$1`,
		id,
	).Scan(&st.ID, &st.Title, &st.Slug, &st.Synopsis, &st.Type, &st.Status, &st.Demography, &st.OriginalLanguage, &releaseYear, &st.Access, &st.PublicationStatus, &publishedAt, &pubAt, &coverKey, &gid, &st.CreatedAt, &st.UpdatedAt)
	if err != nil {
		return Series{}, err
	}
	if releaseYear.Valid {
		v := int(releaseYear.Int32)
		st.ReleaseYear = &v
	}
	if pubAt.Valid {
		st.PublicAt = pubAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}
	if publishedAt.Valid {
		st.PublishedAt = publishedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}
	if coverKey.Valid {
		st.CoverStorageKey = coverKey.String
	}
	if gid.Valid {
		st.GroupID = gid.String
	}
	return st, nil
}

func (s *SeriesService) Update(ctx context.Context, id string, in UpdateSeriesInput) (Series, error) {
	st, err := s.Get(ctx, id)
	if err != nil {
		return Series{}, err
	}
	if in.Title != nil {
		st.Title = *in.Title
	}
	if in.Slug != nil {
		st.Slug = *in.Slug
	}
	if in.Synopsis != nil {
		st.Synopsis = *in.Synopsis
	}
	if in.Type != nil {
		st.Type = *in.Type
	}
	if in.Status != nil {
		st.Status = *in.Status
	}
	if in.Demography != nil {
		st.Demography = *in.Demography
	}
	if in.OriginalLanguage != nil {
		st.OriginalLanguage = *in.OriginalLanguage
	}
	if in.ReleaseYear != nil {
		st.ReleaseYear = in.ReleaseYear
	}
	if in.Access != nil {
		st.Access = *in.Access
	}
	if in.CoverStorageKey != nil {
		st.CoverStorageKey = *in.CoverStorageKey
	}
	if in.GroupID != nil {
		st.GroupID = *in.GroupID
	}
	if in.PublicAt != nil {
		if *in.PublicAt == "" {
			st.PublicAt = ""
		} else {
			st.PublicAt = *in.PublicAt
		}
	}
	var gid any
	if st.GroupID != "" {
		gid = st.GroupID
	}
	var coverKey any
	if st.CoverStorageKey != "" {
		coverKey = st.CoverStorageKey
	}
	var pubAt any
	if st.PublicAt != "" {
		pubAt = st.PublicAt
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE series SET title=$1, slug=$2, synopsis=$3, type=$4, status=$5, demography=$6, original_language=$7, release_year=$8, access=$9, cover_storage_key=$10, group_id=$11, public_at=$12, updated_at=now() WHERE id=$13`,
		st.Title, st.Slug, st.Synopsis, st.Type, st.Status, st.Demography, st.OriginalLanguage, st.ReleaseYear, st.Access, coverKey, gid, pubAt, id)
	return st, err
}

func (s *SeriesService) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM series WHERE id=$1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("serie nao encontrada: %s", id)
	}
	return nil
}

var _ = ports.User{}
