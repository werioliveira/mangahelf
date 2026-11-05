package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/mangashelf/mangashelf/internal/adapters/httpapi"
	"github.com/mangashelf/mangashelf/internal/adapters/storage/local"
	"github.com/mangashelf/mangashelf/internal/adapters/storage/s3"
	"github.com/mangashelf/mangashelf/internal/config"
	"github.com/mangashelf/mangashelf/internal/services"
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

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	if db != nil {
		authSvc := services.NewAuthService(db, cfg)
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
