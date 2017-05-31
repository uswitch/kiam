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
package testutil

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type stubMetadataService struct {
	port     int
	metadata *AWSMetadata
	lock     sync.Mutex
	server   *http.Server
}

func (s *stubMetadataService) serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/latest/meta-data/instance-id", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, s.metadata.InstanceID) })
	s.lock.Lock()
	s.server = &http.Server{Addr: fmt.Sprintf(":%d", s.port), Handler: mux}
	s.lock.Unlock()
	s.server.ListenAndServe()
}

func (s *stubMetadataService) stop(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.server.Shutdown(ctx)
}

func newAWS(metadata *AWSMetadata, port int) *stubMetadataService {
	return &stubMetadataService{port: port, metadata: metadata}
}

type AWSMetadata struct {
	InstanceID string
}

func WithAWS(metadata *AWSMetadata, ctx context.Context, body func(ctx context.Context)) {
	aws := newAWS(metadata, 3199)
	go aws.serve()
	defer aws.stop(ctx)

	// wait for api to be active
	time.Sleep(2 * time.Millisecond)

	body(ctx)
}
