package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mangashelf/mangashelf/internal/services"
)

type AdminHandler struct {
	adminSvc   *services.AdminService
	authSvc    *services.AuthService
	chapterSvc *services.ChapterService
	seriesSvc  *services.SeriesService
}

func NewAdminHandler(adminSvc *services.AdminService, authSvc *services.AuthService, chapterSvc *services.ChapterService, seriesSvc *services.SeriesService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc, authSvc: authSvc, chapterSvc: chapterSvc, seriesSvc: seriesSvc}
}

func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Use(h.requireStaffMiddleware)
		r.Get("/series", h.listSeries)
		r.Post("/series", h.createSeries)
		r.Patch("/series/{id}", h.updateSeries)
		r.Delete("/series/{id}", h.deleteSeries)
		r.Get("/users", h.listUsers)
		r.Patch("/users/{id}", h.updateUser)
		r.Get("/chapters", h.listChapters)
		r.Post("/chapters", h.createChapter)
		r.Patch("/chapters/{id}", h.updateChapter)
		r.Delete("/chapters/{id}", h.deleteChapter)
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

func (h *AdminHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "user_id invalido", http.StatusBadRequest)
		return
	}

	var in services.AdminUserUpdate
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}

	if err := h.adminSvc.UpdateUserAdmin(r.Context(), id, in); err != nil {
		if err.Error() == "usuario nao encontrado" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) listChapters(w http.ResponseWriter, r *http.Request) {
	list, err := h.chapterSvc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *AdminHandler) createChapter(w http.ResponseWriter, r *http.Request) {
	var in services.CreateChapterInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	ch, err := h.chapterSvc.Create(r.Context(), in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, ch)
}

func (h *AdminHandler) deleteChapter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "chapter_id invalido", http.StatusBadRequest)
		return
	}
	if err := h.chapterSvc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) createSeries(w http.ResponseWriter, r *http.Request) {
	var in services.CreateSeriesInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	s, err := h.seriesSvc.Create(r.Context(), in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (h *AdminHandler) updateSeries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "series_id invalido", http.StatusBadRequest)
		return
	}
	var in services.UpdateSeriesInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	s, err := h.seriesSvc.Update(r.Context(), id, in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *AdminHandler) deleteSeries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "series_id invalido", http.StatusBadRequest)
		return
	}
	if err := h.seriesSvc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
