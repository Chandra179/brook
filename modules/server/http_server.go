package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"

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

	r := gin.New()
	r.SetTrustedProxies(cfg.Middleware.RealIP.TrustedProxies)

	r.Use(
		gin.CustomRecovery(func(c *gin.Context, err any) {
			logger.Error(c.Request.Context(), "panic recovered",
				zlogger.Field{Key: "panic", Value: fmt.Sprintf("%v", err)},
				zlogger.Field{Key: "stack", Value: string(debug.Stack())},
			)
			c.AbortWithStatus(http.StatusInternalServerError)
		}),
		middleware.RequestID,
		mdlw.RequestLog(cfg.Middleware.RequestLog),
		middleware.Timeout(time.Duration(cfg.Middleware.TimeoutInSec)*time.Second),
	)

	r.POST("/example", example.Handle)

	addr := fmt.Sprintf(":%s", cfg.Example.HTTP.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Example.HTTP.ReadTimeoutInSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Example.HTTP.WriteTimeoutInSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.Example.HTTP.IdleTimeoutInSec) * time.Second,
	}

	logger.Info(context.Background(), "starting HTTP server", zlogger.Field{Key: "addr", Value: addr})
	if err := srv.ListenAndServe(); err != nil {
		logger.Error(context.Background(), "server error", zlogger.Field{Key: "error", Value: err.Error()})
	}
}
