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
	"fmt"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	khttp "github.com/uswitch/kiam/pkg/http"
	// "github.com/vmg/backoff"
	"net/http"
	"time"
)

var (
	PodNotFound = fmt.Errorf("pod not found")
)

type asyncObj struct {
	obj interface{}
	err error
}

func (s *Server) roleNameHandler(w http.ResponseWriter, req *http.Request) (int, error) {
	requestLog := log.WithFields(khttp.RequestFields(req))
	roleNameTimings := metrics.GetOrRegisterTimer("roleNameHandler", metrics.DefaultRegistry)
	startTime := time.Now()
	defer roleNameTimings.UpdateSince(startTime)

	ip, err := s.clientIP(req)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error parsing ip: %s", err.Error())
	}

	role, err := s.finder.FindRoleFromIP(req.Context(), ip)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if role == "" {
		requestLog.Warnf("no pod found for ip")
		metrics.GetOrRegisterMeter("roleNameHandler.podNotFound", metrics.DefaultRegistry).Mark(1)

		return http.StatusNotFound, PodNotFound
	}

	fmt.Fprint(w, role)
	metrics.GetOrRegisterMeter("roleNameHandler.success", metrics.DefaultRegistry).Mark(1)
	return http.StatusOK, nil
}
