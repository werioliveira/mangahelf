package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mangashelf/mangashelf/internal/ports"
)

type Series struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Slug      string `json:"slug"`
	Synopsis  string `json:"synopsis"`
	Access    string `json:"access"`
	GroupID   string `json:"group_id,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CreateSeriesInput struct {
	Title    string `json:"title"`
	Slug     string `json:"slug"`
	Synopsis string `json:"synopsis"`
	Access   string `json:"access"`
	GroupID  string `json:"group_id"`
}

type UpdateSeriesInput struct {
	Title    *string `json:"title,omitempty"`
	Slug     *string `json:"slug,omitempty"`
	Synopsis *string `json:"synopsis,omitempty"`
	Access   *string `json:"access,omitempty"`
	GroupID  *string `json:"group_id,omitempty"`
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
	if in.Access == "" {
		in.Access = "public"
	}
	var id string
	var gid any
	if in.GroupID != "" {
		gid = in.GroupID
	}
	err := s.db.QueryRowContext(ctx,
		"INSERT INTO series (title, slug, synopsis, access, group_id) VALUES ($1, $2, $3, $4, $5) RETURNING id, title, slug, synopsis, access, group_id, created_at, updated_at",
		in.Title, in.Slug, in.Synopsis, in.Access, gid,
	).Scan(&id, &in.Title, &in.Slug, &in.Synopsis, &in.Access, &gid, new(string), new(string))
	if err != nil {
		return Series{}, err
	}
	var groupID string
	if gid != nil {
		groupID = gid.(string)
	}
	return Series{ID: id, Title: in.Title, Slug: in.Slug, Synopsis: in.Synopsis, Access: in.Access, GroupID: groupID}, nil
}

func (s *SeriesService) List(ctx context.Context) ([]Series, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, title, slug, synopsis, access, group_id, created_at, updated_at FROM series ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Series
	for rows.Next() {
		var st Series
		var gid sql.NullString
		if err := rows.Scan(&st.ID, &st.Title, &st.Slug, &st.Synopsis, &st.Access, &gid, new(string), new(string)); err != nil {
			return nil, err
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
	var gid sql.NullString
	err := s.db.QueryRowContext(ctx, "SELECT id, title, slug, synopsis, access, group_id, created_at, updated_at FROM series WHERE id=$1", id).Scan(&st.ID, &st.Title, &st.Slug, &st.Synopsis, &st.Access, &gid, new(string), new(string))
	if err != nil {
		return Series{}, err
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
	if in.Access != nil {
		st.Access = *in.Access
	}
	if in.GroupID != nil {
		st.GroupID = *in.GroupID
	}
	_, err = s.db.ExecContext(ctx, "UPDATE series SET title=$1, slug=$2, synopsis=$3, access=$4, group_id=$5, updated_at=now() WHERE id=$6", st.Title, st.Slug, st.Synopsis, st.Access, anyGroupID(st.GroupID), id)
	return st, err
}

func anyGroupID(s string) any {
	if s == "" {
		return nil
	}
	return s
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
