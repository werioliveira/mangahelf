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
	"github.com/mangashelf/mangashelf/internal/config"
	"github.com/mangashelf/mangashelf/internal/services"
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
	} else {
		slog.Warn("DATABASE_URL nao definida; rotas de auth indisponiveis")
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
