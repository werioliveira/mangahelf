package authz

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEscalation      = errors.New("authz: privilege escalation blocked")
	ErrForbidden       = errors.New("authz: forbidden")
	ErrLastOwner       = errors.New("authz: cannot remove last owner")
	ErrSelfAssignment  = errors.New("authz: cannot assign role to self")
)

type EscalationGuard struct {
	db       *pgxpool.Pool
	resolver *Resolver
}

func NewEscalationGuard(db *pgxpool.Pool, resolver *Resolver) *EscalationGuard {
	return &EscalationGuard{db: db, resolver: resolver}
}

func (g *EscalationGuard) AssignRole(ctx context.Context, actor Subject, target uuid.UUID, roleID uuid.UUID, sc Scope) error {
	tx, err := g.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	role, err := g.roleForUpdate(ctx, tx, roleID)
	if err != nil {
		return err
	}

	if !g.resolver.Can(ctx, actor, RoleAssign, sc) {
		return ErrForbidden
	}
	if actor.UserID == target {
		return ErrSelfAssignment
	}

	actorRank, err := g.maxPosition(ctx, tx, actor.UserID, sc)
	if err != nil {
		return err
	}
	targetRank, err := g.maxPosition(ctx, tx, target, sc)
	if err != nil {
		return err
	}

	if role.System || role.Position >= actorRank || targetRank >= actorRank {
		return ErrEscalation
	}

	rolePerms, err := g.rolePermissions(ctx, tx, roleID)
	if err != nil {
		return err
	}
	for _, perm := range rolePerms {
		if !g.resolver.Can(ctx, actor, perm, sc) {
			return ErrEscalation
		}
	}

	if err := g.insertUserRole(ctx, tx, target, roleID, actor.UserID); err != nil {
		return err
	}

	if err := g.auditLog(ctx, tx, "role.assign", actor.UserID, "user", target, map[string]any{"role_id": roleID, "scope": sc}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (g *EscalationGuard) RemoveRole(ctx context.Context, actor Subject, target uuid.UUID, roleID uuid.UUID, sc Scope) error {
	tx, err := g.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	role, err := g.roleForUpdate(ctx, tx, roleID)
	if err != nil {
		return err
	}

	if !g.resolver.Can(ctx, actor, RoleAssign, sc) {
		return ErrForbidden
	}
	if actor.UserID == target {
		return ErrSelfAssignment
	}

	actorRank, err := g.maxPosition(ctx, tx, actor.UserID, sc)
	if err != nil {
		return err
	}
	targetRank, err := g.maxPosition(ctx, tx, target, sc)
	if err != nil {
		return err
	}

	if role.Position >= actorRank || targetRank >= actorRank {
		return ErrEscalation
	}

	if role.System {
		count, err := g.countSystemRole(ctx, tx, roleID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastOwner
		}
	}

	if err := g.deleteUserRole(ctx, tx, target, roleID); err != nil {
		return err
	}

	if err := g.auditLog(ctx, tx, "role.remove", actor.UserID, "user", target, map[string]any{"role_id": roleID, "scope": sc}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (g *EscalationGuard) CreateRole(ctx context.Context, actor Subject, name string, position int, permissions []Perm, sc Scope) (uuid.UUID, error) {
	var roleID uuid.UUID
	tx, err := g.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	if !g.resolver.Can(ctx, actor, RoleManage, sc) {
		return uuid.Nil, ErrForbidden
	}

	actorRank, err := g.maxPosition(ctx, tx, actor.UserID, sc)
	if err != nil {
		return uuid.Nil, err
	}
	if position >= actorRank {
		return uuid.Nil, ErrEscalation
	}

	for _, perm := range permissions {
		if !g.resolver.Can(ctx, actor, perm, sc) {
			return uuid.Nil, ErrEscalation
		}
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO roles (name, position, system) VALUES ($1, $2, false) RETURNING id`,
		name, position).Scan(&roleID)
	if err != nil {
		return uuid.Nil, err
	}

	for _, perm := range permissions {
		if _, err := tx.Exec(ctx,
			`INSERT INTO role_permissions (role_id, permission_key) VALUES ($1, $2)`,
			roleID, perm); err != nil {
			return uuid.Nil, err
		}
	}

	if err := g.auditLog(ctx, tx, "role.create", actor.UserID, "role", roleID, map[string]any{"name": name, "position": position}); err != nil {
		return uuid.Nil, err
	}

	return roleID, tx.Commit(ctx)
}

func (g *EscalationGuard) UpdateRole(ctx context.Context, actor Subject, roleID uuid.UUID, newPosition *int, newPermissions []Perm, sc Scope) error {
	tx, err := g.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	role, err := g.roleForUpdate(ctx, tx, roleID)
	if err != nil {
		return err
	}

	if !g.resolver.Can(ctx, actor, RoleManage, sc) {
		return ErrForbidden
	}

	actorRank, err := g.maxPosition(ctx, tx, actor.UserID, sc)
	if err != nil {
		return err
	}

	if newPosition != nil {
		if *newPosition >= actorRank {
			return ErrEscalation
		}
		if role.System && *newPosition != role.Position {
			return ErrEscalation
		}
	}

	if newPermissions != nil {
		for _, perm := range newPermissions {
			if !g.resolver.Can(ctx, actor, perm, sc) {
				return ErrEscalation
			}
		}
	}

	if newPosition != nil {
		if _, err := tx.Exec(ctx, `UPDATE roles SET position = $1 WHERE id = $2`, *newPosition, roleID); err != nil {
			return err
		}
	}

	if newPermissions != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
			return err
		}
		for _, perm := range newPermissions {
			if _, err := tx.Exec(ctx,
				`INSERT INTO role_permissions (role_id, permission_key) VALUES ($1, $2)`,
				roleID, perm); err != nil {
				return err
			}
		}
	}

	if err := g.auditLog(ctx, tx, "role.update", actor.UserID, "role", roleID, nil); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (g *EscalationGuard) BanUser(ctx context.Context, actor Subject, target uuid.UUID, reason string, sc Scope) error {
	if !g.resolver.Can(ctx, actor, UserBan, sc) {
		return ErrForbidden
	}

	actorRank, err := g.maxPosition(ctx, g.db, actor.UserID, sc)
	if err != nil {
		return err
	}
	targetRank, err := g.maxPosition(ctx, g.db, target, sc)
	if err != nil {
		return err
	}

	if targetRank >= actorRank {
		return ErrEscalation
	}

	_, err = g.db.Exec(ctx,
		`INSERT INTO bans (user_id, reason, created_by) VALUES ($1, $2, $3)`,
		target, reason, actor.UserID)
	return err
}

type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

type Role struct {
	ID       uuid.UUID
	Name     string
	Position int
	System   bool
}

func (g *EscalationGuard) roleForUpdate(ctx context.Context, tx pgx.Tx, roleID uuid.UUID) (*Role, error) {
	var r Role
	err := tx.QueryRow(ctx,
		`SELECT id, name, position, system FROM roles WHERE id = $1 FOR UPDATE`, roleID).Scan(&r.ID, &r.Name, &r.Position, &r.System)
	return &r, err
}

func (g *EscalationGuard) maxPosition(ctx context.Context, q Querier, userID uuid.UUID, sc Scope) (int, error) {
	var query string
	var args []interface{}

	if sc.Kind == Group {
		query = `SELECT COALESCE(MAX(r.position), 0) FROM roles r
			JOIN group_members gm ON gm.role_id = r.id
			WHERE gm.user_id = $1 AND gm.group_id = $2`
		args = []interface{}{userID, sc.GroupID}
	} else {
		query = `SELECT COALESCE(MAX(r.position), 0) FROM roles r
			JOIN user_roles ur ON ur.role_id = r.id
			WHERE ur.user_id = $1`
		args = []interface{}{userID}
	}

	var pos int
	err := q.QueryRow(ctx, query, args...).Scan(&pos)
	return pos, err
}

func (g *EscalationGuard) rolePermissions(ctx context.Context, tx pgx.Tx, roleID uuid.UUID) ([]Perm, error) {
	rows, err := tx.Query(ctx, `SELECT permission_key FROM role_permissions WHERE role_id = $1`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []Perm
	for rows.Next() {
		var p Perm
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (g *EscalationGuard) countSystemRole(ctx context.Context, tx pgx.Tx, roleID uuid.UUID) (int, error) {
	var count int
	err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE r.id = $1 AND r.system = true`, roleID).Scan(&count)
	return count, err
}

func (g *EscalationGuard) insertUserRole(ctx context.Context, tx pgx.Tx, userID, roleID, grantedBy uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id, granted_by) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, role_id) DO NOTHING`,
		userID, roleID, grantedBy)
	return err
}

func (g *EscalationGuard) deleteUserRole(ctx context.Context, tx pgx.Tx, userID, roleID uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`,
		userID, roleID)
	return err
}

func (g *EscalationGuard) auditLog(ctx context.Context, tx pgx.Tx, action string, actorID uuid.UUID, targetType string, targetID uuid.UUID, meta map[string]any) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO audit_log (actor_id, action, target_type, target_id, after)
		 VALUES ($1, $2, $3, $4, $5)`,
		actorID, action, targetType, targetID.String(), meta)
	return err
}