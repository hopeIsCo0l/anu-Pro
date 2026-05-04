package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hopeIsCo0l/anu-pro/internal/platform/config"
	"github.com/hopeIsCo0l/anu-pro/internal/platform/db"
	"github.com/hopeIsCo0l/anu-pro/internal/platform/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	log := logger.New(cfg)
	slog.SetDefault(log)

	pool, err := db.New(cfg)
	if err != nil {
		slog.Error("db init failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("worker starting", "env", cfg.Env)
	// TODO: wire river worker (M0-B)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("worker shutting down")
}
