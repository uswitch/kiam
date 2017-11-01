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
	"net/http"
)

// interface for request handlers
type handler interface {
	// all http handlers will implement this function
	Handle(ctx context.Context, w http.ResponseWriter, req *http.Request) (int, error)
}

// clientIPFunc is the function used by handlers to find the client IP address
type clientIPFunc func(req *http.Request) (string, error)
