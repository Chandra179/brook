package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"brook/config"
	_ "brook/docs"
	"brook/logger"
	"brook/middleware"
	"brook/modules/example"
	"brook/router"
	"brook/store"
)

func RunHttpServer() {
	appEnvironment := os.Getenv("APP_ENVIRONMENT")
	if appEnvironment != "dev" && appEnvironment != "prd" {
		log.Fatalf("APP_ENVIRONMENT must be dev or prd, got %q", appEnvironment)
	}

	// ---- Config ----
	configPath := "config/config_prd.yaml"
	if appEnvironment == "dev" {
		configPath = "config/config_dev.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := logger.NewLogger(
		appEnvironment,
		cfg.Logger.Level,
		cfg.Logger.SamplingInitial,
		cfg.Logger.SamplingThereafter,
	)
	if err != nil {
		log.Fatalf("new logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	// ---- Datastores (SQLite + Badger, both embedded) ----
	db, err := store.NewSQLite(context.Background(), cfg.SQLite.DSN)
	if err != nil {
		log.Fatalf("connect sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	kv, err := store.NewBadger(cfg.Badger.Dir)
	if err != nil {
		log.Fatalf("connect badger: %v", err)
	}
	defer func() { _ = kv.Close() }()

	// ---- Middleware / modules ----
	mdlw := middleware.NewDependencies(logger)
	exampleDeps := example.NewDependencies(&example.DependenciesConfig{
		Logger: logger,
		DB:     db,
	})

	if appEnvironment == "dev" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// ---- Router ----
	routerDeps := router.NewDependencies(&router.DependenciesConfig{
		Logger:           logger,
		RequestLog:       mdlw.RequestLog(cfg.Middleware.RequestLog),
		RequestBodyLimit: middleware.RequestBodyLimit(cfg.HTTP.MaxBodySizeInBytes),
		Readiness:        readinessHandler(db, kv),
		Example:          exampleDeps.HandleExample,
	})
	r := routerDeps.New()

	// ---- HTTP server + graceful shutdown ----
	addr := fmt.Sprintf(":%s", cfg.HTTP.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.HTTP.ReadTimeoutInSec) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTP.WriteTimeoutInSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.HTTP.IdleTimeoutInSec) * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("starting HTTP server", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		if err != nil {
			logger.Fatal("server error", zap.Error(err))
		}
	case <-ctx.Done():
		stop()
		logger.Info("shutdown signal received, draining in-flight requests")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(cfg.HTTP.ShutdownTimeoutInSec)*time.Second,
		)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", zap.Error(err))
			return
		}
		logger.Info("HTTP server stopped")
	}
}

type readinessStore interface {
	IsClosed() bool
}

func readinessHandler(db *sql.DB, kv readinessStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			_ = c.Error(errors.New("sqlite readiness check failed"))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		}
		if kv.IsClosed() {
			_ = c.Error(errors.New("badger readiness check failed"))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
