package authz

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mangashelf/mangashelf/internal/auth"
)

var (
	ErrNoPolicy       = errors.New("authz: route has no policy")
	ErrPolicyNotFound = errors.New("authz: policy not found for route")
)

type PolicyKind int

const (
	KindPublic PolicyKind = iota
	KindAuthenticated
	KindPermission
)

type ScopeFn func(r *http.Request) (Scope, error)

type Policy struct {
	Kind       PolicyKind
	Perm       Perm
	Scope      ScopeFn
	MaxAuthAge time.Duration
	NeedMFA    bool
}

func Public() Policy        { return Policy{Kind: KindPublic} }
func Authenticated() Policy { return Policy{Kind: KindAuthenticated} }
func Require(p Perm, sc ScopeFn) Policy {
	return Policy{Kind: KindPermission, Perm: p, Scope: sc}
}

type Router struct {
	*chi.Mux
	authz     *Resolver
	mfaWindow time.Duration
	registry  map[string]Policy
}

func NewRouter(authz *Resolver, mfaWindow time.Duration) *Router {
	return &Router{
		Mux:       chi.NewMux(),
		authz:     authz,
		mfaWindow: mfaWindow,
		registry:  make(map[string]Policy),
	}
}

func (rt *Router) Handle(method, pattern string, pol Policy, h http.HandlerFunc) {
	rt.registry[method+" "+pattern] = pol
	rt.Mux.Method(method, pattern, rt.enforce(pol, h))
}

func (rt *Router) Registry() map[string]Policy {
	return rt.registry
}

func (rt *Router) enforce(pol Policy, h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch pol.Kind {
		case KindPublic:
			h(w, r)
			return
		case KindAuthenticated, KindPermission:
			p, ok := auth.PrincipalFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if pol.Kind == KindPermission {
				sc := GlobalScope()
				if pol.Scope != nil {
					var err error
					if sc, err = pol.Scope(r); err != nil {
						http.Error(w, "not found", http.StatusNotFound)
						return
					}
				}
				subj := SubjectFromPrincipal(p, rt.mfaWindow)
				if !rt.authz.Can(r.Context(), subj, pol.Perm, sc) {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}
			h(w, r)
		}
	})
}

func ScopeFromChapter(repo ChapterRepo) ScopeFn {
	return func(r *http.Request) (Scope, error) {
		id := chi.URLParam(r, "id")
		if id == "" {
			return Scope{}, errors.New("missing chapter id")
		}
		ch, err := repo.Chapter(r.Context(), id)
		if err != nil {
			return Scope{}, err
		}
		return GroupScope(ch.GroupID), nil
	}
}

type ChapterRepo interface {
	Chapter(ctx context.Context, id string) (*Chapter, error)
}

type Chapter struct {
	ID      string
	GroupID uuid.UUID
}