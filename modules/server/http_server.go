package server

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"brook/config"
	_ "brook/docs"
	"brook/middleware"
	"brook/modules/example"
)

func RunHttpServer() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := cfg.Logger.New()
	mdlw := middleware.NewDependencies(logger)
	exampleMod := example.NewDependencies(logger)

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

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.POST("/example", exampleMod.Handle)

	addr := fmt.Sprintf(":%s", cfg.HTTP.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.HTTP.ReadTimeoutInSec) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTP.WriteTimeoutInSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.HTTP.IdleTimeoutInSec) * time.Second,
	}

	logger.Info("starting HTTP server", zap.String("addr", addr))
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server error", zap.Error(err))
	}
}
