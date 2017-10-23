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
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	khttp "github.com/uswitch/kiam/pkg/http"
	"github.com/uswitch/kiam/pkg/k8s"
	"github.com/uswitch/kiam/pkg/server"
	"github.com/vmg/backoff"
	"net/http"
	"time"
)

func findRole(ctx context.Context, finder k8s.RoleFinder, ip string) (string, error) {
	logger := log.WithField("pod.ip", ip)

	roleCh := make(chan string, 1)
	op := func() error {
		role, err := finder.FindRoleFromIP(ctx, ip)
		if err != nil {
			logger.Warnf("error finding role for pod: %s", err.Error())
			return err
		}
		roleCh <- role
		return nil
	}

	strategy := backoff.NewExponentialBackOff()
	strategy.InitialInterval = RetryInterval

	err := backoff.Retry(op, backoff.WithContext(strategy, ctx))
	if err != nil {
		return "", err
	}

	return <-roleCh, nil
}

func (s *Server) roleNameHandler(w http.ResponseWriter, req *http.Request) (int, error) {
	requestLog := log.WithFields(khttp.RequestFields(req))
	roleNameTimings := metrics.GetOrRegisterTimer("roleNameHandler", metrics.DefaultRegistry)
	startTime := time.Now()
	defer roleNameTimings.UpdateSince(startTime)

	ctx, cancel := context.WithTimeout(req.Context(), MaxTime)
	defer cancel()

	ip, err := s.clientIP(req)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error parsing ip: %s", err.Error())
	}

	role, err := findRole(ctx, s.finder, ip)
	if err != nil {
		if err == server.PodNotFoundError {
			requestLog.Errorf("no pod found for ip")
			metrics.GetOrRegisterMeter("roleNameHandler.podNotFound", metrics.DefaultRegistry).Mark(1)
			return http.StatusNotFound, err
		}

		return http.StatusInternalServerError, err
	}

	if role == "" {
		requestLog.Warnf("empty role defined for pod")
		metrics.GetOrRegisterMeter("credentialsHandler.emptyRole", metrics.DefaultRegistry).Mark(1)
		return http.StatusNotFound, EmptyRoleError
	}

	fmt.Fprint(w, role)
	metrics.GetOrRegisterMeter("roleNameHandler.success", metrics.DefaultRegistry).Mark(1)
	return http.StatusOK, nil
}
