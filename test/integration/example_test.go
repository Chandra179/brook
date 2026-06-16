//go:build integration

package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"brook/modules/example"
	"brook/test/integration/helpers"
)

func TestExampleHandler_ReturnsOk(t *testing.T) {
	cfg := helpers.NewTestConfig()
	engine := helpers.NewTestEngine(t, cfg)

	exampleMod := example.NewDependencies(cfg.Logger.New())
	engine.POST("/example", exampleMod.Handle)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/example", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
