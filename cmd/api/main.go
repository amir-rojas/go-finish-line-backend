package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	authjwt "finish-line/internal/auth/adapters/jwt"
	authmiddleware "finish-line/internal/auth/adapters/middleware"
	authpostgres "finish-line/internal/auth/adapters/postgres"
	authrest "finish-line/internal/auth/adapters/rest"
	authservice "finish-line/internal/auth/service"
	"finish-line/internal/common/config"
	"finish-line/internal/common/postgres"
	"finish-line/internal/common/security"
	"finish-line/internal/common/server"
	userpostgres "finish-line/internal/user/adapters/postgres"
	userrest "finish-line/internal/user/adapters/rest"
	userservice "finish-line/internal/user/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger := newLogger(cfg.Env)
	slog.SetDefault(logger)

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := postgres.Connect(cfg.DB.DSN())
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}

	// Shared infrastructure used across modules.
	hasher := security.NewBcryptHasher()
	userRepo := userpostgres.NewRepository(db)
	userSvc := userservice.New(userRepo, hasher)

	// AutoMigrate is a development convenience; production schema changes
	// must ship as explicit, reviewed migrations. Register each module's
	// migration here in dependency order (users before refresh_tokens).
	if !cfg.IsProduction() {
		if err := postgres.RunMigrations(db, userpostgres.Migrate, authpostgres.Migrate); err != nil {
			return fmt.Errorf("running dev migrations: %w", err)
		}
		if err := userSvc.EnsureAdmin(ctx, "Admin", "admin@finishline.dev", "admin.123"); err != nil {
			return fmt.Errorf("seeding admin user: %w", err)
		}
		logger.Info("dev migrations applied and admin ensured")
	}

	// Auth module reuses the user repository (as a UserFinder) and the hasher
	// (as a PasswordVerifier) — the narrow interfaces it actually needs.
	authSvc := authservice.New(
		userRepo,
		hasher,
		authjwt.New(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL),
		authpostgres.NewRepository(db),
		cfg.Auth.RefreshTTL,
	)

	userModule := userrest.NewHandler(userSvc)
	authModule := authrest.NewHandler(authSvc, cfg.Auth.RefreshTTL, cfg.IsProduction())
	authMW := authmiddleware.RequireAuth(authSvc)

	srv := &http.Server{
		Addr: ":" + cfg.AppPort,
		Handler: server.New(logger, db, authMW, server.Modules{
			Public:    []server.RouteRegistrar{authModule},
			Protected: []server.RouteRegistrar{userModule},
		}),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down server: %w", err)
	}

	return nil
}

func newLogger(env string) *slog.Logger {
	if env == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}
