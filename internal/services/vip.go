package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"
)

type VipPlan struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Duration  int    `json:"duration"` // dias, ex: 30, 90, 365
	Perks     string `json:"perks"`    // JSON
	CreatedAt string `json:"created_at"`
}

type CreatePlanInput struct {
	Name     string `json:"name"`
	Duration int    `json:"duration"` // dias, ex: 30, 90, 365
	Perks    string `json:"perks"`
}

type GrantVipInput struct {
	UserID string `json:"user_id"`
	PlanID string `json:"plan_id"`
	Source string `json:"source"` // admin | voucher | payment
}

type RedeemVoucherInput struct {
	Code   string `json:"code"`
	UserID string `json:"user_id"`
}

type VipStatus struct {
	HasActiveVip  bool   `json:"has_active_vip"`
	PlanName      string `json:"plan_name,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	DaysRemaining int    `json:"days_remaining,omitempty"`
	Source        string `json:"source,omitempty"`
}

type VipService struct {
	db *sql.DB
}

func NewVipService(db *sql.DB) *VipService {
	return &VipService{db: db}
}

func (s *VipService) CreatePlan(ctx context.Context, in CreatePlanInput) (VipPlan, error) {
	if in.Name == "" {
		return VipPlan{}, errors.New("nome e obrigatorio")
	}
	if in.Duration <= 0 {
		return VipPlan{}, errors.New("duracao deve ser maior que zero")
	}
	var id string
	err := s.db.QueryRowContext(ctx,
		"INSERT INTO vip_plans (name, duration, perks) VALUES ($1, $2, $3) RETURNING id, name, duration, perks, created_at",
		in.Name, in.Duration, in.Perks,
	).Scan(&id, &in.Name, &in.Duration, &in.Perks, new(string))
	if err != nil {
		return VipPlan{}, err
	}
	return VipPlan{ID: id, Name: in.Name, Duration: in.Duration, Perks: in.Perks}, nil
}

func (s *VipService) ListPlans(ctx context.Context) ([]VipPlan, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, duration, perks, created_at FROM vip_plans ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VipPlan
	for rows.Next() {
		var p VipPlan
		if err := rows.Scan(&p.ID, &p.Name, &p.Duration, &p.Perks, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *VipService) GrantVip(ctx context.Context, in GrantVipInput) error {
	if in.UserID == "" {
		return errors.New("user_id e obrigatorio")
	}
	if in.PlanID == "" {
		return errors.New("plan_id e obrigatorio")
	}
	if in.Source == "" {
		in.Source = "admin"
	}
	// verificar se o plano existe e buscar duracao em dias
	var days int
	err := s.db.QueryRowContext(ctx, "SELECT duration FROM vip_plans WHERE id=$1", in.PlanID).Scan(&days)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("plano nao encontrado")
		}
		return err
	}
	// verificar se o usuario existe
	var exists bool
	err = s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", in.UserID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("usuario nao encontrado")
	}
	// inserir assinatura VIP
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO vip_memberships (user_id, plan_id, starts_at, expires_at, source) VALUES ($1, $2, now(), now() + $3 * interval '1 day', $4)",
		in.UserID, in.PlanID, days, in.Source,
	)
	return err
}

func (s *VipService) RedeemVoucher(ctx context.Context, in RedeemVoucherInput) error {
	if in.Code == "" {
		return errors.New("codigo e obrigatorio")
	}
	if in.UserID == "" {
		return errors.New("user_id e obrigatorio")
	}
	// buscar voucher valido e nao expirado
	var planID string
	var maxUses, usedCount int
	var expiresAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		"SELECT plan_id, max_uses, used_count, expires_at FROM vip_vouchers WHERE code_hash=$1",
		in.Code,
	).Scan(&planID, &maxUses, &usedCount, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("voucher nao encontrado")
		}
		return err
	}
	if usedCount >= maxUses {
		return errors.New("voucher esgotado")
	}
	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		return errors.New("voucher expirado")
	}
	// verificar se o usuario existe
	var exists bool
	err = s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", in.UserID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("usuario nao encontrado")
	}
	// buscar duracao do plano
	var days int
	err = s.db.QueryRowContext(ctx, "SELECT duration FROM vip_plans WHERE id=$1", planID).Scan(&days)
	if err != nil {
		return err
	}
	// conceder VIP
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO vip_memberships (user_id, plan_id, starts_at, expires_at, source) VALUES ($1, $2, now(), now() + $3 * interval '1 day', 'voucher')",
		in.UserID, planID, days,
	)
	if err != nil {
		return err
	}
	// incrementar contador de usos
	_, err = s.db.ExecContext(ctx, "UPDATE vip_vouchers SET used_count = used_count + 1 WHERE code_hash=$1", in.Code)
	return err
}

func (s *VipService) GetVipStatus(ctx context.Context, userID string) (VipStatus, error) {
	var status VipStatus
	var planName sql.NullString
	var expiresAt sql.NullTime
	var source sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT vp.name, vm.expires_at, vm.source
		 FROM vip_memberships vm
		 JOIN vip_plans vp ON vp.id = vm.plan_id
		 WHERE vm.user_id=$1 AND vm.expires_at > now()
		 ORDER BY vm.expires_at ASC
		 LIMIT 1`,
		userID,
	).Scan(&planName, &expiresAt, &source)
	if err != nil {
		if err == sql.ErrNoRows {
			status.HasActiveVip = false
			return status, nil
		}
		return status, err
	}
	status.HasActiveVip = true
	if planName.Valid {
		status.PlanName = planName.String
	}
	if expiresAt.Valid {
		status.ExpiresAt = expiresAt.Time.Format(time.RFC3339)
		status.DaysRemaining = int(time.Until(expiresAt.Time).Hours() / 24)
		if status.DaysRemaining < 0 {
			status.DaysRemaining = 0
		}
	}
	if source.Valid {
		status.Source = source.String
	}
	return status, nil
}

