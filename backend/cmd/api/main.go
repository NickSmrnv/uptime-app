package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/uptime-app/backend/internal/app"
	"github.com/uptime-app/backend/internal/config"
	"github.com/uptime-app/backend/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()
	if err := server.Run(ctx, server.New(cfg.Address, application.Handler)); err != nil {
		log.Fatal(err)
	}
}
