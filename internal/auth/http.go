package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	svc *Service
	cfg Config
}

func NewAuthHandler(svc *Service, cfg Config) *AuthHandler {
	return &AuthHandler{svc: svc, cfg: cfg}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/auth/register", h.register)
	r.Post("/auth/login", h.login)
	r.Post("/auth/logout", h.logout)
	r.Get("/auth/me", h.me)
	r.Post("/auth/mfa/totp/begin", h.totpBegin)
	r.Post("/auth/mfa/totp/confirm", h.totpConfirm)
	r.Post("/auth/mfa/totp/remove", h.totpRemove)
	r.Get("/auth/mfa/status", h.mfaStatus)
	r.Post("/auth/mfa/verify", h.mfaVerify)
	r.Post("/auth/password/forgot", h.forgotPassword)
	r.Post("/auth/password/reset", h.resetPassword)
	r.Post("/auth/password/change", h.changePassword)
	r.Post("/auth/email/change", h.changeEmail)
	r.Post("/auth/email/change/confirm", h.confirmEmailChange)
	r.Post("/auth/email/verify", h.verifyEmail)
	r.Post("/auth/email/resend", h.resendVerification)
	r.Get("/auth/csrf", h.csrfToken)
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	var in RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	u, err := h.svc.Register(r.Context(), in)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"user_id": u.ID.String(), "email": u.Email})
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var in LoginInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	ip := ClientIP(r, h.cfg.TrustedProxies, h.cfg.ClientIPHeader)
	ua := r.UserAgent()

	res, err := h.svc.Login(r.Context(), in, ip, ua)
	if err != nil {
		h.handleError(w, err)
		return
	}

	if res.MFARequired {
		writeJSON(w, http.StatusOK, map[string]any{
			"mfa_required": true,
			"methods":      res.MFAMethods,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Logout(r.Context()); err != nil {
		http.Error(w, "logout failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"user_id": p.UserID.String()})
}

func (h *AuthHandler) totpBegin(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	url, err := h.svc.EnrollTOTP(r.Context(), p.UserID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"otpauth_url": url})
}

func (h *AuthHandler) totpConfirm(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var in struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	codes, err := h.svc.ConfirmTOTP(r.Context(), p.UserID, in.Code)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}

func (h *AuthHandler) totpRemove(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.svc.RemoveTOTP(r.Context(), p.UserID); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) mfaStatus(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	enrolled, confirmed, recoveryCount, err := h.svc.GetTOTPStatus(r.Context(), p.UserID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	credCount := 0
	if h.svc.webauthn != nil {
		creds, _ := h.svc.ListWebAuthnCredentials(r.Context(), p.UserID)
		credCount = len(creds)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"totp_enrolled":      enrolled,
		"totp_confirmed":     confirmed,
		"recovery_codes":     recoveryCount,
		"passkey_count":      credCount,
	})
}

func (h *AuthHandler) mfaVerify(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Method string `json:"method"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	ip := ClientIP(r, h.cfg.TrustedProxies, h.cfg.ClientIPHeader)
	ua := r.UserAgent()

	res, err := h.svc.CompleteMFA(r.Context(), in.Method, in.Code, ip, ua)
	if err != nil {
		h.handleError(w, err)
		return
	}

	if res.MFARequired {
		writeJSON(w, http.StatusOK, map[string]any{
			"mfa_required": true,
			"methods":      res.MFAMethods,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	ip := ClientIP(r, h.cfg.TrustedProxies, h.cfg.ClientIPHeader)

	if err := h.svc.ForgotPassword(r.Context(), in.Email, ip); err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "check_email"})
}

func (h *AuthHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := h.svc.ResetPassword(r.Context(), in.Token, in.NewPassword); err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	ip := ClientIP(r, h.cfg.TrustedProxies, h.cfg.ClientIPHeader)

	if err := h.svc.ChangePassword(r.Context(), p.UserID, in.CurrentPassword, in.NewPassword, ip); err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) changeEmail(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var in struct {
		NewEmail string `json:"new_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := h.svc.ChangeEmail(r.Context(), p.UserID, in.NewEmail); err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "check_email"})
}

func (h *AuthHandler) confirmEmailChange(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := h.svc.ConfirmEmailChange(r.Context(), in.Token); err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := h.svc.VerifyEmail(r.Context(), in.Token); err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) resendVerification(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	u, err := h.svc.q.GetUserByID(r.Context(), p.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	ip := ClientIP(r, h.cfg.TrustedProxies, h.cfg.ClientIPHeader)

	if err := h.svc.ResendVerification(r.Context(), u.Email, ip); err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "check_email"})
}

func (h *AuthHandler) csrfToken(w http.ResponseWriter, r *http.Request) {
	token := h.svc.sessions.GetCSRF(r.Context())
	if token == "" {
		http.Error(w, "no session", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"csrf_token": token})
}

func (h *AuthHandler) handleError(w http.ResponseWriter, err error) {
	var rl ErrRateLimited
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
	case errors.Is(err, ErrAccountBlocked):
		http.Error(w, "account blocked", http.StatusForbidden)
	case errors.Is(err, ErrAccountSuspended):
		http.Error(w, "account suspended", http.StatusForbidden)
	case errors.Is(err, ErrEmailExists):
		http.Error(w, "email already exists", http.StatusConflict)
	case errors.Is(err, ErrEmailNotVerified):
		http.Error(w, "email not verified", http.StatusForbidden)
	case errors.Is(err, ErrMFARequired):
		http.Error(w, "mfa required", http.StatusForbidden)
	case errors.Is(err, ErrInvalidToken), errors.Is(err, ErrTokenExpired), errors.Is(err, ErrTokenUsed):
		http.Error(w, "invalid or expired token", http.StatusBadRequest)
	case errors.As(err, &rl):
		w.Header().Set("Retry-After", rl.RetryAfter.String())
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	case strings.Contains(err.Error(), "password too short"):
		http.Error(w, "password too short (min 12 chars)", http.StatusBadRequest)
	case strings.Contains(err.Error(), "password too long"):
		http.Error(w, "password too long (max 256 bytes)", http.StatusBadRequest)
	case strings.Contains(err.Error(), "password contains"):
		http.Error(w, "password cannot contain email or username", http.StatusBadRequest)
	case strings.Contains(err.Error(), "password is too common"):
		http.Error(w, "password is too common", http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}