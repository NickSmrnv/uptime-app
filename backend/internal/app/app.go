package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/uptime-app/backend/internal/config"
	"github.com/uptime-app/backend/internal/handler"
	"github.com/uptime-app/backend/internal/repository"
	"github.com/uptime-app/backend/internal/server"
	"github.com/uptime-app/backend/internal/service"
	"github.com/uptime-app/backend/internal/storage"
)

type Application struct {
	Handler http.Handler
	Close   func() error
}

func New(ctx context.Context, cfg config.Config) (Application, error) {
	db, closeDB, err := repository.OpenPostgres(ctx, cfg)
	if err != nil {
		return Application{}, err
	}
	if err := repository.Migrate(ctx, db); err != nil {
		_ = closeDB()
		return Application{}, fmt.Errorf("migrate database: %w", err)
	}
	uploads, err := storage.NewUploadStorage(cfg.UploadStorageDir)
	if err != nil {
		_ = closeDB()
		return Application{}, fmt.Errorf("create upload storage: %w", err)
	}
	users := repository.NewUserRepository(db)
	auth := service.NewAuthenticationService(users, repository.NewSessionRepository(db), uploads, service.AuthConfig{JWTSecret: cfg.JWTSecret, JWTIssuer: cfg.JWTIssuer, AccessTokenTTL: cfg.AccessTokenTTL, RefreshTokenTTL: cfg.RefreshTokenTTL})
	mux := http.NewServeMux()
	handler.NewAuthHandler(auth, handler.CookieConfig{Secure: cfg.CookieSecure, RefreshTokenTTL: cfg.RefreshTokenTTL}).RegisterRoutes(mux)
	handler.NewMonitorHandler(service.NewMonitorService(repository.NewMonitorRepository(db), users, auth)).RegisterRoutes(mux)
	handler.NewUploadHandler(auth, service.NewUploadService(uploads)).RegisterRoutes(mux)
	mux.Handle("GET /uploads/{path...}", uploads)
	return Application{Handler: server.WithAPICORS(mux, cfg.CORSAllowedOrigin), Close: closeDB}, nil
}
