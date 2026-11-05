package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	serviceAuth "github.com/mangashelf/mangashelf/internal/services/auth"
	"github.com/mangashelf/mangashelf/internal/services/oidc"
)

type OIDCHandler struct {
	oidcSvc   *oidc.AuthService
	authSvc   *serviceAuth.Service
}

func NewOIDCHandler(oidcSvc *oidc.AuthService, authSvc *serviceAuth.Service) *OIDCHandler {
	return &OIDCHandler{oidcSvc: oidcSvc, authSvc: authSvc}
}

func (h *OIDCHandler) RegisterRoutes(mux http.Handler) {
	// Usar chi diretamente para registrar as rotas OIDC
	if chiMux, ok := mux.(*chi.Mux); ok {
		chiMux.Get("/auth/oidc/:provider/login", h.handleOIDC)
		chiMux.Get("/auth/oidc/:provider/callback", h.handleOIDC)
	}
}

func (h *OIDCHandler) handleOIDC(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/auth/oidc/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "provedor inválido", http.StatusBadRequest)
		return
	}

	providerName := parts[0]
	action := parts[1]

	switch action {
	case "login":
		h.handleOIDCLogin(w, r, providerName)
	case "callback":
		h.handleOIDCCallback(w, r, providerName)
	default:
		http.Error(w, "ação inválida", http.StatusBadRequest)
	}
}

func (h *OIDCHandler) handleOIDCLogin(w http.ResponseWriter, r *http.Request, providerName string) {
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

func (h *OIDCHandler) handleOIDCCallback(w http.ResponseWriter, r *http.Request, providerName string) {
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
