package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMetrics(t *testing.T) {
	type args struct {
		serviceName string
	}
	tests := []struct {
		name string
		args args
		want *Metrics
	}{
		{
			name: "Default Metrics",
			args: args{
				serviceName: "test_service",
			},
			want: &Metrics{
				Registry: prometheus.NewRegistry(),
				CreateRequests: prometheus.NewCounter(
					prometheus.CounterOpts{
						Namespace: "test_service",
						Name:      "create_requests_total",
						Help:      "Total number of create requests",
					},
				),
				CreateErrors: prometheus.NewCounter(
					prometheus.CounterOpts{
						Namespace: "test_service",
						Name:      "create_errors_total",
						Help:      "Total number of create errors",
					},
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewMetrics(tt.args.serviceName)
			if got.CreateErrors.Desc().String() != tt.want.CreateErrors.Desc().String() {
				t.Errorf("NewMetrics() = %v, want %v", got, tt.want)
			}
			if got.CreateRequests.Desc().String() != tt.want.CreateRequests.Desc().String() {
				t.Errorf("NewMetrics() = %v, want %v", got, tt.want)
			}
		})
	}
}
