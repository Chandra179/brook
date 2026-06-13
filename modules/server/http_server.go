package server

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"brook/config"
	"brook/middleware"
	"brook/modules/example"
)

func newLogger(level string) *zap.Logger {
	if level == "dev" {
		l, _ := zap.NewDevelopment()
		return l
	}
	l, _ := zap.NewProduction()
	return l
}

func RunHttpServer() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := newLogger(cfg.Middleware.Logger.Level)
	mdlw := middleware.NewDependencies(logger)

	r := gin.New()
	r.SetTrustedProxies(cfg.Middleware.RealIP.TrustedProxies)

	r.Use(
		gin.CustomRecovery(func(c *gin.Context, err any) {
			logger.Error("panic recovered",
				zap.String("panic", fmt.Sprintf("%v", err)),
				zap.String("stack", string(debug.Stack())),
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

	logger.Info("starting HTTP server", zap.String("addr", addr))
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server error", zap.Error(err))
	}
}
