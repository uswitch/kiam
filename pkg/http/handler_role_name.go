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
package http

import (
	"fmt"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/k8s"
	"github.com/vmg/backoff"
	"k8s.io/client-go/pkg/api/v1"
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
	requestLog := log.WithFields(requestFields(req))
	roleNameTimings := metrics.GetOrRegisterTimer("roleNameHandler", metrics.DefaultRegistry)
	startTime := time.Now()
	defer roleNameTimings.UpdateSince(startTime)

	ip, err := s.clientIP(req)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error parsing ip: %s", err.Error())
	}

	respCh := make(chan *asyncObj)
	go func() {
		podCh := make(chan *v1.Pod, 1)
		op := func() error {
			pod, err := s.finder.FindPodForIP(ip)
			if err != nil {
				return err
			}

			if pod == nil {
				requestLog.Warnf("no pod found for ip")
				return PodNotFound
			}

			podCh <- pod
			return nil
		}

		strategy := backoff.NewExponentialBackOff()
		strategy.MaxElapsedTime = s.cfg.MaxElapsedTime

		err = backoff.Retry(op, backoff.WithContext(strategy, req.Context()))
		if err != nil {
			respCh <- &asyncObj{obj: nil, err: err}
		} else {
			pod := <-podCh
			respCh <- &asyncObj{obj: pod, err: nil}
		}
	}()

	select {
	case <-req.Context().Done():
		if req.Context().Err() != nil {
			return http.StatusInternalServerError, req.Context().Err()
		}
	case resp := <-respCh:
		if resp.err == PodNotFound {
			metrics.GetOrRegisterMeter("roleNameHandler.podNotFound", metrics.DefaultRegistry).Mark(1)
			return http.StatusNotFound, fmt.Errorf("pod not found for ip %s", ip)
		} else if resp.err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error finding pod: %s", err.Error())
		}

		pod := resp.obj.(*v1.Pod)
		log.WithFields(k8s.PodFields(pod)).Infof("found pod")

		role := k8s.PodRole(pod)
		if role == "" {
			return http.StatusNotFound, fmt.Errorf("no role for pod %s", ip)
		}

		fmt.Fprint(w, role)
		metrics.GetOrRegisterMeter("roleNameHandler.success", metrics.DefaultRegistry).Mark(1)
		return http.StatusOK, nil
	}

	return http.StatusOK, nil
}
