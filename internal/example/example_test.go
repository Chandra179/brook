package example_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"brook/config"
	"brook/internal/example"
	"brook/middleware"
	"brook/store"
)

func TestHandleExampleMapsErrorsAndKeepsLogsSafe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		body       string
		response   string
		logMessage string
		status     int
		closedDB   bool
	}{
		{name: "malformed JSON", body: `{"name":`, status: http.StatusBadRequest, response: "invalid request", logMessage: "invalid create example request"},
		{name: "private malformed JSON", body: `{"name":"private body"`, status: http.StatusBadRequest, response: "invalid request", logMessage: "invalid create example request"},
		{name: "missing name", body: `{}`, status: http.StatusBadRequest, response: "invalid request", logMessage: "invalid create example request"},
		{name: "reserved name", body: `{"name":"admin"}`, status: http.StatusConflict, response: "name is reserved", logMessage: "name is reserved"},
		{name: "store failure", body: `{"name":"private-name"}`, closedDB: true, status: http.StatusInternalServerError, response: "internal error", logMessage: "create example failed"},
		{name: "created", body: `{"name":"brook"}`, status: http.StatusCreated, response: "brook"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&output), zap.InfoLevel)
			log := zap.New(core)
			db := openMigratedExampleDB(t, filepath.Join(t.TempDir(), "handler.db"))
			if tt.closedDB {
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
			}
			deps := example.NewDependencies(&example.DependenciesConfig{Logger: log, DB: db})
			r := gin.New()
			r.Use(middleware.RequestID, middleware.NewDependencies(log).RequestLog(config.RequestLogConfig{}))
			r.POST("/example", deps.HandleExample)

			req := httptest.NewRequest(http.MethodPost, "/example", strings.NewReader(tt.body))
			req.Header.Set("X-Request-ID", "request-123")
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)

			if response.Code != tt.status || !strings.Contains(response.Body.String(), tt.response) {
				t.Fatalf("response = (%d, %q), want (%d, %q)", response.Code, response.Body.String(), tt.status, tt.response)
			}
			line := output.String()
			if !strings.Contains(line, "request-123") || strings.Contains(line, "private-name") || strings.Contains(line, "private body") || strings.Contains(line, "database is closed") {
				t.Fatalf("unsafe request log: %q", line)
			}
			if tt.logMessage != "" && !strings.Contains(line, tt.logMessage) {
				t.Fatalf("log = %q, missing %q", line, tt.logMessage)
			}
		})
	}
}

func TestCreateExamplePersistsUsingMigration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "examples.db")
	db := openMigratedExampleDB(t, dbPath)
	deps := example.NewDependencies(&example.DependenciesConfig{Logger: zap.NewNop(), DB: db})

	created, err := deps.CreateExample(context.Background(), "brook")
	if err != nil {
		t.Fatalf("CreateExample: %v", err)
	}
	if _, parseErr := uuid.Parse(created.ID); parseErr != nil || created.CreatedAt.IsZero() {
		t.Fatalf("created example has invalid ID or timestamp: %+v", created)
	}
	if closeErr := db.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	reopened, err := store.NewSQLite(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	var name string
	var persistedAt time.Time
	if err := reopened.QueryRowContext(context.Background(), "SELECT name, created_at FROM examples WHERE id = ?", created.ID).Scan(&name, &persistedAt); err != nil || name != "brook" || !persistedAt.Equal(created.CreatedAt) {
		t.Fatalf("persisted example = (%q, %v, %v), want brook and %v", name, persistedAt, err, created.CreatedAt)
	}
}

func TestCreateExampleCanceledWriteDoesNotPersist(t *testing.T) {
	db := openMigratedExampleDB(t, filepath.Join(t.TempDir(), "examples.db"))
	deps := example.NewDependencies(&example.DependenciesConfig{Logger: zap.NewNop(), DB: db})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := deps.CreateExample(ctx, "cancelled"); !errors.Is(err, context.Canceled) {
		t.Fatalf("CreateExample error = %v, want context.Canceled", err)
	}
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT count(*) FROM examples").Scan(&count); err != nil || count != 0 {
		t.Fatalf("row count = (%d, %v), want zero", count, err)
	}
}

func openMigratedExampleDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := store.NewSQLite(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration, err := os.ReadFile(filepath.Join("..", "..", "store", "migrations", "sqlite", "00001_create_examples.sql"))
	if err != nil {
		t.Fatal(err)
	}
	_, up, ok := strings.Cut(string(migration), "-- +goose Up")
	if !ok {
		t.Fatal("migration is missing an Up section")
	}
	up, _, ok = strings.Cut(up, "-- +goose Down")
	if !ok {
		t.Fatal("migration is missing a Down section")
	}
	if _, err := db.ExecContext(context.Background(), up); err != nil {
		t.Fatal(fmt.Errorf("apply example migration: %w", err))
	}
	return db
}
