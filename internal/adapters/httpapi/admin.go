package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mangashelf/mangashelf/internal/services"
)

type AdminHandler struct {
	adminSvc *services.AdminService
	authSvc  *services.AuthService
}

func NewAdminHandler(adminSvc *services.AdminService, authSvc *services.AuthService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc, authSvc: authSvc}
}

func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Use(h.requireStaffMiddleware)
		r.Get("/series", h.listSeries)
		r.Get("/users", h.listUsers)
		r.Patch("/chapters/{id}", h.updateChapter)
	})
}

func (h *AdminHandler) requireStaffMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := h.authSvc.UserIDFromSession(r)
		if !ok {
			http.Error(w, "nao autenticado", http.StatusUnauthorized)
			return
		}
		isStaff, err := h.adminSvc.IsStaff(r.Context(), userID)
		if err != nil {
			http.Error(w, "erro ao verificar permissao", http.StatusInternalServerError)
			return
		}
		if !isStaff {
			http.Error(w, "permissao negada (staff requerido)", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *AdminHandler) listSeries(w http.ResponseWriter, r *http.Request) {
	list, err := h.adminSvc.ListSeriesAdmin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *AdminHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	list, err := h.adminSvc.ListUsersAdmin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *AdminHandler) updateChapter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "chapter_id invalido", http.StatusBadRequest)
		return
	}

	var in services.AdminChapterUpdate
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}

	if err := h.adminSvc.UpdateChapterAdmin(r.Context(), id, in); err != nil {
		if err.Error() == "capitulo nao encontrado" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ensure strconv is imported even if not directly used in this file after refactor
var _ = strconv.Itoa
