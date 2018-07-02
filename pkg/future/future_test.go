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
	"testing"
	"time"
)

func TestReturnsValue(t *testing.T) {
	f := New(func() (interface{}, error) {
		return "hello", nil
	})
	val, _ := f.Get(context.Background())
	msg, ok := val.(string)
	if !ok || msg != "hello" {
		t.Error("expected hello, was", val)
	}

	val2, _ := f.Get(context.Background())
	msg2, ok := val2.(string)
	if !ok || msg2 != "hello" {
		t.Error("expected hello, was", val2)
	}
}

func TestCancelsWhenBlocked(t *testing.T) {
	f := New(func() (interface{}, error) {
		time.Sleep(1 * time.Second)
		return "bar", nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := f.Get(ctx)
	if err != context.DeadlineExceeded {
		t.Error("Unexpected error:", err.Error())
	}
}

func BenchmarkFutureGet(b *testing.B) {
	f := New(func() (interface{}, error) {
		return 1, nil
	})
	for n := 0; n < b.N; n++ {
		f.Get(context.Background())
	}
}
