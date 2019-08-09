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
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/server"
)

type Server struct {
	cfg    *ServerOptions
	server *http.Server
}

type ServerOptions struct {
	ListenPort           int
	MetadataEndpoint     string
	AllowIPQuery         bool
	WhitelistRouteRegexp *regexp.Regexp
}

func DefaultOptions() *ServerOptions {
	return &ServerOptions{
		MetadataEndpoint:     "http://169.254.169.254",
		ListenPort:           3100,
		AllowIPQuery:         false,
		WhitelistRouteRegexp: regexp.MustCompile("^$"),
	}
}

func NewWebServer(config *ServerOptions, client server.Client) (*Server, error) {
	http, err := buildHTTPServer(config, client)
	if err != nil {
		return nil, err
	}
	return &Server{cfg: config, server: http}, nil
}

func buildHTTPServer(config *ServerOptions, client server.Client) (*http.Server, error) {
	router := mux.NewRouter()
	router.Handle("/ping", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "pong") }))

	h := newHealthHandler(client, config.MetadataEndpoint)
	h.Install(router)

	r := newRoleHandler(client, buildClientIP(config))
	r.Install(router)

	c := newCredentialsHandler(client, buildClientIP(config))
	c.Install(router)

	metadataURL, err := url.Parse(config.MetadataEndpoint)
	if err != nil {
		return nil, err
	}

	p := newProxyHandler(httputil.NewSingleHostReverseProxy(metadataURL), config.WhitelistRouteRegexp)
	p.Install(router)

	listen := fmt.Sprintf(":%d", config.ListenPort)
	return &http.Server{Addr: listen, Handler: loggingHandler(router)}, nil
}

func buildClientIP(config *ServerOptions) clientIPFunc {
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
