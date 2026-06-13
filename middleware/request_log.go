package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"brook/config"
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
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("duration_ms", duration.Milliseconds()),
		}

		if reqID := GetRequestID(c.Request.Context()); reqID != "" {
			fields = append(fields, zap.String("request_id", reqID))
		}

		if cfg.LogQuery && c.Request.URL.RawQuery != "" {
			fields = append(fields, zap.String("query", c.Request.URL.RawQuery))
		}

		d.logger.Info("request completed", fields...)
	}
}
