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
	"net/http"
	"time"
)

// interface for request handlers
type handler interface {
	// all http handlers will implement this function
	Handle(ctx context.Context, w http.ResponseWriter, req *http.Request) (int, error)
}

// clientIPFunc is the function used by handlers to find the client IP address
type clientIPFunc func(req *http.Request) (string, error)

const (
	handlerMaxDuration = time.Second * 5 //
)

// adapts between handler and http.Handler
type handlerAdapter struct {
	h handler
}

func (a *handlerAdapter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), handlerMaxDuration)
	defer cancel()

	status, err := a.h.Handle(ctx, w, req)

	if err != nil {
		log.WithFields(khttp.RequestFields(req)).WithField("status", status).Errorf("error processing request: %s", err.Error())
		http.Error(w, err.Error(), status)
	}
}

func adapt(h handler) *handlerAdapter {
	return &handlerAdapter{h: h}
}

// uses a meter to record error statuses
type metricHandler struct {
	name string
	h    handler
}

func withMeter(name string, h handler) handler {
	return &metricHandler{
		name: name,
		h:    h,
	}
}

func (m *metricHandler) Handle(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	status, err := m.h.Handle(ctx, w, r)
	getResponseMeter(m.name, status).Mark(1)
	return status, err
}

func getResponseMeter(name string, result int) metrics.Meter {
	bucket := getStatusBucket(result)
	return metrics.GetOrRegisterMeter(fmt.Sprintf("handlerResponse-%s.%s", name, bucket), metrics.DefaultRegistry)
}

func getStatusBucket(status int) string {
	if status >= 200 && status < 300 {
		return "2xx"
	}
	if status >= 300 && status < 400 {
		return "3xx"
	}
	if status >= 400 && status < 500 {
		return "4xx"
	}
	if status >= 500 && status < 600 {
		return "5xx"
	}
	return "unknown"
}
