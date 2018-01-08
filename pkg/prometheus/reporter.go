// Licensed under https://github.com/deathowl/go-metrics-prometheus/commit/adef8c6b8d2e5eb5cec6d56f7fccac51b8984419
package prometheus

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rcrowley/go-metrics"
)

// PrometheusConfig provides a container with config parameters for the
// Prometheus Exporter

type PrometheusConfig struct {
	Registry      metrics.Registry // Registry to be exported
	subsystem     string
	promRegistry  prometheus.Registerer //Prometheus registry
	FlushInterval time.Duration         //interval to update prom metrics
	gauges        map[string]prometheus.Gauge
}

// NewPrometheusProvider returns a Provider that produces Prometheus metrics.
// Namespace and subsystem are applied to all produced metrics.
func NewPrometheusProvider(r metrics.Registry, subsystem string, promRegistry prometheus.Registerer) *PrometheusConfig {
	return &PrometheusConfig{
		subsystem:    subsystem,
		Registry:     r,
		promRegistry: promRegistry,
		gauges:       make(map[string]prometheus.Gauge),
	}
}

var (
	prometheusKey = regexp.MustCompile("\\W+")
)

func (c *PrometheusConfig) flattenKey(key string) string {
	return prometheusKey.ReplaceAllString(strings.ToLower(key), "_")
}

func (c *PrometheusConfig) gaugeFromNameAndValue(name string, val float64) {
	key := fmt.Sprintf("%s_%s_%s", "kiam", c.subsystem, name)
	g, ok := c.gauges[key]
	if !ok {
		g = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.flattenKey("kiam"),
			Subsystem: c.flattenKey(c.subsystem),
			Name:      c.flattenKey(name),
			Help:      name,
		})
		c.promRegistry.MustRegister(g)
		c.gauges[key] = g
	}
	g.Set(val)
}

func (c *PrometheusConfig) UpdatePrometheusMetricsOnce() error {
	c.Registry.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		case metrics.Counter:
			c.gaugeFromNameAndValue(name, float64(metric.Count()))
		case metrics.Gauge:
			c.gaugeFromNameAndValue(name, float64(metric.Value()))
		case metrics.GaugeFloat64:
			c.gaugeFromNameAndValue(name, float64(metric.Value()))
		case metrics.Histogram:
			samples := metric.Snapshot().Sample().Values()
			if len(samples) > 0 {
				lastSample := samples[len(samples)-1]
				c.gaugeFromNameAndValue(name, float64(lastSample))
			}
		case metrics.Meter:
			lastSample := metric.Snapshot().Rate1()
			c.gaugeFromNameAndValue(name, float64(lastSample))
		case metrics.Timer:
			lastSample := metric.Snapshot().Rate1()
			c.gaugeFromNameAndValue(name, float64(lastSample))
		}
	})
	return nil
}
