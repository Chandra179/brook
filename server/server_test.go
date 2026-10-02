package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"brook/config"
	servermocks "brook/mocks/server"
	"brook/store"
)

func TestReadinessReportsStoreAndDrainState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := store.NewSQLite(context.Background(), filepath.Join(t.TempDir(), "ready.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	kv := servermocks.NewMockReadinessStore(t)
	kv.EXPECT().IsClosed().Return(false).Once()
	var draining atomic.Bool
	check := readinessHandler(db, kv, &draining)
	if got := readinessStatus(check); got != http.StatusOK {
		t.Fatalf("ready status = %d, want 200", got)
	}
	draining.Store(true)
	if got := readinessStatus(check); got != http.StatusServiceUnavailable {
		t.Fatalf("draining status = %d, want 503", got)
	}

	closedStore := servermocks.NewMockReadinessStore(t)
	closedStore.EXPECT().IsClosed().Return(true).Once()
	if got := readinessStatus(readinessHandler(db, closedStore, new(atomic.Bool))); got != http.StatusServiceUnavailable {
		t.Fatalf("closed Badger status = %d, want 503", got)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if got := readinessStatus(readinessHandler(db, closedStore, new(atomic.Bool))); got != http.StatusServiceUnavailable {
		t.Fatalf("closed SQLite status = %d, want 503", got)
	}
}

func readinessStatus(handler gin.HandlerFunc) int {
	r := gin.New()
	r.GET("/ready", handler)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	return response.Code
}

func TestRunHTTPServerReturnsListenErrorAndReleasesBadger(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = occupied.Close() }()
	cfg := testConfig(t)
	cfg.HTTP.Port = strconv.Itoa(occupied.Addr().(*net.TCPAddr).Port)

	if runErr := runHTTPServer(context.Background(), cfg, "prd"); runErr == nil || !strings.Contains(runErr.Error(), "listen HTTP") {
		t.Fatalf("runHTTPServer error = %v, want listen error", runErr)
	}
	kv, err := store.NewBadger(cfg.Badger.Dir)
	if err != nil {
		t.Fatalf("Badger lock was not released: %v", err)
	}
	if err := kv.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRunHTTPServerReturnsPartialStartupError(t *testing.T) {
	cfg := testConfig(t)
	cfg.Badger.Dir = cfg.SQLite.DSN // A database file cannot be a Badger directory.
	if err := runHTTPServer(context.Background(), cfg, "prd"); err == nil || !strings.Contains(err.Error(), "connect badger") {
		t.Fatalf("runHTTPServer error = %v, want Badger startup error", err)
	}
	// The SQLite database remains usable after the partial startup failed.
	db, err := store.NewSQLite(context.Background(), cfg.SQLite.DSN)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRunHttpServerReturnsInvalidEnvironmentError(t *testing.T) {
	t.Setenv("APP_ENVIRONMENT", "unknown")
	if err := RunHttpServer(context.Background()); err == nil || !strings.Contains(err.Error(), "APP_ENVIRONMENT") {
		t.Fatalf("RunHttpServer error = %v, want environment validation error", err)
	}
}

func TestRunHTTPServerCancellationReleasesBadger(t *testing.T) {
	portListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := portListener.Addr().(*net.TCPAddr).Port
	if closeErr := portListener.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	cfg := testConfig(t)
	cfg.HTTP.Port = strconv.Itoa(port)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runHTTPServer(ctx, cfg, "prd") }()

	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	client := localHTTPClient(t)
	deadline := time.After(5 * time.Second)
	for {
		resp, requestErr := client.Get(url)
		if requestErr == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		select {
		case runErr := <-done:
			t.Fatalf("server stopped before ready: %v", runErr)
		case <-deadline:
			t.Fatal("server never became ready")
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	if runErr := waitError(t, done); runErr != nil {
		t.Fatalf("cancelled server error = %v, want nil", runErr)
	}
	kv, err := store.NewBadger(cfg.Badger.Dir)
	if err != nil {
		t.Fatalf("Badger lock was not released after cancellation: %v", err)
	}
	if err := kv.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestServeHTTPDrainsActiveRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	started := make(chan struct{})
	release := make(chan struct{})
	draining := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = w.Write([]byte("done"))
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- serveHTTP(ctx, &http.Server{Handler: handler}, listener, time.Second, func() { close(draining) })
	}()
	client := localHTTPClient(t)
	requestResult := make(chan error, 1)
	go func() {
		resp, err := client.Get("http://" + listener.Addr().String())
		if err != nil {
			requestResult <- err
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			requestResult <- fmt.Errorf("status %d", resp.StatusCode)
			return
		}
		requestResult <- nil
	}()
	waitSignal(t, started, "active request")
	cancel()
	waitSignal(t, draining, "draining")
	select {
	case err := <-done:
		t.Fatalf("server stopped before active request completed: %v", err)
	default:
	}
	close(release)
	if err := waitError(t, requestResult); err != nil {
		t.Fatal(err)
	}
	if err := waitError(t, done); err != nil {
		t.Fatalf("graceful shutdown error = %v, want nil", err)
	}
}

func TestServeHTTPTimeoutClosesActiveConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	started := make(chan struct{})
	requestCancelled := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(requestCancelled)
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- serveHTTP(ctx, &http.Server{Handler: handler}, listener, 75*time.Millisecond, func() {})
	}()
	client := localHTTPClient(t)
	requestResult := make(chan error, 1)
	go func() {
		resp, err := client.Get("http://" + listener.Addr().String())
		if err == nil {
			_ = resp.Body.Close()
		}
		requestResult <- err
	}()
	waitSignal(t, started, "active request")
	cancel()
	if err := waitError(t, done); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown error = %v, want deadline exceeded", err)
	}
	waitSignal(t, requestCancelled, "request cancellation")
	if err := waitError(t, requestResult); err == nil {
		t.Fatal("active request completed despite forced connection close")
	}
}

func TestServeHTTPReturnsServingFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if closeErr := listener.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	var draining atomic.Bool
	err = serveHTTP(context.Background(), &http.Server{}, listener, time.Second, func() { draining.Store(true) })
	if err == nil || !strings.Contains(err.Error(), "serve HTTP") || !draining.Load() {
		t.Fatalf("serveHTTP = (%v, draining=%v), want serving failure and draining", err, draining.Load())
	}
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	return &config.Config{
		HTTP: config.HTTPConfig{
			Port: "0", ReadTimeoutInSec: 5, WriteTimeoutInSec: 5,
			IdleTimeoutInSec: 5, ShutdownTimeoutInSec: 1, MaxBodySizeInBytes: 1024,
		},
		Logger: config.LoggerConfig{Level: "error", SamplingInitial: 100, SamplingThereafter: 100},
		SQLite: config.SQLiteConfig{DSN: filepath.Join(dir, "brook.db")},
		Badger: config.BadgerConfig{Dir: filepath.Join(dir, "badger")},
	}
}

func localHTTPClient(t *testing.T) *http.Client {
	t.Helper()
	transport := &http.Transport{Proxy: nil}
	t.Cleanup(transport.CloseIdleConnections)
	return &http.Client{Transport: transport, Timeout: 5 * time.Second}
}

func waitSignal(t *testing.T, signal <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
	}
}

func waitError(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for result")
		return nil
	}
}
