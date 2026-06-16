package helpers

import (
	"testing"

	"github.com/gin-gonic/gin"

	"brook/config"
	"brook/middleware"
)

func NewTestEngine(t *testing.T, cfg *config.Config) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	logger := cfg.Logger.New()
	mdlw := middleware.NewDependencies(logger)

	r := gin.New()
	r.Use(
		middleware.RequestID,
		mdlw.RequestLog(cfg.Middleware.RequestLog),
	)
	return r
}

func NewTestConfig() *config.Config {
	return &config.Config{
		Logger: config.LoggerConfig{Level: "prod"},
		Middleware: config.MiddlewareConfig{
			RequestLog: config.RequestLogConfig{LogQuery: true},
		},
	}
}
