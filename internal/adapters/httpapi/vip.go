package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mangashelf/mangashelf/internal/services"
)

type VipHandler struct {
	vipSvc  *services.VipService
	authSvc *services.AuthService
}

func NewVipHandler(vipSvc *services.VipService, authSvc *services.AuthService) *VipHandler {
	return &VipHandler{
		vipSvc: vipSvc,
		authSvc: authSvc,
	}
}

func (h *VipHandler) requireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := h.authSvc.UserIDFromSession(r)
		if !ok {
			http.Error(w, "nao autenticado", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *VipHandler) RegisterRoutes(r chi.Router) {
	r.Route("/vip", func(r chi.Router) {
		r.Use(h.requireAuthMiddleware)
		r.Get("/plans", h.listPlans)
		r.Post("/plans", h.createPlan)
		r.Post("/grant", h.grantVip)
		r.Post("/voucher", h.redeemVoucher)
		r.Post("/vouchers", h.createVoucher)
		r.Get("/me", h.me)
	})
}

func (h *VipHandler) listPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.vipSvc.ListPlans(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, plans)
}

func (h *VipHandler) createPlan(w http.ResponseWriter, r *http.Request) {
	var in services.CreatePlanInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	plan, err := h.vipSvc.CreatePlan(r.Context(), in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, plan)
}

func (h *VipHandler) grantVip(w http.ResponseWriter, r *http.Request) {
	var in services.GrantVipInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	if err := h.vipSvc.GrantVip(r.Context(), in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *VipHandler) redeemVoucher(w http.ResponseWriter, r *http.Request) {
	var in services.RedeemVoucherInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	if err := h.vipSvc.RedeemVoucher(r.Context(), in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *VipHandler) createVoucher(w http.ResponseWriter, r *http.Request) {
	type createVoucherInput struct {
		PlanID    string  `json:"plan_id"`
		MaxUses   int     `json:"max_uses"`
		ExpiresAt *string `json:"expires_at,omitempty"`
	}
	var in createVoucherInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	if in.PlanID == "" {
		http.Error(w, "plan_id e obrigatorio", http.StatusBadRequest)
		return
	}
	if in.MaxUses <= 0 {
		in.MaxUses = 1
	}
	var expArg *time.Time
	if in.ExpiresAt != nil && *in.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *in.ExpiresAt)
		if err != nil {
			http.Error(w, "expires_at invalido (use RFC3339)", http.StatusBadRequest)
			return
		}
		expArg = &t
	}
	code, err := h.vipSvc.CreateVoucher(r.Context(), in.PlanID, in.MaxUses, expArg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"code": code})
}

func (h *VipHandler) me(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authSvc.UserIDFromSession(r)
	if !ok {
		http.Error(w, "nao autenticado", http.StatusUnauthorized)
		return
	}
	status, err := h.vipSvc.GetVipStatus(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, status)
}
