package k8s

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	dropAnnounce = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "kiam",
			Subsystem: "k8s",
			Name:      "dropped_pods_total",
			Help:      "Number of dropped pods because of full buffer",
		},
	)
)

func init() {
	prometheus.MustRegister(dropAnnounce)
}
