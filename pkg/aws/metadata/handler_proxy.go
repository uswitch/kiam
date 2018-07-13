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
	"regexp"

	"github.com/gorilla/mux"
)

type proxyHandler struct {
	backingService       http.Handler
	whitelistRouteRegexp *regexp.Regexp
}

func (p *proxyHandler) Install(router *mux.Router) {
	router.Handle("/{path}", adapt(withMeter("proxy", p)))
}

type teeWriter struct {
	http.ResponseWriter
	status int
}

func (w *teeWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (p *proxyHandler) Handle(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	route := mux.Vars(r)["path"]
	if p.whitelistRouteRegexp.MatchString(route) {
		writer := &teeWriter{w, http.StatusOK}
		p.backingService.ServeHTTP(writer, r)
		return writer.status, nil
	} else {
		return http.StatusNotFound, fmt.Errorf("request blocked by whitelist-route-regexp %q: %s", p.whitelistRouteRegexp, route)
	}
}

func newProxyHandler(backingService http.Handler, whitelistRouteRegexp *regexp.Regexp) *proxyHandler {
	if whitelistRouteRegexp.String() == "" {
		whitelistRouteRegexp = regexp.MustCompile("^$")
	}
	return &proxyHandler{
		backingService:       backingService,
		whitelistRouteRegexp: whitelistRouteRegexp,
	}
}
