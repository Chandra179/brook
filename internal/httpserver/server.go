package httpserver

import (
	"context"
	"net/http"
	"time"

	"brook/internal/middleware"

	"github.com/Chandra179/gosdk/logger"
)

func Server() {
	log := logger.NewLogger("dev")
	deps := middleware.NewDependencies(log)

	globalChain := func(h http.Handler) http.Handler {
		return middleware.Chain(h,
			deps.Recovery(),
			middleware.RequestID,
			middleware.Timeout(middleware.TimeoutConfig{Duration: 30 * time.Second}),
		)
	}

	mux := http.NewServeMux()
	mux.Handle("POST /orders", globalChain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
	))

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      globalChain(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 35 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Info(context.Background(), "http server starting", logger.Field{Key: "addr", Value: srv.Addr})
	if err := srv.ListenAndServe(); err != nil {
		log.Error(context.Background(), "http server error", logger.Field{Key: "error", Value: err.Error()})
	}
}
