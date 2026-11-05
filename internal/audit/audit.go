package audit

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAuditFailed = errors.New("audit: failed to log")

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

type Actor struct {
	ID uuid.UUID
}

type Meta map[string]any

func (s *Service) Log(ctx context.Context, action string, actor *Actor, meta Meta) {
	if actor == nil {
		s.log(ctx, nil, action, "", "", nil, meta)
		return
	}
	s.log(ctx, &actor.ID, action, "", "", nil, meta)
}

func (s *Service) LogTx(ctx context.Context, tx pgx.Tx, action string, actorID uuid.UUID, targetType string, targetID uuid.UUID, before, after Meta) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO audit_log (actor_id, action, target_type, target_id, before, after)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		actorID, action, targetType, targetID.String(), before, after)
	return err
}

func (s *Service) log(ctx context.Context, actorID *uuid.UUID, action, targetType, targetID string, before, after Meta) {
	var actor interface{} = nil
	if actorID != nil {
		actor = *actorID
	}
	var tType interface{} = nil
	if targetType != "" {
		tType = targetType
	}
	var tID interface{} = nil
	if targetID != "" {
		tID = targetID
	}

	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)

	_, err := s.db.Exec(ctx,
		`INSERT INTO audit_log (actor_id, action, target_type, target_id, before, after)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		actor, action, tType, tID, beforeJSON, afterJSON)
	if err != nil {
		return
	}
}

func (s *Service) Denied(ctx context.Context, perm string, scope Scope) {
	s.log(ctx, nil, "authz.denied", "", "", nil, Meta{"permission": perm, "scope": scope.String()})
}

type Scope struct {
	Kind    int
	GroupID uuid.UUID
}

func (s Scope) String() string {
	if s.Kind == 1 {
		return "group:" + s.GroupID.String()
	}
	return "global"
}