func generateVoucherCode() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *VipService) CreateVoucher(ctx context.Context, planID string, maxUses int, expiresAt *time.Time) (string, error) {
	code, err := generateVoucherCode()
	if err != nil {
		return "", err
	}
	// verificar se o plano existe
	var days int
	err = s.db.QueryRowContext(ctx, "SELECT duration FROM vip_plans WHERE id=$1", planID).Scan(&days)
	if err != nil {
		return "", err
	}
	_ = days // duracao nao eh necessaria na criacao do voucher, so no resgate
	var expArg any
	if expiresAt != nil {
		expArg = *expiresAt
	}
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO vip_vouchers (code_hash, plan_id, max_uses, expires_at) VALUES ($1, $2, $3, $4)",
		code, planID, maxUses, expArg,
	)
	if err != nil {
		return "", err
	}
	return code, nil
}

var _ = fmt.Sprintf

// VipPlanUpdate representa mudancas permitidas em um plano VIP.
type VipPlanUpdate struct {
	Name        *string `json:"name,omitempty"`
	Duration    *int    `json:"duration,omitempty"`
	Perks       *string `json:"perks,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// UpdatePlan permite editar um plano VIP existente.
func (s *VipService) UpdatePlan(ctx context.Context, planID string, in VipPlanUpdate) error {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM vip_plans WHERE id=$1)", planID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("plano nao encontrado")
	}

	sets := []string{}
	args := []any{}
	argIdx := 1
	if in.Name != nil {
		sets = append(sets, "name=$"+strconv.Itoa(argIdx))
		args = append(args, *in.Name)
		argIdx++
	}
	if in.Duration != nil {
		sets = append(sets, "duration=$"+strconv.Itoa(argIdx))
		args = append(args, *in.Duration)
		argIdx++
	}
	if in.Perks != nil {
		sets = append(sets, "perks=$"+strconv.Itoa(argIdx))
		args = append(args, *in.Perks)
		argIdx++
	}
	if len(sets) == 0 {
		return errors.New("nenhum campo para atualizar")
	}
	sets = append(sets, "created_at=now()")
	args = append(args, planID)

	query := "UPDATE vip_plans SET " + sets[0]
	for i := 1; i < len(sets); i++ {
		query += ", " + sets[i]
	}
	query += " WHERE id=$" + strconv.Itoa(argIdx)

	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}
