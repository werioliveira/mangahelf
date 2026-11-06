package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/alexedwards/scs/v2"
	_ "github.com/lib/pq"
	"github.com/mangashelf/mangashelf/internal/adapters/httpapi"
	"github.com/mangashelf/mangashelf/internal/adapters/search/meilisearch"
	"github.com/mangashelf/mangashelf/internal/adapters/search/postgres"
	"github.com/mangashelf/mangashelf/internal/adapters/storage/local"
	"github.com/mangashelf/mangashelf/internal/adapters/storage/s3"
	"github.com/mangashelf/mangashelf/internal/config"
	"github.com/mangashelf/mangashelf/internal/services"
	"github.com/mangashelf/mangashelf/internal/services/oidc"
	"github.com/mangashelf/mangashelf/internal/ports"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config invalido", "err", err)
		os.Exit(1)
	}

	var db *sql.DB
	if cfg.DatabaseURL != "" {
		var openErr error
		db, openErr = sql.Open("postgres", cfg.DatabaseURL)
		if openErr != nil {
			slog.Error("falha ao abrir banco", "err", openErr)
			os.Exit(1)
		}
		defer db.Close()
		if err := db.Ping(); err != nil {
			slog.Error("falha ao pingar banco", "err", err)
			os.Exit(1)
		}
		slog.Info("banco conectado")
	}

	var storage ports.Storage
	switch cfg.StorageDriver {
	case "s3":
		st, err := s3.New(s3.S3Config{
			Endpoint:        cfg.S3Endpoint,
			Region:          cfg.S3Region,
			Bucket:          cfg.S3Bucket,
			AccessKey:       cfg.S3AccessKey,
			SecretKey:       cfg.S3SecretKey,
			ForcePathStyle:  cfg.S3ForcePathStyle,
		})
		if err != nil {
			slog.Error("falha ao inicializar storage s3", "err", err)
			os.Exit(1)
		}
		storage = st
		slog.Info("storage s3 inicializado", "bucket", cfg.S3Bucket, "endpoint", cfg.S3Endpoint)
	case "local":
		storage = local.New(cfg.DataDir)
		slog.Info("storage local inicializado", "dir", cfg.DataDir)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	sm := scs.New()
	sm.Cookie.Name = "ms_session"
	sm.Cookie.Path = "/"
	sm.Cookie.Secure = false
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	r.Use(sm.LoadAndSave)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	if db != nil {
		authSvc := services.NewAuthService(db, sm)
		authHandler := httpapi.NewAuthHandler(authSvc)
		authHandler.RegisterRoutes(r)
		slog.Info("rotas de auth registradas")

		seriesSvc := services.NewSeriesService(db)
		chapterSvc := services.NewChapterService(db)
		catalogHandler := httpapi.NewCatalogHandler(seriesSvc, chapterSvc)
		catalogHandler.RegisterRoutes(r)
		slog.Info("rotas de catalogo registradas")

		mediaSvc := services.NewMediaService(db, storage, cfg, authSvc)
		mediaHandler := httpapi.NewMediaHandler(mediaSvc, authSvc)
		mediaHandler.RegisterRoutes(r)
		slog.Info("rotas de upload registradas")

		pageSvc := services.NewPageService(db)
		imageSvc := services.NewImageURLBuilder(cfg)
		accessSvc := services.NewAccessResolver(db)
		readerHandler := httpapi.NewReaderHandler(pageSvc, imageSvc, authSvc, accessSvc)
		readerHandler.RegisterRoutes(r)
		slog.Info("rotas de leitor registradas")

		vipSvc := services.NewVipService(db)
		vipHandler := httpapi.NewVipHandler(vipSvc, authSvc)
		vipHandler.RegisterRoutes(r)
		slog.Info("rotas de vip registradas")

		settingsSvc := services.NewSettingService(db)
		settingsHandler := httpapi.NewSettingsHandler(settingsSvc, authSvc)
		settingsHandler.RegisterRoutes(r)
		slog.Info("rotas de configuracoes registradas")

		adminSvc := services.NewAdminService(db)
		adminHandler := httpapi.NewAdminHandler(adminSvc, authSvc, chapterSvc)
		adminHandler.RegisterRoutes(r)
		slog.Info("rotas de admin registradas")

		// OIDC / OAuth2
		authService := services.NewAuthService(db, sm)
		oidcSvc := oidc.NewAuthService(db, authService)
		oidcHandler := httpapi.NewOIDCHandler(oidcSvc, authService)
		oidcHandler.RegisterRoutes(r)

		// Registrar provedores OAuth/OIDC se configurados
		if cfg.OIDCGoogleClientID != "" && cfg.OIDCGoogleClientSecret != "" {
			googleProvider := oidc.GoogleConfig(cfg.OIDCGoogleClientID, cfg.OIDCGoogleClientSecret)
			oidc.RegisterProvider("google", googleProvider)
			slog.Info("provedor OIDC Google registrado")
		}
		if cfg.OIDCGitHubClientID != "" && cfg.OIDCGitHubClientSecret != "" {
			githubProvider := oidc.GitHubConfig(cfg.OIDCGitHubClientID, cfg.OIDCGitHubClientSecret)
			oidc.RegisterProvider("github", githubProvider)
			slog.Info("provedor OIDC GitHub registrado")
		}
		slog.Info("rotas de OIDC registradas")

		// Busca: escolher engine baseado na config
		var searchEngine ports.SearchEngine = postgres.New(db)
		if searchEngineSetting, _, err := settingsSvc.Get(context.Background(), "search_engine"); err == nil {
			var engine string
			if err := json.Unmarshal(searchEngineSetting.Value, &engine); err == nil && engine == "meilisearch" {
				meiliCfg := meilisearch.Config{
					Host:   cfg.MeilisearchURL,
					APIKey: cfg.MeilisearchAPIKey,
					Index:  "mangashelf",
				}
				meili, err := meilisearch.New(meiliCfg)
				if err != nil {
					slog.Warn("falha ao inicializar meilisearch, caindo para postgres", "err", err)
				} else {
					searchEngine = meili
					slog.Info("busca meilisearch inicializada", "host", cfg.MeilisearchURL)
				}
			}
		}

		searchSvc := services.NewSearchService(db, searchEngine)
		searchHandler := httpapi.NewSearchHandler(searchSvc)
		searchHandler.RegisterRoutes(r)
		slog.Info("rotas de busca registradas")
	} else {
		slog.Warn("DATABASE_URL nao definida; rotas de auth e catalogo indisponiveis")
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	go func() {
		slog.Info("api ouvindo", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("api falhou", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("desligamento falhou", "err", err)
	}
	slog.Info("api desligado")
}
