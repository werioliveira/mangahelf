package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mangashelf/mangashelf/internal/services"
)

type CatalogHandler struct {
	seriesSvc *services.SeriesService
	chapterSvc *services.ChapterService
}

func NewCatalogHandler(seriesSvc *services.SeriesService, chapterSvc *services.ChapterService) *CatalogHandler {
	return &CatalogHandler{seriesSvc: seriesSvc, chapterSvc: chapterSvc}
}

func (h *CatalogHandler) RegisterRoutes(r chi.Router) {
	r.Route("/series", func(r chi.Router) {
		r.Get("/", h.listSeries)
		r.Post("/", h.createSeries)
		r.Get("/{id}", h.getSeries)
		r.Put("/{id}", h.updateSeries)
		r.Delete("/{id}", h.deleteSeries)
	})
	r.Route("/chapters", func(r chi.Router) {
		r.Get("/", h.listChapters)
		r.Post("/", h.createChapter)
		r.Get("/{id}", h.getChapter)
		r.Put("/{id}", h.updateChapter)
		r.Delete("/{id}", h.deleteChapter)
		r.Get("/lookup", h.lookupChapter)
	})
}

func decodeBody(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// --- Series ---

func (h *CatalogHandler) listSeries(w http.ResponseWriter, r *http.Request) {
	list, err := h.seriesSvc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *CatalogHandler) createSeries(w http.ResponseWriter, r *http.Request) {
	var in services.CreateSeriesInput
	if err := decodeBody(r, &in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	st, err := h.seriesSvc.Create(r.Context(), in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, st)
}

func (h *CatalogHandler) getSeries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	st, err := h.seriesSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "serie nao encontrada", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (h *CatalogHandler) updateSeries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in services.UpdateSeriesInput
	if err := decodeBody(r, &in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	st, err := h.seriesSvc.Update(r.Context(), id, in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (h *CatalogHandler) deleteSeries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.seriesSvc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Chapters ---

func (h *CatalogHandler) listChapters(w http.ResponseWriter, r *http.Request) {
	list, err := h.chapterSvc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *CatalogHandler) createChapter(w http.ResponseWriter, r *http.Request) {
	var in services.CreateChapterInput
	if err := decodeBody(r, &in); err != nil {
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

func (h *CatalogHandler) getChapter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := h.chapterSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "capitulo nao encontrado", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

func (h *CatalogHandler) updateChapter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in services.UpdateChapterInput
	if err := decodeBody(r, &in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	ch, err := h.chapterSvc.Update(r.Context(), id, in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

func (h *CatalogHandler) deleteChapter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.chapterSvc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CatalogHandler) lookupChapter(w http.ResponseWriter, r *http.Request) {
	seriesSlug := r.URL.Query().Get("series_slug")
	numberStr := r.URL.Query().Get("number")
	if seriesSlug == "" || numberStr == "" {
		http.Error(w, "series_slug e number sao obrigatorios", http.StatusBadRequest)
		return
	}
	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		http.Error(w, "number invalido", http.StatusBadRequest)
		return
	}
	ch, err := h.chapterSvc.LookupBySeriesAndNumber(r.Context(), seriesSlug, number)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "capitulo nao encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         ch.ID,
		"series_id":  ch.SeriesID,
		"number":     ch.Number,
		"title":      ch.Title,
		"state":      ch.State,
		"access":     ch.Access,
	})
}


