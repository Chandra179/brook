package middleware

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type dependencies struct {
	logger              *zap.Logger
	registry            *prometheus.Registry
	httpRequestDuration *prometheus.HistogramVec
	httpRequestsTotal   *prometheus.CounterVec
}

func NewDependencies(logger *zap.Logger, registry *prometheus.Registry) *dependencies {
	httpRequestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "HTTP request duration in seconds.",
	}, []string{"method", "route", "status"})

	httpRequestsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "route", "status"})

	registry.MustRegister(httpRequestDuration, httpRequestsTotal)

	return &dependencies{
		logger:              logger,
		registry:            registry,
		httpRequestDuration: httpRequestDuration,
		httpRequestsTotal:   httpRequestsTotal,
	}
}
