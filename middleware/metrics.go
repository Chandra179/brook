package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Metrics returns Gin middleware that records a request-duration histogram
// and a request counter, labeled by method, route (c.FullPath(), not the
// raw path — keeps cardinality bounded), and status.
func (d *dependencies) Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())

		d.httpRequestDuration.WithLabelValues(c.Request.Method, route, status).Observe(time.Since(start).Seconds())
		d.httpRequestsTotal.WithLabelValues(c.Request.Method, route, status).Inc()
	}
}
