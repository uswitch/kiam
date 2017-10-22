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
	"github.com/gorilla/mux"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/aws/sts"
	khttp "github.com/uswitch/kiam/pkg/http"
	"github.com/uswitch/kiam/pkg/server"
	"github.com/vmg/backoff"
	"net/http"
	"time"
)

func (s *Server) credentialsHandler(w http.ResponseWriter, req *http.Request) (int, error) {
	credentialTimings := metrics.GetOrRegisterTimer("credentialsHandler", metrics.DefaultRegistry)
	startTime := time.Now()
	defer credentialTimings.UpdateSince(startTime)

	req.ParseForm()

	ip, err := s.clientIP(req)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error parsing client ip %s: %s", ip, err.Error())
	}

	ctx, cancel := context.WithTimeout(req.Context(), MaxTime)
	defer cancel()

	logger := log.WithFields(khttp.RequestFields(req))

	roleCh := make(chan string, 1)
	op := func() error {
		foundRole, err := s.finder.FindRoleFromIP(ctx, ip)
		if err != nil {
			logger.Errorf("error finding role for ip %s", ip)
			return err
		}
		roleCh <- foundRole
		return nil
	}

	strategy := backoff.NewExponentialBackOff()
	strategy.InitialInterval = RetryInterval
	err = backoff.Retry(op, backoff.WithContext(strategy, ctx))

	if err != nil {
		if err == server.PodNotFoundError {
			metrics.GetOrRegisterMeter("credentialsHandler.podNotFound", metrics.DefaultRegistry).Mark(1)
			return http.StatusNotFound, fmt.Errorf("no pod found for ip %s", ip)
		}

		return http.StatusInternalServerError, fmt.Errorf("error finding pod for ip %s: %s", ip, err.Error())
	}

	foundRole := <-roleCh

	if foundRole == "" {
		metrics.GetOrRegisterMeter("credentialsHandler.emptyRole", metrics.DefaultRegistry).Mark(1)
		return http.StatusNotFound, EmptyRoleError
	}

	role := mux.Vars(req)["role"]
	if role == "" {
		return http.StatusBadRequest, fmt.Errorf("no role specified")
	}

	if foundRole != role {
		return http.StatusForbidden, fmt.Errorf("unable to assume role %s, role on pod specified is %s", role, foundRole)
	}

	credsCh := make(chan *sts.Credentials, 1)
	op = func() error {
		credentials, err := s.credentials.CredentialsForRole(ctx, role)
		if err != nil {
			logger.WithField("pod.iam.role", role).Errorf("error getting credentials for role: %s", err.Error())
			return err
		}
		credsCh <- credentials
		return nil
	}

	strategy = backoff.NewExponentialBackOff()
	strategy.InitialInterval = RetryInterval
	err = backoff.Retry(op, backoff.WithContext(strategy, ctx))

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("unexpected error: %s", ctx.Err().Error())
	}

	credentials := <-credsCh
	err = json.NewEncoder(w).Encode(credentials)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error encoding credentials: %s", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	metrics.GetOrRegisterMeter("credentialsHandler.success", metrics.DefaultRegistry).Mark(1)
	return http.StatusOK, nil
}
