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
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/server"
	"net/http"
	"net/url"
	"time"
)

type roleHandler struct {
	client   server.Client
	clientIP clientIPFunc
}

func trailingSlashSuffixRedirectHandler(rw http.ResponseWriter, req *http.Request) {
	u, err := url.Parse(req.URL.String())
	if err != nil {
		log.Errorf("error parsing uri: %s", err)
		http.Error(rw, "error parsing uri", http.StatusInternalServerError)
		return
	}

	u.Path = fmt.Sprintf("%s/", u.Path)
	http.Redirect(rw, req, u.String(), http.StatusTemporaryRedirect)
}

func (h *roleHandler) Install(router *mux.Router) {
	handler := adapt(withMeter("roleName", h))
	router.Handle("/{version}/meta-data/iam/security-credentials/", handler)
	router.HandleFunc("/{version}/meta-data/iam/security-credentials", trailingSlashSuffixRedirectHandler)
}

func (h *roleHandler) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request) (int, error) {
	roleNameTimings := metrics.GetOrRegisterTimer("roleNameHandler", metrics.DefaultRegistry)
	startTime := time.Now()
	defer roleNameTimings.UpdateSince(startTime)

	err := req.ParseForm()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	ip, err := h.clientIP(req)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	role, err := findRole(ctx, h.client, ip)

	if err != nil {
		metrics.GetOrRegisterMeter("roleNameHandler.findRoleError", metrics.DefaultRegistry).Mark(1)
		return http.StatusInternalServerError, err
	}

	if role == "" {
		metrics.GetOrRegisterMeter("credentialsHandler.emptyRole", metrics.DefaultRegistry).Mark(1)
		return http.StatusNotFound, EmptyRoleError
	}

	fmt.Fprint(w, role)
	metrics.GetOrRegisterMeter("roleNameHandler.success", metrics.DefaultRegistry).Mark(1)

	return http.StatusOK, nil
}

const (
	retryInterval = time.Millisecond * 5
)

func findRole(ctx context.Context, client server.Client, ip string) (string, error) {
	logger := log.WithField("pod.ip", ip)

	roleCh := make(chan string, 1)
	op := func() error {
		role, err := client.GetRole(ctx, ip)
		if err != nil {
			logger.Warnf("error finding role for pod: %s", err.Error())
			return err
		}
		roleCh <- role
		return nil
	}

	strategy := backoff.NewExponentialBackOff()
	strategy.InitialInterval = retryInterval

	err := backoff.Retry(op, backoff.WithContext(strategy, ctx))
	if err != nil {
		return "", err
	}

	return <-roleCh, nil
}
