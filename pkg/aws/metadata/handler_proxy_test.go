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
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

func performRequest(allowed, path string) (int, *httptest.ResponseRecorder) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var hits int
	backingService := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	})
	handler := newProxyHandler(backingService, regexp.MustCompile(allowed))
	router := mux.NewRouter()
	handler.Install(router)

	r, _ := http.NewRequest("GET", path, nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, r.WithContext(ctx))

	return hits, rr
}

func TestProxyDefaultBlacklistingRoot(t *testing.T) {
	hits, rr := performRequest("", "/")

	if hits != 0 {
		t.Error("unexpected reverse proxy hit")
	}
	if rr.Code != http.StatusNotFound {
		t.Error("unexpected status", rr.Code)
	}
	if !strings.HasPrefix(rr.Body.String(), "request blocked by whitelist-route-regexp") {
		t.Error("unexpected body:", rr.Body.String())
	}
}

func readPrometheusSimpleCounterValue(name string) float64 {
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		panic(err)
	}

	for _, m := range metrics {
		if m.GetName() == name {
			return m.Metric[0].Counter.GetValue()
		}
	}

	return 0
}

func TestProxyFiltering(t *testing.T) {
	requestsInitial := readPrometheusCounterValue("kiam_metadata_responses_total", "handler", "proxy")
	blockedInitial := readPrometheusSimpleCounterValue("kiam_metadata_proxy_requests_blocked_total")
	hits, rr := performRequest("foo.*", "/bar")

	if hits != 0 {
		t.Error("unexpected reverse proxy hit")
	}
	if rr.Code != http.StatusNotFound {
		t.Error("unexpected status", rr.Code)
	}
	if !strings.HasPrefix(rr.Body.String(), "request blocked by whitelist-route-regexp") {
		t.Error("unexpected body:", rr.Body.String())
	}

	responses := readPrometheusCounterValue("kiam_metadata_responses_total", "handler", "proxy")
	if responses - requestsInitial != 1 {
		t.Error("expected responses_total to be 1, was", responses)
	}
	blocked := readPrometheusSimpleCounterValue("kiam_metadata_proxy_requests_blocked_total")
	if blocked - blockedInitial != 1 {
		t.Error("expected blocked total to be 1, was", blocked)
	}
}

func TestProxyFilteringSubpath(t *testing.T) {
	hits, rr := performRequest("foo.*", "/bar/baz")

	if hits != 0 {
		t.Error("unexpected reverse proxy hit")
	}
	if rr.Code != http.StatusNotFound {
		t.Error("unexpected status", rr.Code)
	}
	if !strings.HasPrefix(rr.Body.String(), "request blocked by whitelist-route-regexp") {
		t.Error("unexpected body:", rr.Body.String())
	}
}

func TestProxyWhitelisting(t *testing.T) {
	hits, rr := performRequest("foo.*", "/foo")

	if hits != 1 {
		t.Error("expected reverse proxy hit")
	}
	if rr.Code != http.StatusOK {
		t.Error("unexpected status", rr.Code)
	}
}
