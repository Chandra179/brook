package order

import (
	"context"
	"log"
	"net/http"

	"github.com/Chandra179/gosdk/logger"

	"brook/config"
	"brook/middleware"
)

func RunHttpServer() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	appLog := logger.NewLogger(cfg.Middleware.Logger.Level)
	deps := middleware.NewDependencies(appLog)

	mux := http.NewServeMux()
	// mux.HandleFunc("POST /orders", HandleCreateOrder)

	chain := middleware.Chain(
		mux,
		deps.Recovery(),
		middleware.RequestID,
		middleware.Timeout(middleware.TimeoutConfig{Duration: cfg.Middleware.Timeout}),
	)

	srv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      chain,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	appLog.Info(context.Background(), "starting HTTP server", logger.Field{Key: "addr", Value: cfg.HTTP.Addr})
	if err := srv.ListenAndServe(); err != nil {
		appLog.Error(context.Background(), "server error", logger.Field{Key: "error", Value: err.Error()})
	}
}
