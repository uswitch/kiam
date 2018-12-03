package pprof

import (
	"context"
	log "github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
)

func NewServer(listenAddr string) http.Server {
	server := http.Server{Addr: listenAddr, Handler: http.DefaultServeMux}
	return server
}

// Starts server and shuts down when context signals
func ListenAndWait(ctx context.Context, server http.Server) {
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Errorf("error starting pprof http server: %s", err.Error())
		}
	}()
	<-ctx.Done()
	log.Infof("shutting down pprof server")
	server.Shutdown(context.Background())
}
