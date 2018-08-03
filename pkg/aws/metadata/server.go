// Copyright 2017 uSwitch
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package metadata

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/server"
)

type Server struct {
	cfg    *ServerConfig
	server *http.Server
}

type ServerConfig struct {
	ListenPort       int
	MetadataEndpoint string
	AllowIPQuery     bool
}

func NewConfig(port int) *ServerConfig {
	return &ServerConfig{
		MetadataEndpoint: "http://169.254.169.254",
		ListenPort:       port,
		AllowIPQuery:     false,
	}
}

func NewWebServer(config *ServerConfig, client server.Client, statsd bool) (*Server, error) {
	http, err := buildHTTPServer(config, client, statsd)
	if err != nil {
		return nil, err
	}
	return &Server{cfg: config, server: http}, nil
}

func buildHTTPServer(config *ServerConfig, client server.Client, statsd bool) (*http.Server, error) {
	router := mux.NewRouter()
	router.Handle("/ping", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "pong") }))

	h := &healthHandler{endpoint: config.MetadataEndpoint, statsd: statsd}
	router.Handle("/health", adapt(withMeter("health", h)))

	clientIP := buildClientIP(config)

	r := &roleHandler{
		client:   client,
		clientIP: clientIP,
		statsd:   statsd,
	}
	r.Install(router)

	c := &credentialsHandler{
		clientIP: clientIP,
		client:   client,
		statsd:   statsd,
	}
	c.Install(router)

	metadataURL, err := url.Parse(config.MetadataEndpoint)
	if err != nil {
		return nil, err
	}
	router.Handle("/{path:.*}", httputil.NewSingleHostReverseProxy(metadataURL))

	listen := fmt.Sprintf(":%d", config.ListenPort)
	return &http.Server{Addr: listen, Handler: loggingHandler(router)}, nil
}

func buildClientIP(config *ServerConfig) clientIPFunc {
	remote := func(req *http.Request) (string, error) {
		return ParseClientIP(req.RemoteAddr)
	}

	if config.AllowIPQuery {
		return func(req *http.Request) (string, error) {
			ip := req.Form.Get("ip")
			if ip != "" {
				return ip, nil
			}
			return remote(req)
		}
	}

	return remote
}

func (s *Server) Serve() error {
	log.Infof("listening :%d", s.cfg.ListenPort)
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) {
	log.Infoln("starting server shutdown")
	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	s.server.Shutdown(c)
	log.Infoln("gracefully shutdown server")
}

func ParseClientIP(addr string) (string, error) {
	parts := strings.Split(addr, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("incorrect format, expected ip:port, was: %s", addr)
	}

	return strings.Join(parts[0:len(parts)-1], ":"), nil
}
