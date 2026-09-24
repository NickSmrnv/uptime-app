package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	MonitorConcurrency int
	Address            string
	DatabaseURL        string
	JWTSecret          []byte
	JWTIssuer          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	CookieSecure       bool
	AppEnvironment     string
	CORSAllowedOrigin  string
	UploadStorageDir   string
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLife      time.Duration
}

// Load rejects unsafe or unusable environment combinations before the application opens resources.
func Load() (Config, error) {
	accessTTL, err := durationEnv("ACCESS_TOKEN_TTL", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	refreshTTL, err := durationEnv("REFRESH_TOKEN_TTL", 30*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	cookieSecure, err := boolEnv("COOKIE_SECURE", true)
	if err != nil {
		return Config{}, err
	}
	maxOpen, err := intEnv("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return Config{}, err
	}
	maxIdle, err := intEnv("DB_MAX_IDLE_CONNS", 10)
	if err != nil {
		return Config{}, err
	}
	connMaxLife, err := durationEnv("DB_CONN_MAX_LIFETIME", time.Hour)
	if err != nil {
		return Config{}, err
	}
	monitorConcurrency, err := intEnv("MONITOR_CONCURRENCY", 20)
	if err != nil {
		return Config{}, err
	}
	if monitorConcurrency < 1 || monitorConcurrency > 1000 {
		return Config{}, fmt.Errorf("MONITOR_CONCURRENCY must be between 1 and 1000")
	}
	cfg := Config{MonitorConcurrency: monitorConcurrency, Address: stringEnv("HTTP_ADDR", ":8080"), DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")), JWTSecret: []byte(strings.TrimSpace(os.Getenv("JWT_SECRET"))), JWTIssuer: stringEnv("JWT_ISSUER", "uptime-api"), AccessTokenTTL: accessTTL, RefreshTokenTTL: refreshTTL, CookieSecure: cookieSecure, AppEnvironment: stringEnv("APP_ENV", "development"), CORSAllowedOrigin: stringEnv("CORS_ALLOWED_ORIGIN", "http://localhost:3000"), UploadStorageDir: stringEnv("UPLOAD_STORAGE_DIR", "uploads"), DBMaxOpenConns: maxOpen, DBMaxIdleConns: maxIdle, DBConnMaxLife: connMaxLife}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.UploadStorageDir == "" {
		return Config{}, fmt.Errorf("UPLOAD_STORAGE_DIR must not be empty")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	if cfg.AccessTokenTTL <= 0 || cfg.RefreshTokenTTL <= 0 {
		return Config{}, fmt.Errorf("token TTL values must be positive")
	}
	if cfg.DBMaxOpenConns < 1 || cfg.DBMaxIdleConns < 0 || cfg.DBConnMaxLife <= 0 {
		return Config{}, fmt.Errorf("database pool settings are invalid")
	}
	if cfg.AppEnvironment == "production" && !cfg.CookieSecure {
		return Config{}, fmt.Errorf("COOKIE_SECURE must be enabled in production")
	}
	if cfg.AppEnvironment == "production" && cfg.CORSAllowedOrigin == "http://localhost:3000" {
		return Config{}, fmt.Errorf("CORS_ALLOWED_ORIGIN must be set in production")
	}
	return cfg, nil
}

func stringEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func boolEnv(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func intEnv(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}
