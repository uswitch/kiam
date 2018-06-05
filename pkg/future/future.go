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
package future

import (
	"context"
)

type Future struct {
	val  interface{}
	err  error
	done chan struct{}
}

type FutureFn func() (interface{}, error)

func (f *Future) Get(ctx context.Context) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-f.done:
		return f.val, f.err
	}
}

func New(f FutureFn) *Future {
	future := &Future{
		done: make(chan struct{}),
	}
	go func() {
		future.val, future.err = f()
		close(future.done)
	}()
	return future
}
