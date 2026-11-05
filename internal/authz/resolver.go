package authz

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mangashelf/mangashelf/internal/auth"
	"log/slog"
)

var (
	ErrUnknownPermission = errors.New("authz: unknown permission")
	ErrInactiveUser      = errors.New("authz: user inactive")
	ErrMFARequired       = errors.New("authz: MFA required")
)

type Subject struct {
	UserID      uuid.UUID
	Active      bool
	MFAVerified bool
}

type Resolver struct {
	db  *pgxpool.Pool
	defs map[Perm]Def
}

func NewResolver(db *pgxpool.Pool) *Resolver {
	return &Resolver{db: db, defs: CatalogMap()}
}

type grant struct {
	Source  string
	Key     Perm
	Effect  string
}

func (r *Resolver) Can(ctx context.Context, s Subject, perm Perm, sc Scope) bool {
	def, known := r.defs[perm]
	if !known || !s.Active {
		return false
	}
	if def.RequiresMFA && !s.MFAVerified {
		return false
	}
	if sc.Kind == Group && !def.allowsScope(Group) {
		return false
	}

	grants, err := r.grantsFor(ctx, s.UserID, sc)
	if err != nil {
		slog.ErrorContext(ctx, "authz: grants lookup failed", "err", err)
		return false
	}

	var allow, groupRole, globalRole bool
	for _, g := range grants {
		if g.Key != perm {
			continue
		}
		switch {
		case g.Source == "override" && g.Effect == "deny":
			return false
		case g.Source == "override" && g.Effect == "allow":
			allow = true
		case g.Source == "role_group":
			groupRole = true
		case g.Source == "role_global":
			globalRole = true
		}
	}
	return allow || groupRole || globalRole
}

func (r *Resolver) grantsFor(ctx context.Context, userID uuid.UUID, sc Scope) ([]grant, error) {
	var groupID interface{} = nil
	if sc.Kind == Group {
		groupID = sc.GroupID
	}

	query := `
		SELECT 'role_global'::text AS source, rp.permission_key, 'allow'::text AS effect
		FROM user_roles ur
		JOIN role_permissions rp ON rp.role_id = ur.role_id
		WHERE ur.user_id = $1
		UNION ALL
		SELECT 'role_group', rp.permission_key, 'allow'
		FROM group_members gm
		JOIN role_permissions rp ON rp.role_id = gm.role_id
		WHERE gm.user_id = $1 AND gm.group_id = $2
		UNION ALL
		SELECT 'override', o.permission_key, o.effect
		FROM user_overrides o
		WHERE o.user_id = $1
		  AND (o.scope_group_id IS NULL OR o.scope_group_id = $2)
		  AND (o.expires_at IS NULL OR o.expires_at > now());
	`

	var args []interface{}
	args = append(args, userID)
	if groupID != nil {
		args = append(args, groupID)
	} else {
		args = append(args, nil)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grants []grant
	for rows.Next() {
		var g grant
		if err := rows.Scan(&g.Source, &g.Key, &g.Effect); err != nil {
			return nil, err
		}
		grants = append(grants, g)
	}
	return grants, rows.Err()
}

func SubjectFromPrincipal(p *auth.Principal, mfaWindow time.Duration) Subject {
	return Subject{
		UserID:      p.UserID,
		Active:      p.Status == "active",
		MFAVerified: p.MFAAt != nil && time.Since(*p.MFAAt) <= mfaWindow,
	}
}