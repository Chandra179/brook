package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	servermocks "brook/mocks/server"
	"brook/store"
)

func TestReadinessHandlerChecksDependencies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := store.NewSQLite(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	r := gin.New()
	readyStore := servermocks.NewMockReadinessStore(t)
	readyStore.EXPECT().IsClosed().Return(false)
	r.GET("/ready", readinessHandler(db, readyStore))

	res := httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("ready status = %d, want %d", res.Code, http.StatusOK)
	}

	r = gin.New()
	closedStore := servermocks.NewMockReadinessStore(t)
	closedStore.EXPECT().IsClosed().Return(true)
	r.GET("/ready", readinessHandler(db, closedStore))
	res = httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("not-ready status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}
