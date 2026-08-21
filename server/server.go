package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grafana/pyroscope-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"brook/config"
	_ "brook/docs"
	"brook/logger"
	"brook/middleware"
	"brook/modules/example"
	"brook/modules/foo"
	"brook/store"
	"brook/tracing"
)

func RunHttpServer() {
	appEnvironment := os.Getenv("APP_ENVIRONMENT")

	configPath := "config/config_prd.yaml"
	if appEnvironment == "dev" {
		configPath = "config/config_dev.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if cfg.Profiling.Enabled {
		runtime.SetMutexProfileFraction(5)
		runtime.SetBlockProfileRate(5)

		profiler, pErr := pyroscope.Start(pyroscope.Config{
			ApplicationName: cfg.Profiling.ApplicationName,
			ServerAddress:   cfg.Profiling.ServerAddress,
			Logger:          pyroscope.StandardLogger,
			ProfileTypes: []pyroscope.ProfileType{
				pyroscope.ProfileCPU,
				pyroscope.ProfileAllocObjects,
				pyroscope.ProfileAllocSpace,
				pyroscope.ProfileInuseObjects,
				pyroscope.ProfileInuseSpace,
				pyroscope.ProfileGoroutines,
				pyroscope.ProfileMutexCount,
				pyroscope.ProfileMutexDuration,
				pyroscope.ProfileBlockCount,
				pyroscope.ProfileBlockDuration,
			},
		})
		if pErr != nil {
			log.Fatalf("start pyroscope: %v", pErr)
		}
		defer func() { _ = profiler.Stop() }()
	}

	logger, err := logger.NewLogger(appEnvironment)
	if err != nil {
		log.Fatalf("new logger: %v", err)
	}

	if cfg.Tracing.Enabled {
		tp, tErr := tracing.NewProvider(context.Background(), cfg.Tracing)
		if tErr != nil {
			log.Fatalf("start tracing: %v", tErr)
		}
		otel.SetTracerProvider(tp)
		defer func() { _ = tp.Shutdown(context.Background()) }()
	}

	pool, err := store.NewPool(context.Background(), cfg.Postgres)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	mdlw := middleware.NewDependencies(logger, registry)
	exampleDeps := example.NewDependencies(&example.DependenciesConfig{
		Logger: logger,
		Store:  example.NewPostgresStore(pool),
	})
	fooDeps := foo.NewDependencies(&foo.DependenciesConfig{
		Logger:  logger,
		Example: exampleDeps,
	})

	if appEnvironment == "dev" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	_ = r.SetTrustedProxies(cfg.Middleware.RealIP.TrustedProxies)

	r.Use(
		gin.CustomRecovery(func(c *gin.Context, err any) {
			logger.Error("panic recovered",
				zap.String("panic", fmt.Sprintf("%v", err)),
				zap.String("stack", string(debug.Stack())),
			)
			c.AbortWithStatus(http.StatusInternalServerError)
		}),
		otelgin.Middleware(cfg.Tracing.ApplicationName),
		middleware.RequestID,
		mdlw.RequestLog(cfg.Middleware.RequestLog),
		mdlw.Metrics(),
	)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.POST("/example", exampleDeps.HandleExample)
	r.POST("/foo", fooDeps.HandleFoo)

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
