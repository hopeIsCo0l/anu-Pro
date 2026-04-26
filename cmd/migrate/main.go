package main

import (
	"log/slog"
	"os"

	"github.com/hopeIsCo0l/anu-pro/internal/platform/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	slog.Info("running migrations", "env", cfg.Env)
	// goose migration runner goes here (M0-A4 ticket)
}
