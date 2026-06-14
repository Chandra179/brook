package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"brook/config"
	"brook/middleware"
	"brook/modules/example"
)

func TestServerSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Logger: config.LoggerConfig{Level: "prod"},
		Middleware: config.MiddlewareConfig{
			RequestLog:   config.RequestLogConfig{LogQuery: true},
			TimeoutInSec: 30,
		},
	}

	logger := cfg.Logger.New()
	mdlw := middleware.NewDependencies(logger)
	exampleMod := example.NewDependencies(logger)

	r := gin.New()
	r.Use(
		middleware.RequestID,
		mdlw.RequestLog(cfg.Middleware.RequestLog),
		middleware.Timeout(time.Duration(cfg.Middleware.TimeoutInSec)*time.Second),
	)
	r.POST("/example", exampleMod.Handle)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/example", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if id := w.Header().Get("X-Request-ID"); id == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}
