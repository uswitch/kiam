package metadata

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	handlerTimer = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "kiam",
			Subsystem: "metadata",
			Name:      "handler_latency_milliseconds",
			Help:      "Bucketed histogram of handler timings",

			// 1ms to 5min
			Buckets: prometheus.ExponentialBuckets(.001, 2, 13),
		},
		[]string{"handler"},
	)

	credentialFetchError = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kiam",
			Subsystem: "metadata",
			Name:      "credential_fetch_errors_total",
			Help:      "Number of errors fetching the credentials for a pod",
		},
		[]string{"handler"},
	)

	credentialEncodeError = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kiam",
			Subsystem: "metadata",
			Name:      "credential_encode_errors_total",
			Help:      "Number of errors encoding credentials for a pod",
		},
		[]string{"handler"},
	)

	findRoleError = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kiam",
			Subsystem: "metadata",
			Name:      "find_role_errors_total",
			Help:      "Number of errors finding the role for a pod",
		},
		[]string{"handler"},
	)

	emptyRole = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kiam",
			Subsystem: "metadata",
			Name:      "empty_role_total",
			Help:      "Number of empty roles returned",
		},
		[]string{"handler"},
	)

	success = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kiam",
			Subsystem: "metadata",
			Name:      "success_total",
			Help:      "Number of successful responses from a handler",
		},
		[]string{"handler"},
	)

	responses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kiam",
			Subsystem: "metadata",
			Name:      "responses_total",
			Help:      "Responses from mocked out metadata handlers",
		},
		[]string{"handler", "code"},
	)

	proxyDenies = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "kiam",
			Subsystem: "metadata",
			Name:      "proxy_requests_blocked_total",
			Help:      "Number of access requests to the proxy handler that were blocked by the regexp",
		},
	)
)

func init() {
	prometheus.MustRegister(handlerTimer)
	prometheus.MustRegister(findRoleError)
	prometheus.MustRegister(emptyRole)
	prometheus.MustRegister(success)
	prometheus.MustRegister(responses)
	prometheus.MustRegister(proxyDenies)
}
