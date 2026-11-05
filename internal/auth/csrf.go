package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
)

type CSRFConfig struct {
	AllowedOrigins map[string]struct{}
	TrustedProxies []string
}

func CSRFMiddleware(cfg Config, sm *scs.SessionManager) func(http.Handler) http.Handler {
	allowed := cfg.AllowedOrigins
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			if s := r.Header.Get("Sec-Fetch-Site"); s != "" && s != "same-origin" && s != "none" {
				http.Error(w, "CSRF: invalid fetch site", http.StatusForbidden)
				return
			}

			if o := r.Header.Get("Origin"); o != "" {
				if _, ok := allowed[o]; !ok {
					http.Error(w, "CSRF: invalid origin", http.StatusForbidden)
					return
				}
			}

			if strings.HasPrefix(r.URL.Path, "/admin/") {
				if r.Header.Get("Sec-Fetch-Site") == "cross-site" || r.Header.Get("X-MS-Admin") != "1" {
					http.Error(w, "CSRF: admin requires same-origin", http.StatusForbidden)
					return
				}
			}

			if _, authed := PrincipalFromContext(r.Context()); authed {
				want := sm.GetString(r.Context(), "csrf")
				got := r.Header.Get("X-CSRF-Token")
				if want == "" || subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
					http.Error(w, "CSRF: invalid token", http.StatusForbidden)
					return
				}
			} else if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				http.Error(w, "CSRF: invalid content type", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CSRFTokenHandler(sm *scs.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := sm.GetString(r.Context(), "csrf")
		if token == "" {
			http.Error(w, "no session", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"csrf_token":"` + token + `"}`))
	}
}