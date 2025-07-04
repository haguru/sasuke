package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics struct defines the structure for the metrics collector
type Metrics struct {
	Registry           *prometheus.Registry
	CreateRequests       prometheus.Counter
	CreateErrors		 prometheus.Counter
	
}

// NewMetrics creates a new Metrics instance
func NewMetrics(serviceName string) *Metrics {
	registry := prometheus.NewRegistry()

	createRequests := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: serviceName,
			Name:      "create_requests_total",
			Help:      "Total number of create requests",
		},
	)

	createErrors := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: serviceName,
			Name:      "create_errors_total",
			Help:      "Total number of create errors",
		},
	)

	registry.MustRegister(createRequests, createErrors)

	return &Metrics{
		Registry:           registry,
		CreateRequests:       createRequests,
		CreateErrors:         createErrors,
	}
}

