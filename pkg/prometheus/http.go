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
	server *http.Server
}

func prometheusMetrics(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "This is a test!")
}

// NewServer creates a prometheus text format HTTP metrics server
func NewServer(subsystem, listenAddr string) *TelemetryServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	return &TelemetryServer{server: server}
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
		prom := NewPrometheusProvider(metrics.DefaultRegistry, "subsystem", prometheus.DefaultRegisterer)
		refreshCounter := metrics.GetOrRegisterCounter("metrics_refresh", metrics.DefaultRegistry)

		for {
			select {
			case _ = <-time.Tick(time.Second * 5):
				err := prom.UpdatePrometheusMetricsOnce()
				if err != nil {
					log.Errorf("error updating prometheus metrics: %s", err)
				}
				refreshCounter.Inc(1)

				mets, err := prometheus.DefaultGatherer.Gather()
				if err != nil {
					log.Error(err)
				}

				for _, obj := range mets {
					log.Debugf("%s", obj.GetName())
				}
			case _ = <-ctx.Done():
				return
			}
		}
	}()
}
