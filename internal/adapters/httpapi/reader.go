package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mangashelf/mangashelf/internal/ports"
	"github.com/mangashelf/mangashelf/internal/services"
)

type PageResponse struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	StorageKey  string `json:"storage_key,omitempty"`
}

type ReaderHandler struct {
	pageSvc    *services.PageService
	imageSvc   *services.ImageURLBuilder
	authSvc    *services.AuthService
	accessSvc  *services.AccessResolver
}

func NewReaderHandler(pageSvc *services.PageService, imageSvc *services.ImageURLBuilder, authSvc *services.AuthService, accessSvc *services.AccessResolver) *ReaderHandler {
	return &ReaderHandler{pageSvc: pageSvc, imageSvc: imageSvc, authSvc: authSvc, accessSvc: accessSvc}
}

func (h *ReaderHandler) RegisterRoutes(r chi.Router) {
	r.Get("/chapters/{id}/pages", h.pages)
}

func (h *ReaderHandler) requireAuth(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := h.authSvc.UserIDFromSession(r)
	if !ok {
		http.Error(w, "nao autenticado", http.StatusUnauthorized)
		return "", false
	}
	return userID, true
}

func (h *ReaderHandler) pages(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.authSvc.UserIDFromSession(r) // anonymous allowed for public content
	chapterID := chi.URLParam(r, "id")
	if chapterID == "" {
		http.Error(w, "chapter_id invalido", http.StatusBadRequest)
		return
	}

	variant := r.URL.Query().Get("variant")
	if variant == "" {
		variant = "full"
	}

	// Resolver único de acesso (proteção VIP/early/vip_only)
	can, err := h.accessSvc.CanRead(r.Context(), userID, chapterID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !can {
		http.Error(w, "acesso negado a este capitulo", http.StatusForbidden)
		return
	}

	pages, err := h.pageSvc.List(r.Context(), chapterID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]PageResponse, len(pages))
	for i, p := range pages {
		pp := ports.Page{
			StorageKey: p.StorageKey,
			Width:      p.Width,
			Height:     p.Height,
			Variant:    variant,
			Access:     "public",
		}
		u, err := h.imageSvc.Build(r.Context(), pp, variant)
		if err != nil {
			u = ""
		}
		out[i] = PageResponse{
			Number:      p.Number,
			URL:         u,
			Width:       p.Width,
			Height:      p.Height,
			StorageKey:  p.StorageKey,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

var _ = context.Background
