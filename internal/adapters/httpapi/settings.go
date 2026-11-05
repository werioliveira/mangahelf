package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mangashelf/mangashelf/internal/services"
)

type SettingsHandler struct {
	settingsSvc *services.SettingService
	authSvc     *services.AuthService
}

func NewSettingsHandler(settingsSvc *services.SettingService, authSvc *services.AuthService) *SettingsHandler {
	return &SettingsHandler{
		settingsSvc: settingsSvc,
		authSvc:     authSvc,
	}
}

func (h *SettingsHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin/settings", func(r chi.Router) {
		r.Use(h.requireStaffMiddleware)
		r.Get("/", h.listSettings)
		r.Patch("/{key}", h.updateSetting)
	})
}

func (h *SettingsHandler) requireStaffMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := h.authSvc.UserIDFromSession(r)
		if !ok {
			http.Error(w, "nao autenticado", http.StatusUnauthorized)
			return
		}
		// TODO: verificar isStaff via SettingService.IsStaff quando o deadlock do DB for resolvido
		_ = userID
		next.ServeHTTP(w, r)
	})
}

type settingUpdateInput struct {
	Value        interface{} `json:"value"`
	Type         string      `json:"type"`
	SettingGroup string      `json:"setting_group"`
}

func (h *SettingsHandler) listSettings(w http.ResponseWriter, r *http.Request) {
	group := r.URL.Query().Get("group")
	var settings []services.Setting
	var err error
	if group != "" {
		settings, err = h.settingsSvc.GetGroup(r.Context(), group)
	} else {
		for _, g := range []string{"site", "search", "ui", "internal"} {
			gs, err := h.settingsSvc.GetGroup(r.Context(), g)
			if err != nil {
				continue
			}
			settings = append(settings, gs...)
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"settings": settings,
		"total":    len(settings),
	})
}

func (h *SettingsHandler) updateSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		http.Error(w, "chave obrigatoria", http.StatusBadRequest)
		return
	}

	var in settingUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}

	// Validar tipo
	validTypes := map[string]bool{"string": true, "boolean": true, "color": true, "json": true}
	if in.Type != "" && !validTypes[in.Type] {
		http.Error(w, "tipo invalido (string, boolean, color, json)", http.StatusBadRequest)
		return
	}

	// Validar grupo
	validGroups := map[string]bool{"site": true, "search": true, "ui": true, "internal": true}
	if in.SettingGroup != "" && !validGroups[in.SettingGroup] {
		http.Error(w, "grupo invalido (site, search, ui, internal)", http.StatusBadRequest)
		return
	}

	// Buscar a configuracao existente para preservar tipo/grupo se nao fornecidos
	existing, ok, err := h.settingsSvc.Get(r.Context(), key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "configuracao nao encontrada", http.StatusNotFound)
		return
	}

	typ := in.Type
	if typ == "" {
		typ = existing.Type
	}
	group := in.SettingGroup
	if group == "" {
		group = existing.SettingGroup
	}

	if err := h.settingsSvc.Set(r.Context(), key, typ, group, in.Value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
