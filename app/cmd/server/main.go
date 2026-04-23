package main

import (
	"context"
	"log"
	"log/slog"
	"os/signal"
	"syscall"

	appconfig "devops-demo/app/internal/config"
	"devops-demo/app/internal/server"
)

func main() {
	cfg := appconfig.Load()
	slog.SetDefault(slog.New(slog.NewTextHandler(log.Writer(), &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("starting tiny llm frontend", "addr", cfg.Addr, "catalogPath", cfg.CatalogPath, "defaultService", cfg.DefaultService)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, cfg); err != nil {
		log.Fatal(err)
	}
}
