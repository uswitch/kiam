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
	"github.com/cenkalti/backoff"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/uswitch/kiam/pkg/server"
	"io/ioutil"
	"net/http"
)

type healthHandler struct {
	client   server.Client
	endpoint string
}

func (h *healthHandler) Install(router *mux.Router) {
	router.Handle("/health", adapt(withMeter("health", h)))
}

func (h *healthHandler) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request) (int, error) {
	timer := prometheus.NewTimer(handlerTimer.WithLabelValues("health"))
	defer timer.ObserveDuration()

	deep := req.URL.Query().Get("deep")
	if deep != "" {
		health, err := findServerHealth(ctx, h.client)
		if err != nil {
			return http.StatusInternalServerError, err
		} else if health != "ok" {
			return http.StatusInternalServerError, fmt.Errorf("server health: %s", health)
		}
	}

	metaReq, err := http.NewRequest("GET", fmt.Sprintf("%s/latest/meta-data/instance-id", h.endpoint), nil)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("couldn't create request: %s", err)
	}

	client := &http.Client{}
	resp, err := client.Do(metaReq.WithContext(ctx))
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("couldn't read metadata response: %s", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("couldn't read metadata response: %s", err)
	}

	fmt.Fprint(w, string(body))
	return http.StatusOK, nil
}

func findServerHealth(ctx context.Context, client server.Client) (string, error) {
	var health string
	op := func() error {
		var err error
		health, err = client.Health(ctx)
		return err
	}

	strategy := backoff.NewExponentialBackOff()
	strategy.InitialInterval = retryInterval

	err := backoff.Retry(op, backoff.WithContext(strategy, ctx))
	if err != nil {
		return "", err
	}

	return health, nil
}

func newHealthHandler(client server.Client, endpoint string) *healthHandler {
	return &healthHandler{
		client:   client,
		endpoint: endpoint,
	}
}
