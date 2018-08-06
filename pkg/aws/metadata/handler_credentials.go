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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cenkalti/backoff"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/server"
	"github.com/uswitch/kiam/pkg/statsd"
)

type credentialsHandler struct {
	clientIP clientIPFunc
	client   server.Client
}

func (c *credentialsHandler) Install(router *mux.Router) {
	router.Handle("/{version}/meta-data/iam/security-credentials/{role:.*}", adapt(withMeter("credentials", c)))
}

func (c *credentialsHandler) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request) (int, error) {
	timer := prometheus.NewTimer(handlerTimer.WithLabelValues("credentials"))
	defer timer.ObserveDuration()
	if statsd.Enabled {
		defer statsd.Client.NewTiming().Send("handler.credentials")
	}

	err := req.ParseForm()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	ip, err := c.clientIP(req)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	requestedRole := mux.Vars(req)["role"]
	credentials, err := c.fetchCredentials(ctx, ip, requestedRole)
	if err != nil {
		credentialFetchError.WithLabelValues("credentials").Inc()
		return http.StatusInternalServerError, fmt.Errorf("error fetching credentials: %s", err)
	}

	err = json.NewEncoder(w).Encode(credentials)
	if err != nil {
		credentialEncodeError.WithLabelValues("credentials").Inc()
		return http.StatusInternalServerError, fmt.Errorf("error encoding credentials: %s", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	success.WithLabelValues("credentials").Inc()
	return http.StatusOK, nil
}

func (c *credentialsHandler) fetchCredentials(ctx context.Context, ip, requestedRole string) (*sts.Credentials, error) {
	credsCh := make(chan *sts.Credentials, 1)
	op := func() error {
		creds, err := c.client.GetCredentials(ctx, ip, requestedRole)
		if err != nil {
			return err
		}
		credsCh <- creds
		return nil
	}

	strategy := backoff.NewExponentialBackOff()
	strategy.InitialInterval = retryInterval

	err := backoff.Retry(op, backoff.WithContext(strategy, ctx))
	if err != nil {
		return nil, err
	}
	return <-credsCh, nil
}
