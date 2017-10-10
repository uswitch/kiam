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
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rcrowley/go-metrics"
	"github.com/uswitch/kiam/pkg/aws/sts"
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

	foundRole, err := s.finder.FindRoleFromIP(ip)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error finding pod for ip %s: %s", ip, err.Error())
	}

	if foundRole == "" {
		metrics.GetOrRegisterMeter("credentialsHandler.podNotFound", metrics.DefaultRegistry).Mark(1)
		return http.StatusNotFound, fmt.Errorf("no pod found for ip %s", ip)
	}

	role := mux.Vars(req)["role"]
	if role == "" {
		return http.StatusBadRequest, fmt.Errorf("no role specified")
	}

	if foundRole != role {
		return http.StatusForbidden, fmt.Errorf("unable to assume role %s, role on pod specified is %s", role, foundRole)
	}

	respCh := make(chan *asyncObj)
	go func() {
		credentials, err := s.credentials.CredentialsForRole(role)
		respCh <- &asyncObj{credentials, err}
	}()

	select {
	case <-req.Context().Done():
		if req.Context().Err() != nil {
			return http.StatusInternalServerError, fmt.Errorf("unexpected error: %s", req.Context().Err().Error())
		}

	case resp := <-respCh:
		if resp.err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error requesting credentials: %s", resp.err.Error())
		}

		creds := resp.obj.(*sts.Credentials)
		err = json.NewEncoder(w).Encode(creds)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error encoding credentials: %s", err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
	}

	metrics.GetOrRegisterMeter("credentialsHandler.success", metrics.DefaultRegistry).Mark(1)
	return http.StatusOK, nil
}
