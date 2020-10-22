package sts

import "github.com/prometheus/client_golang/prometheus"

var (
	cacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "kiam",
			Subsystem: "sts",
			Name:      "cacheSize",
			Help:      "Current size of the metadata cache",
		})

	cacheHit = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "kiam",
			Subsystem: "sts",
			Name:      "cache_hit_total",
			Help:      "Number of cache hits to the metadata cache",
		},
	)

	cacheMiss = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "kiam",
			Subsystem: "sts",
			Name:      "cache_miss_total",
			Help:      "Number of cache misses to the metadata cache",
		},
	)

	errorIssuing = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "kiam",
			Subsystem: "sts",
			Name:      "issuing_errors_total",
			Help:      "Number of errors issuing credentials",
		},
	)

	assumeRole = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "kiam",
			Subsystem: "sts",
			Name:      "assumerole_timing_seconds",
			Help:      "Bucketed histogram of assumeRole timings",

			// 1ms to 5min
			Buckets: prometheus.ExponentialBuckets(.001, 2, 13),
		},
	)

	assumeRoleExecuting = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "kiam",
			Subsystem: "sts",
			Name:      "assumerole_current",
			Help:      "Number of assume role calls currently executing",
		},
	)
)

func init() {
	prometheus.MustRegister(cacheHit)
	prometheus.MustRegister(cacheMiss)
	prometheus.MustRegister(cacheSize)
	prometheus.MustRegister(errorIssuing)
	prometheus.MustRegister(assumeRole)
	prometheus.MustRegister(assumeRoleExecuting)
}
