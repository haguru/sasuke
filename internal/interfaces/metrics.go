package interfaces

import "github.com/prometheus/client_golang/prometheus"

type Metrics interface {
	GetRegistry() *prometheus.Registry
	IncCounter(name string)
	AddCounter(name string, value float64)
	ObserveHistogram(name string, value float64)
	AddGauge(name string, value float64)
	SetGauge(name string, value float64)
	IncGauge(name string)
	DecGauge(name string)
	SubGauge(name string, value float64)
	SetCurrentTimeGauge(name string)
	ObserveHistogramVec(name string, value float64, labels ...string)
	IncCounterVec(name string, labels ...string)
	AddCounterVec(name string, value float64, labels ...string)
	SetGaugeVec(name string, value float64, labels ...string)
	IncGaugeVec(name string, labels ...string)
	DecGaugeVec(name string, labels ...string)
	// RegisterCounter registers a new counter metric.
	RegisterCounter(name, help string)
	// RegisterCounterVec registers a new counter metric with labels.
	RegisterCounterVec(name, help string, labels []string)
	// RegisterHistogram registers a new histogram metric.
	RegisterHistogram(name, help string, buckets []float64)
	// RegisterHistogramVec registers a new histogram metric with labels.
	RegisterHistogramVec(name, help string, buckets []float64, labels []string)
	// RegisterGauge registers a new gauge metric.
	RegisterGauge(name, help string)
	// RegisterGaugeVec registers a new gauge metric with labels.
	RegisterGaugeVec(name, help string, labels []string)
}
