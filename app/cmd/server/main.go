package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	appconfig "devops-demo/app/internal/config"
	"devops-demo/app/internal/server"
)

func main() {
	cfg := appconfig.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, cfg); err != nil {
		log.Fatal(err)
	}
}
