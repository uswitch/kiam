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
	"github.com/uswitch/kiam/pkg/server"
	"github.com/vmg/backoff"
	"net/http"
	"time"
)

var (
	MaxTime = time.Millisecond * 500
)

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

	roleCh := make(chan string, 1)
	op := func() error {
		role, err := s.finder.FindRoleFromIP(ctx, ip)
		if err != nil {
			return err
		}
		roleCh <- role
		return nil
	}

	strategy := backoff.NewExponentialBackOff()
	strategy.InitialInterval = 5 * time.Millisecond
	err = backoff.Retry(op, backoff.WithContext(strategy, ctx))
	if err != nil {
		fmt.Println("error: ", err)

		if err == server.PodNotFoundError {
			requestLog.Warnf("no pod found for ip")
			metrics.GetOrRegisterMeter("roleNameHandler.podNotFound", metrics.DefaultRegistry).Mark(1)
			return http.StatusNotFound, err
		}

		return http.StatusInternalServerError, err
	}

	role := <-roleCh
	if role == "" {
		requestLog.Warnf("empty role defined for pod")
		metrics.GetOrRegisterMeter("credentialsHandler.emptyRole", metrics.DefaultRegistry).Mark(1)
		return http.StatusNotFound, EmptyRoleError
	}

	fmt.Fprint(w, role)
	metrics.GetOrRegisterMeter("roleNameHandler.success", metrics.DefaultRegistry).Mark(1)
	return http.StatusOK, nil
}
