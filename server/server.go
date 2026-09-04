package server

import (
	"context"
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

	// ---- Config ----
	configPath := "config/config_prd.yaml"
	if appEnvironment == "dev" {
		configPath = "config/config_dev.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := logger.NewLogger(appEnvironment)
	if err != nil {
		log.Fatalf("new logger: %v", err)
	}

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
		Logger:     logger,
		RequestLog: mdlw.RequestLog(cfg.Middleware.RequestLog),
		Example:    exampleDeps.HandleExample,
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
