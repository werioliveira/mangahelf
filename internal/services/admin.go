package services

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"
)

// AdminService oferece operacoes de moderacao para staff/editor.
type AdminService struct {
	db *sql.DB
}

func NewAdminService(db *sql.DB) *AdminService {
	return &AdminService{db: db}
}

// IsStaff verifica se o usuario tem permissao de moderacao (qualquer permissao de
// chapter.publish, chapter.set_access, media.manage ou report.handle).
func (s *AdminService) IsStaff(ctx context.Context, userID string) (bool, error) {
	var exists bool
    err := s.db.QueryRowContext(ctx,
        `SELECT EXISTS(
             SELECT 1 FROM user_roles ur
             JOIN role_permissions rp ON rp.role_id = ur.role_id
             WHERE ur.user_id = $1::uuid
               AND rp.permission_key IN ('chapter.publish','chapter.set_access','media.manage','report.handle')
        )`,
        userID,
    ).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// SeriesAdmin representa uma serie para listagem administrativa.
type SeriesAdmin struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Synopsis    string `json:"synopsis"`
	Access      string `json:"access"`
	GroupID     string `json:"group_id,omitempty"`
	State       string `json:"state"`
	ChapterCount int   `json:"chapter_count"`
	CreatedAt   string `json:"created_at"`
}

// ListSeriesAdmin lista todas as series para moderacao.
func (s *AdminService) ListSeriesAdmin(ctx context.Context) ([]SeriesAdmin, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT s.id, s.title, s.slug, s.synopsis, s.access, s.group_id,
		       COUNT(c.id) as chapter_count, s.created_at
		 FROM series s
		 LEFT JOIN chapters c ON c.series_id = s.id
		 GROUP BY s.id, s.title, s.slug, s.synopsis, s.access, s.group_id, s.created_at
		 ORDER BY s.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SeriesAdmin
	for rows.Next() {
		var se SeriesAdmin
		var gid sql.NullString
		var synopsis sql.NullString
		if err := rows.Scan(&se.ID, &se.Title, &se.Slug, &synopsis, &se.Access, &gid, &se.ChapterCount, &se.CreatedAt); err != nil {
			return nil, err
		}
		if synopsis.Valid {
			se.Synopsis = synopsis.String
		}
		if gid.Valid {
			se.GroupID = gid.String
		}
		out = append(out, se)
	}
	return out, rows.Err()
}

// AdminChapterUpdate representa mudancas permitidas por admin em um capitulo.
type AdminChapterUpdate struct {
	State  *string `json:"state,omitempty"`
	Access *string `json:"access,omitempty"`
	Title  *string `json:"title,omitempty"`
}

// UpdateChapterAdmin permite que staff edite um capitulo.
func (s *AdminService) UpdateChapterAdmin(ctx context.Context, chapterID string, in AdminChapterUpdate) error {
	// verificar se o capitulo existe
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM chapters WHERE id=$1)", chapterID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("capitulo nao encontrado")
	}

	// construir update dinamico
	sets := []string{}
	args := []any{}
	argIdx := 1
	if in.State != nil {
		sets = append(sets, "state=$"+strconv.Itoa(argIdx))
		args = append(args, *in.State)
		argIdx++
	}
	if in.Access != nil {
		sets = append(sets, "access=$"+strconv.Itoa(argIdx))
		args = append(args, *in.Access)
		argIdx++
	}
	if in.Title != nil {
		sets = append(sets, "title=$"+strconv.Itoa(argIdx))
		args = append(args, *in.Title)
		argIdx++
	}
	if len(sets) == 0 {
		return errors.New("nenhum campo para atualizar")
	}
	sets = append(sets, "updated_at=now()")
	args = append(args, chapterID)

	query := "UPDATE chapters SET " + sets[0]
	for i := 1; i < len(sets); i++ {
		query += ", " + sets[i]
	}
	query += " WHERE id=$" + strconv.Itoa(argIdx)

	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

// AdminUser representa um usuario para listagem administrativa.
type AdminUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	IsActive  bool   `json:"is_active"`
	Roles     string `json:"roles,omitempty"`
	HasVip    bool   `json:"has_vip"`
	VipExpires string `json:"vip_expires,omitempty"`
	CreatedAt string `json:"created_at"`
}

// ListUsersAdmin lista todos os usuarios para moderacao.
func (s *AdminService) ListUsersAdmin(ctx context.Context) ([]AdminUser, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT u.id, u.email, u.status,
		       COALESCE(string_agg(DISTINCT rp.permission_key, ','), '') as roles,
		       CASE WHEN vm.expires_at > now() THEN true ELSE false END as has_vip,
		       vm.expires_at as vip_expires,
		       u.created_at
		 FROM users u
		 LEFT JOIN user_roles ur ON ur.user_id = u.id
		 LEFT JOIN role_permissions rp ON rp.role_id = ur.role_id
		 LEFT JOIN vip_memberships vm ON vm.user_id = u.id AND vm.expires_at > now()
		 GROUP BY u.id, u.email, u.status, vm.expires_at, u.created_at
		 ORDER BY u.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AdminUser
	for rows.Next() {
	var u AdminUser
	var roles sql.NullString
	var status string
	var hasVip sql.NullBool
	var vipExpires sql.NullTime
	if err := rows.Scan(&u.ID, &u.Email, &status, &roles, &hasVip, &vipExpires, &u.CreatedAt); err != nil {
		return nil, err
	}
	u.IsActive = (status == "active")
		if roles.Valid {
			u.Roles = roles.String
		}
		if hasVip.Valid {
			u.HasVip = hasVip.Bool
		}
		if vipExpires.Valid {
			u.VipExpires = vipExpires.Time.Format(time.RFC3339)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

var _ = errors.New

// AdminUserUpdate representa mudancas permitidas por admin em um usuario.
type AdminUserUpdate struct {
	Status *string `json:"status,omitempty"`
}

// UpdateUserAdmin permite que staff edite o status de um usuario.
func (s *AdminService) UpdateUserAdmin(ctx context.Context, userID string, in AdminUserUpdate) error {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", userID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("usuario nao encontrado")
	}

	sets := []string{}
	args := []any{}
	argIdx := 1
	if in.Status != nil {
		sets = append(sets, "status=$"+strconv.Itoa(argIdx))
		args = append(args, *in.Status)
		argIdx++
	}
	if len(sets) == 0 {
		return errors.New("nenhum campo para atualizar")
	}
	sets = append(sets, "updated_at=now()")
	args = append(args, userID)

	query := "UPDATE users SET " + sets[0]
	for i := 1; i < len(sets); i++ {
		query += ", " + sets[i]
	}
	query += " WHERE id=$" + strconv.Itoa(argIdx)

	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}
