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
	"io/ioutil"
	"net/http"
	"time"
)

func (s *Server) healthHandler(w http.ResponseWriter, req *http.Request) (int, error) {
	healthTimer := metrics.GetOrRegisterTimer("healthHandler", metrics.DefaultRegistry)
	started := time.Now()
	defer healthTimer.UpdateSince(started)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/latest/meta-data/instance-id", s.cfg.MetadataEndpoint), nil)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("couldn't create request: %s", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req.WithContext(ctx))
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
