package middleware

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"brook/config"
	"brook/zlogger"
)

// Request bodies are intentionally NOT logged. After JSON decoding the body
// seen here is not what the client sent (whitespace stripped, keys reordered,
// unknown fields dropped). Logging it provides no debugging value, risks PII
// leakage, and adds complexity. Use structured error responses returned to
// the client instead.

// RequestLog returns Gin middleware that logs every HTTP request with method,
// path, status code, duration, query params, and the request ID from the
// context. Requests matching SkipPaths are passed through without logging.
func (d *Dependencies) RequestLog(cfg config.RequestLogConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, p := range cfg.SkipPaths {
			if c.Request.URL.Path == p {
				c.Next()
				return
			}
		}

		start := time.Now()
		c.Next()

		duration := time.Since(start)
		fields := []zlogger.Field{
			{Key: "method", Value: c.Request.Method},
			{Key: "path", Value: c.Request.URL.Path},
			{Key: "status", Value: strconv.Itoa(c.Writer.Status())},
			{Key: "duration_ms", Value: fmt.Sprintf("%.2f", float64(duration.Milliseconds()))},
		}

		if reqID := GetRequestID(c.Request.Context()); reqID != "" {
			fields = append(fields, zlogger.Field{Key: "request_id", Value: reqID})
		}

		if cfg.LogQuery && c.Request.URL.RawQuery != "" {
			fields = append(fields, zlogger.Field{Key: "query", Value: c.Request.URL.RawQuery})
		}

		d.logger.Info(c.Request.Context(), "request completed", fields...)
	}
}
