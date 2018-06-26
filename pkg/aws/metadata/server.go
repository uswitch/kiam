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
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/k8s"
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
	DisableUserData  bool
}

func NewConfig(port int) *ServerConfig {
	return &ServerConfig{
		MetadataEndpoint: "http://169.254.169.254",
		ListenPort:       port,
		AllowIPQuery:     false,
		DisableUserData:  false,
	}
}

func NewWebServer(config *ServerConfig, finder k8s.RoleFinder, credentials sts.CredentialsProvider, policy server.AssumeRolePolicy) (*Server, error) {
	http, err := buildHTTPServer(config, finder, credentials, policy)
	if err != nil {
		return nil, err
	}
	return &Server{cfg: config, server: http}, nil
}

func buildHTTPServer(config *ServerConfig, finder k8s.RoleFinder, credentials sts.CredentialsProvider, policy server.AssumeRolePolicy) (*http.Server, error) {
	router := mux.NewRouter()
	router.Handle("/metrics", exp.ExpHandler(metrics.DefaultRegistry))
	router.Handle("/ping", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "pong") }))

	h := &healthHandler{config.MetadataEndpoint}
	router.Handle("/health", adapt(withMeter("health", h)))

	clientIP := buildClientIP(config)

	r := &roleHandler{
		roleFinder: finder,
		clientIP:   clientIP,
	}

	securityCredsHandler := adapt(withMeter("roleName", r))
	router.Handle("/{version}/meta-data/iam/security-credentials", securityCredsHandler)
	router.Handle("/{version}/meta-data/iam/security-credentials/", securityCredsHandler)

	c := &credentialsHandler{
		roleFinder:          finder,
		credentialsProvider: credentials,
		clientIP:            clientIP,
		policy:              policy,
	}
	router.Handle("/{version}/meta-data/iam/security-credentials/{role:.*}", adapt(withMeter("credentials", c)))

	if config.DisableUserData {
		router.Handle("/{version}/user-data", http.NotFoundHandler())
	}

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
