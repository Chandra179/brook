package router_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"brook/router"
)

func TestRecoveryKeepsPanicAndHeadersOutOfLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var defaultRecoveryOutput bytes.Buffer
	previousWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &defaultRecoveryOutput
	t.Cleanup(func() { gin.DefaultErrorWriter = previousWriter })

	var output bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&output), zap.DebugLevel)
	log := zap.New(core)
	engine := router.NewDependencies(&router.DependenciesConfig{
		Logger: log,
		Example: func(*gin.Context) {
			panic("private panic value")
		},
	}).New()

	request := httptest.NewRequest(http.MethodPost, "/example", strings.NewReader("private request body"))
	request.Header.Set("X-Request-ID", "request-123")
	request.Header.Set("X-Other-Secret", "private header value")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	line := output.String()
	for _, want := range []string{"panic recovered", "request-123", "POST", "/example", "stack"} {
		if !strings.Contains(line, want) {
			t.Fatalf("log = %q, missing %q", line, want)
		}
	}
	for _, private := range []string{"private panic value", "private header value", "private request body"} {
		if strings.Contains(line, private) || strings.Contains(defaultRecoveryOutput.String(), private) {
			t.Fatalf("private value %q appeared in recovery output", private)
		}
	}
	if defaultRecoveryOutput.Len() != 0 {
		t.Fatalf("Gin wrote a separate recovery log: %q", defaultRecoveryOutput.String())
	}
}
