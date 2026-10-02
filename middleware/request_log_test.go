package middleware

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"brook/config"
)

func TestRequestLogIncludesCorrelationAndSanitizesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&output),
		zap.InfoLevel,
	)
	deps := NewDependencies(zap.New(core))

	r := gin.New()
	r.Use(RequestID, deps.RequestLog(config.RequestLogConfig{
		LogQuery:       true,
		QueryAllowlist: []string{"page", "token"},
	}))
	r.GET("/items", func(c *gin.Context) {
		_ = c.Error(errors.New("private failure detail"))
		c.Status(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/items?page=2&token=secret&ignored=value", nil)
	req.Header.Set(headerKey, "request-123")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	logLine := output.String()
	for _, want := range []string{"request-123", "/items", "request failed", "page=2", "REDACTED"} {
		if !strings.Contains(logLine, want) {
			t.Errorf("log = %q, missing %q", logLine, want)
		}
	}
	if strings.Contains(logLine, "secret") || strings.Contains(logLine, "ignored=value") || strings.Contains(logLine, "private failure detail") {
		t.Errorf("log leaked query data: %q", logLine)
	}
}

func TestRequestLogUsesOnlyApprovedErrorMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		meta any
		want string
	}{
		{name: "approved", meta: SafeErrorMessage("create example failed"), want: "create example failed"},
		{name: "missing", want: "request failed"},
		{name: "empty", meta: SafeErrorMessage(""), want: "request failed"},
		{name: "plain string", meta: "private metadata", want: "request failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			core := zapcore.NewCore(
				zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
				zapcore.AddSync(&output),
				zap.InfoLevel,
			)
			r := gin.New()
			r.Use(RequestID, NewDependencies(zap.New(core)).RequestLog(config.RequestLogConfig{}))
			r.GET("/items", func(c *gin.Context) {
				_ = c.Error(errors.New("private error detail")).SetMeta(tt.meta)
				c.Status(http.StatusInternalServerError)
			})
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/items", nil))

			line := output.String()
			if !strings.Contains(line, tt.want) || strings.Contains(line, "private error detail") || strings.Contains(line, "private metadata") {
				t.Fatalf("unsafe request log: %q", line)
			}
		})
	}
}
