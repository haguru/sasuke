package metrics

import (
	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics is a flexible Prometheus metrics collector.
type Metrics struct {
	Registry      *prometheus.Registry
	counters      map[string]prometheus.Counter
	counterVecs   map[string]*prometheus.CounterVec
	histograms    map[string]prometheus.Histogram
	histogramVecs map[string]*prometheus.HistogramVec
	gauges        map[string]prometheus.Gauge
	gaugeVecs     map[string]*prometheus.GaugeVec
}

// NewMetrics creates a new flexible Metrics instance.
func NewMetrics(serviceName string) interfaces.Metrics {
	registry := prometheus.NewRegistry()
	return &Metrics{
		Registry:      registry,
		counters:      make(map[string]prometheus.Counter),
		histograms:    make(map[string]prometheus.Histogram),
		gauges:        make(map[string]prometheus.Gauge),
		counterVecs:   make(map[string]*prometheus.CounterVec),
		histogramVecs: make(map[string]*prometheus.HistogramVec),
		gaugeVecs:     make(map[string]*prometheus.GaugeVec),
	}
}

// GetRegistry returns the Prometheus registry.
func (m *Metrics) GetRegistry() *prometheus.Registry {
	return m.Registry
}

// RegisterCounter registers a new counter metric.
func (m *Metrics) RegisterCounter(name, help string) {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
	})
	m.Registry.MustRegister(counter)
	m.counters[name] = counter
}

// RegisterCounterVec registers a new counter metric with labels.
func (m *Metrics) RegisterCounterVec(name, help string, labels []string) {
	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labels)
	m.Registry.MustRegister(counterVec)
	m.counterVecs[name] = counterVec
}

// RegisterHistogram registers a new histogram metric.
func (m *Metrics) RegisterHistogram(name, help string, buckets []float64) {
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: buckets,
	})
	m.Registry.MustRegister(histogram)
	m.histograms[name] = histogram
}

// RegisterHistogramVec registers a new histogram metric with labels.
func (m *Metrics) RegisterHistogramVec(name, help string, buckets []float64, labels []string) {
	histogramVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: buckets,
	}, labels)
	m.Registry.MustRegister(histogramVec)
	m.histogramVecs[name] = histogramVec
}

// RegisterGauge registers a new gauge metric.
func (m *Metrics) RegisterGauge(name, help string) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	})
	m.Registry.MustRegister(gauge)
	m.gauges[name] = gauge
}

// RegisterGaugeVec registers a new gauge metric with labels.
func (m *Metrics) RegisterGaugeVec(name, help string, labels []string) {
	gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	}, labels)
	m.Registry.MustRegister(gaugeVec)
	m.gaugeVecs[name] = gaugeVec
}

// IncCounter increments a counter by 1.
func (m *Metrics) IncCounter(name string) {
	if counter, ok := m.counters[name]; ok {
		counter.Inc()
	}
}

// AddCounter adds a value to a counter.
func (m *Metrics) AddCounter(name string, value float64) {
	if counter, ok := m.counters[name]; ok {
		counter.Add(value)
	}
}

// IncCounterVec increments a counter in a CounterVec with labels.
func (m *Metrics) IncCounterVec(name string, labels ...string) {
	if counterVec, ok := m.counterVecs[name]; ok {
		counterVec.WithLabelValues(labels...).Inc()
	}
}

// AddCounterVec adds a value to a CounterVec with labels.
func (m *Metrics) AddCounterVec(name string, value float64, labels ...string) {
	if counterVec, ok := m.counterVecs[name]; ok {
		counterVec.WithLabelValues(labels...).Add(value)
	}
}

// ObserveHistogram observes a value in a histogram.
func (m *Metrics) ObserveHistogram(name string, value float64) {
	if histogram, ok := m.histograms[name]; ok {
		histogram.Observe(value)
	}
}

// ObserveHistogramVec observes a value in a histogram with labels.
func (m *Metrics) ObserveHistogramVec(name string, value float64, labels ...string) {
	if histogramVec, ok := m.histogramVecs[name]; ok {
		histogramVec.WithLabelValues(labels...).Observe(value)
	}
}

// AddGauge adds the given value to the Gauge. (The value can be negative,
// resulting in a decrease of the Gauge.)
func (m *Metrics) AddGauge(name string, value float64) {
	if gauge, ok := m.gauges[name]; ok {
		gauge.Add(value)
	}
}

// SetGauge sets a gauge to a specific value.
func (m *Metrics) SetGauge(name string, value float64) {
	if gauge, ok := m.gauges[name]; ok {
		gauge.Set(value)
	}
}

// IncGauge increments a gauge by 1.
func (m *Metrics) IncGauge(name string) {
	if gauge, ok := m.gauges[name]; ok {
		gauge.Inc()
	}
}

// DecGauge decrements a gauge by 1.
func (m *Metrics) DecGauge(name string) {
	if gauge, ok := m.gauges[name]; ok {
		gauge.Dec()
	}
}

// SubGauge subtracts the given value from the Gauge. (The value can be negative,
// resulting in an increase of the Gauge.)
func (m *Metrics) SubGauge(name string, value float64) {
	if gauge, ok := m.gauges[name]; ok {
		gauge.Sub(value)
	}
}

// SetCurrentTimeGauge sets the gauge to the current time in seconds since epoch.
func (m *Metrics) SetCurrentTimeGauge(name string) {
	if gauge, ok := m.gauges[name]; ok {
		gauge.SetToCurrentTime()
	}
}

// SetGaugeVec sets a gauge with labels to a specific value.
func (m *Metrics) SetGaugeVec(name string, value float64, labels ...string) {
	if gaugeVec, ok := m.gaugeVecs[name]; ok {
		gaugeVec.WithLabelValues(labels...).Set(value)
	}
}

// IncGaugeVec increments a gauge with labels by 1.
func (m *Metrics) IncGaugeVec(name string, labels ...string) {
	if gaugeVec, ok := m.gaugeVecs[name]; ok {
		gaugeVec.WithLabelValues(labels...).Inc()
	}
}

// DecGaugeVec decrements a gauge with labels by 1.
func (m *Metrics) DecGaugeVec(name string, labels ...string) {
	if gaugeVec, ok := m.gaugeVecs[name]; ok {
		gaugeVec.WithLabelValues(labels...).Dec()
	}
}
