package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mangashelf/mangashelf/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config invalido", "err", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("worker iniciado", "driver", cfg.StorageDriver, "pipeline", cfg.ImagePipeline)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("worker desligando")
	<-time.After(time.Second)
	logger.Info("worker desligado")
}
