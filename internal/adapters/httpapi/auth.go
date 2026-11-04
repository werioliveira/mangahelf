package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mangashelf/mangashelf/internal/services"
)

type AuthHandler struct {
	svc *services.AuthService
}

func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/auth/register", h.register)
	r.Post("/auth/login", h.login)
	r.Post("/auth/logout", h.logout)
	r.Get("/auth/me", h.me)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decodeCreds(r *http.Request) (email, password string, ok bool) {
	var in map[string]string
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return "", "", false
	}
	email = in["email"]
	password = in["password"]
	ok = email != "" && password != ""
	return
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	email, password, ok := decodeCreds(r)
	if !ok {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	u, err := h.svc.Register(r.Context(), services.RegisterInput{Email: email, Password: password})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.svc.SaveSession(w, r, u.ID); err != nil {
		http.Error(w, "erro ao criar sessao", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"user_id": u.ID, "email": u.Email})
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	email, password, ok := decodeCreds(r)
	if !ok {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	u, err := h.svc.Login(r.Context(), email, password)
	if err != nil {
		http.Error(w, "credenciais invalidas", http.StatusUnauthorized)
		return
	}
	if err := h.svc.SaveSession(w, r, u.ID); err != nil {
		http.Error(w, "erro ao criar sessao", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"user_id": u.ID, "email": u.Email})
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ClearSession(w, r); err != nil {
		http.Error(w, "erro ao encerrar sessao", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	id, ok := h.svc.UserIDFromSession(r)
	if !ok {
		http.Error(w, "nao autenticado", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"user_id": id})
}
