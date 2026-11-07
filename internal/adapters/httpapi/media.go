package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mangashelf/mangashelf/internal/services"
)

type MediaHandler struct {
	mediaSvc *services.MediaService
	authSvc  *services.AuthService
}

func NewMediaHandler(mediaSvc *services.MediaService, authSvc *services.AuthService) *MediaHandler {
	return &MediaHandler{mediaSvc: mediaSvc, authSvc: authSvc}
}

func (h *MediaHandler) RegisterRoutes(r chi.Router) {
	r.Post("/admin/uploads/presign", h.presignUpload)
	r.Post("/admin/uploads", h.createUpload)
}

func (h *MediaHandler) requireAuth(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := h.authSvc.UserIDFromSession(r)
	if !ok {
		http.Error(w, "nao autenticado", http.StatusUnauthorized)
		return "", false
	}
	return userID, true
}

func (h *MediaHandler) presignUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAuth(w, r)
	if !ok {
		return
	}
	var in services.PresignUploadInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	out, err := h.mediaSvc.PresignUpload(r.Context(), userID, in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *MediaHandler) createUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAuth(w, r)
	if !ok {
		return
	}
	var in services.CreateUploadInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	out, err := h.mediaSvc.CreateUpload(r.Context(), userID, in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
