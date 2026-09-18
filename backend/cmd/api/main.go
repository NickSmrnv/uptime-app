package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/uptime-app/backend/docs"
	"github.com/uptime-app/backend/internal/app"
	"github.com/uptime-app/backend/internal/config"
	"github.com/uptime-app/backend/internal/server"
)

// @title Uptime API
// @version 1.0
// @description HTTP API for Uptime authentication, profile management, and file uploads.
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and an access token.
// @securityDefinitions.apikey RefreshCookie
// @in header
// @name Cookie
// @description Cookie header containing the HttpOnly refresh_token issued by the authentication endpoints.

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
