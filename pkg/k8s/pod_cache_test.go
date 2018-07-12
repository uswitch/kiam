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
package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/uswitch/kiam/pkg/testutil"
	kt "k8s.io/client-go/tools/cache/testing"
)

const bufferSize = 10

func TestFindsRunningPod(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	c := NewPodCache(source, time.Second, bufferSize)
	source.Add(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Failed", "failed_role"))
	source.Add(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Running", "running_role"))
	c.Run(ctx)

	found, _ := c.GetPodByIP("192.168.0.1")
	if found == nil {
		t.Error("should have found pod")
	}
	if found.ObjectMeta.Annotations["iam.amazonaws.com/role"] != "running_role" {
		t.Error("wrong role found")
	}
}

func TestFindRoleActive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	c := NewPodCache(source, time.Second, bufferSize)
	source.Add(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Failed", "failed_role"))
	source.Modify(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Failed", "running_role"))
	source.Modify(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Running", "running_role"))
	c.Run(ctx)

	active, _ := c.IsActivePodsForRole("failed_role")
	if active {
		t.Error("expected no active pods in failed_role")
	}

	active, _ = c.IsActivePodsForRole("running_role")
	if !active {
		t.Error("expected running pod")
	}
}

func BenchmarkFindPodsByIP(b *testing.B) {
	b.StopTimer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	c := NewPodCache(source, time.Second, bufferSize)
	for i := 0; i < 1000; i++ {
		source.Add(testutil.NewPodWithRole("ns", fmt.Sprintf("name-%d", i), fmt.Sprintf("ip-%d", i), "Running", "foo_role"))
	}
	c.Run(ctx)

	b.StartTimer()

	for n := 0; n < b.N; n++ {
		c.GetPodByIP("ip-500")
	}
}

func BenchmarkIsActiveRole(b *testing.B) {
	b.StopTimer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	// we setup a simulation to approximate usage patterns: many pods with a handful using the same role:
	// 1000 pods but 10 pods per-role. the implementation of the cache degrades as the number of running
	// pods per role increases: there are more slice operations as the number of cache hits increases.
	for i := 0; i < 1000; i++ {
		role := i % 100
		source.Add(testutil.NewPodWithRole("ns", fmt.Sprintf("name-%d", i), fmt.Sprintf("ip-%d", i), "Running", fmt.Sprintf("role-%d", role)))
	}
	c := NewPodCache(source, time.Second, bufferSize)
	c.Run(ctx)

	b.StartTimer()

	for n := 0; n < b.N; n++ {
		c.IsActivePodsForRole("role-0")
	}
}
