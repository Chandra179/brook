package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"brook/config"
	_ "brook/docs"
	"brook/internal/example"
	"brook/logger"
	"brook/middleware"
	"brook/router"
	"brook/store"
)

// RunHttpServer starts Brook and returns after the server and its stores stop.
// The caller owns signal handling and process exit.
func RunHttpServer(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	appEnvironment := os.Getenv("APP_ENVIRONMENT")
	if appEnvironment != "dev" && appEnvironment != "prd" {
		return errors.New("APP_ENVIRONMENT must be dev or prd")
	}

	configPath := "config/config_prd.yaml"
	if appEnvironment == "dev" {
		configPath = "config/config_dev.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return runHTTPServer(ctx, cfg, appEnvironment)
}

func runHTTPServer(ctx context.Context, cfg *config.Config, appEnvironment string) (result error) {
	appLogger, err := logger.NewLogger(
		appEnvironment,
		cfg.Logger.Level,
		cfg.Logger.SamplingInitial,
		cfg.Logger.SamplingThereafter,
	)
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}
	defer func() { _ = appLogger.Sync() }()

	db, err := store.NewSQLite(ctx, cfg.SQLite.DSN)
	if err != nil {
		return fmt.Errorf("connect sqlite: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			result = errors.Join(result, fmt.Errorf("close sqlite: %w", closeErr))
		}
	}()

	kv, err := store.NewBadger(cfg.Badger.Dir)
	if err != nil {
		return fmt.Errorf("connect badger: %w", err)
	}
	defer func() {
		if closeErr := kv.Close(); closeErr != nil {
			result = errors.Join(result, fmt.Errorf("close badger: %w", closeErr))
		}
	}()

	mdlw := middleware.NewDependencies(appLogger)
	exampleDeps := example.NewDependencies(&example.DependenciesConfig{
		Logger: appLogger,
		DB:     db,
	})
	if appEnvironment == "dev" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	var draining atomic.Bool
	routerDeps := router.NewDependencies(&router.DependenciesConfig{
		Logger:           appLogger,
		RequestLog:       mdlw.RequestLog(cfg.Middleware.RequestLog),
		RequestBodyLimit: middleware.RequestBodyLimit(cfg.HTTP.MaxBodySizeInBytes),
		Readiness:        readinessHandler(db, kv, &draining),
		Example:          exampleDeps.HandleExample,
	})
	addr := fmt.Sprintf(":%s", cfg.HTTP.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      routerDeps.New(),
		ReadTimeout:  time.Duration(cfg.HTTP.ReadTimeoutInSec) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTP.WriteTimeoutInSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.HTTP.IdleTimeoutInSec) * time.Second,
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen HTTP: %w", err)
	}
	defer func() { _ = listener.Close() }()

	appLogger.Info("starting HTTP server", zap.String("addr", listener.Addr().String()))
	result = serveHTTP(ctx, srv, listener,
		time.Duration(cfg.HTTP.ShutdownTimeoutInSec)*time.Second,
		func() { draining.Store(true) },
	)
	if result == nil {
		appLogger.Info("HTTP server stopped")
	}
	return result
}

func serveHTTP(ctx context.Context, srv *http.Server, listener net.Listener, shutdownTimeout time.Duration, markDraining func()) error {
	serverErr := make(chan error, 1)
	go func() { serverErr <- srv.Serve(listener) }()

	var result error
	serveFinished := false
	select {
	case err := <-serverErr:
		serveFinished = true
		if !errors.Is(err, http.ErrServerClosed) {
			result = fmt.Errorf("serve HTTP: %w", err)
		}
	case <-ctx.Done():
	}

	markDraining()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		result = errors.Join(result, fmt.Errorf("shutdown HTTP: %w", err))
		if closeErr := srv.Close(); closeErr != nil {
			result = errors.Join(result, fmt.Errorf("close HTTP: %w", closeErr))
		}
	}
	if !serveFinished {
		if err := <-serverErr; !errors.Is(err, http.ErrServerClosed) {
			result = errors.Join(result, fmt.Errorf("serve HTTP: %w", err))
		}
	}
	return result
}

type readinessStore interface {
	IsClosed() bool
}

func readinessHandler(db *sql.DB, kv readinessStore, draining *atomic.Bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if draining.Load() {
			_ = c.Error(errors.New("server draining")).SetMeta(middleware.SafeErrorMessage("server draining"))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			_ = c.Error(fmt.Errorf("sqlite readiness: %w", err)).SetMeta(middleware.SafeErrorMessage("sqlite readiness check failed"))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		}
		if kv.IsClosed() {
			_ = c.Error(errors.New("badger readiness check failed")).SetMeta(middleware.SafeErrorMessage("badger readiness check failed"))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
