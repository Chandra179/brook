//go:build integration

package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"brook/test/integration/helpers"
)

func TestMiddlewareStack_GeneratesRequestID(t *testing.T) {
	cfg := helpers.NewTestConfig()
	engine := helpers.NewTestEngine(t, cfg)

	engine.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if id := w.Header().Get("X-Request-ID"); id == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}

func TestMiddlewareStack_ReusesExistingRequestID(t *testing.T) {
	cfg := helpers.NewTestConfig()
	engine := helpers.NewTestEngine(t, cfg)

	engine.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Request-ID", "client-provided-id")
	engine.ServeHTTP(w, req)

	if id := w.Header().Get("X-Request-ID"); id != "client-provided-id" {
		t.Errorf("expected client-provided-id, got %s", id)
	}
}

func TestMiddlewareStack_RequestLogDoesNotPanic(t *testing.T) {
	cfg := helpers.NewTestConfig()
	engine := helpers.NewTestEngine(t, cfg)

	engine.GET("/log-test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/log-test?q=hello", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
