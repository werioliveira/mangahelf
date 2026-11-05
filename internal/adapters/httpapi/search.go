package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mangashelf/mangashelf/internal/services"
)

type SearchHandler struct {
	searchSvc *services.SearchService
}

func NewSearchHandler(searchSvc *services.SearchService) *SearchHandler {
	return &SearchHandler{searchSvc: searchSvc}
}

func (h *SearchHandler) RegisterRoutes(r chi.Router) {
	r.Get("/search", h.search)
}

func (h *SearchHandler) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "query 'q' é obrigatória", http.StatusBadRequest)
		return
	}

	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	publicOnly := r.URL.Query().Get("public_only") == "true"

	results, err := h.searchSvc.Search(r.Context(), services.SearchInput{
		Query:       q,
		Limit:       limit,
		Offset:      offset,
		PublicOnly:  publicOnly,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"query":  q,
		"limit":  limit,
		"offset": offset,
		"total":  len(results),
		"results": results,
	})
}

var _ = strconv.Atoi
