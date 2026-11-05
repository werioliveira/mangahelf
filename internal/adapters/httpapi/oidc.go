package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mangashelf/mangashelf/internal/services"
	"github.com/mangashelf/mangashelf/internal/services/oidc"
)

type OIDCHandler struct {
	oidcSvc   *oidc.AuthService
	authSvc   *services.AuthService
}

func NewOIDCHandler(oidcSvc *oidc.AuthService, authSvc *services.AuthService) *OIDCHandler {
	return &OIDCHandler{oidcSvc: oidcSvc, authSvc: authSvc}
}

func (h *OIDCHandler) RegisterRoutes(r chi.Router) {
	r.Get("/auth/oidc/{provider}/login", h.handleOIDCLogin)
	r.Get("/auth/oidc/{provider}/callback", h.handleOIDCCallback)
}

func (h *OIDCHandler) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, ok := oidc.GetProvider(providerName)
	if !ok {
		http.Error(w, "provedor não encontrado", http.StatusNotFound)
		return
	}

	state, err := oidc.GenerateState()
	if err != nil {
		http.Error(w, "erro ao gerar estado", http.StatusInternalServerError)
		return
	}

	next := r.URL.Query().Get("next")
	if next == "" {
		next = "/"
	}
	oidc.StoreState(state, next)

	authURL := provider.AuthURL(state, r.Host+"/auth/oidc/"+providerName+"/callback")
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *OIDCHandler) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, ok := oidc.GetProvider(providerName)
	if !ok {
		http.Error(w, "provedor não encontrado", http.StatusNotFound)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		http.Error(w, "erro na autenticação: "+errorParam, http.StatusBadRequest)
		return
	}

	if code == "" || state == "" {
		http.Error(w, "código ou estado ausente", http.StatusBadRequest)
		return
	}

	next, ok := oidc.GetAndDeleteState(state)
	if !ok {
		http.Error(w, "estado inválido ou expirado", http.StatusBadRequest)
		return
	}

	accessToken, err := provider.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "erro ao trocar código por token", http.StatusInternalServerError)
		return
	}

	userInfo, err := provider.GetUserInfo(r.Context(), accessToken)
	if err != nil {
		http.Error(w, "erro ao obter informações do usuário", http.StatusInternalServerError)
		return
	}

	user, err := h.oidcSvc.FindOrCreateOIDCUser(r.Context(), providerName, userInfo)
	if err != nil {
		http.Error(w, "erro ao criar/atualizar usuário", http.StatusInternalServerError)
		return
	}

	if err := h.authSvc.SaveSession(w, r, user.ID); err != nil {
		http.Error(w, "erro ao criar sessão", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, next, http.StatusTemporaryRedirect)
}
