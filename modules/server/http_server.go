package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"brook/config"
	"brook/middleware"
	"brook/modules/example"
	"brook/zlogger"
)

func RunHttpServer() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := zlogger.New(cfg.Middleware.Logger.Level)
	mdlw := middleware.NewDependencies(logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /example", example.Handle)

	chain := middleware.Chain(
		mux,
		mdlw.Recovery(),
		middleware.RequestID,
		middleware.Timeout(middleware.TimeoutConfig{
			Duration: time.Duration(cfg.Middleware.TimeoutInSec) * time.Second,
		}),
	)

	addr := fmt.Sprintf(":%s", cfg.Example.HTTP.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      chain,
		ReadTimeout:  time.Duration(cfg.Example.HTTP.ReadTimeoutInSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Example.HTTP.WriteTimeoutInSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.Example.HTTP.IdleTimeoutInSec) * time.Second,
	}

	logger.Info(context.Background(), "starting HTTP server", zlogger.Field{Key: "addr", Value: addr})
	if err := srv.ListenAndServe(); err != nil {
		logger.Error(context.Background(), "server error", zlogger.Field{Key: "error", Value: err.Error()})
	}
}
