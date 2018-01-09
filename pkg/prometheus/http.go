package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

// TelemetryServer runs an HTTP service for exporting
// metrics
type TelemetryServer struct {
	server    *http.Server
	subsystem string
	sync      time.Duration
}

func prometheusMetrics(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "This is a test!")
}

// NewServer creates a prometheus text format HTTP metrics server
func NewServer(subsystem, listenAddr string, syncInterval time.Duration) *TelemetryServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	return &TelemetryServer{server: server, subsystem: subsystem, sync: syncInterval}
}

// Listen starts an HTTP service exporting metrics. It stops
// when the passed context is completed.
func (s *TelemetryServer) Listen(ctx context.Context) {
	go func() {
		log.Infof("started prometheus metric listener %s", s.server.Addr)
		s.server.ListenAndServe()
	}()
	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		log.Infof("stopping prometheus metric listener")
		s.server.Shutdown(ctx)
	}()
	go func() {
		prom := NewPrometheusSyncer(metrics.DefaultRegistry, s.subsystem, prometheus.DefaultRegisterer)
		refreshCounter := metrics.GetOrRegisterCounter("metrics_refresh", metrics.DefaultRegistry)

		for {
			select {
			case _ = <-time.Tick(s.sync):
				err := prom.Sync()
				if err != nil {
					log.Errorf("error updating prometheus metrics: %s", err)
				}
				refreshCounter.Inc(1)

			case _ = <-ctx.Done():
				return
			}
		}
	}()
}
