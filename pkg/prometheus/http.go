package prometheus

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// TelemetryServer runs an HTTP service for exporting
// metrics
type TelemetryServer struct {
	server    *http.Server
	subsystem string
	sync      time.Duration
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
}
