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
package kiam

import (
	"context"
	"fmt"
	"github.com/uswitch/kiam/pkg/k8s"
	"github.com/uswitch/kiam/pkg/testutil"
	kt "k8s.io/client-go/tools/cache/testing"
	"testing"
	"time"
)

func TestFindsRunningPod(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	c := k8s.NewPodCache(source, time.Second)
	source.Add(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Failed", "failed_role"))
	source.Add(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Running", "running_role"))
	c.Run(ctx)

	// we'll take to wait for the delta to be processed
	<-c.Pods()

	found, _ := c.FindPodForIP("192.168.0.1")
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
	c := k8s.NewPodCache(source, time.Second)
	source.Add(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Failed", "failed_role"))
	source.Modify(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Failed", "running_role"))
	source.Modify(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Running", "running_role"))
	c.Run(ctx)
	for i := 0; i < 2; i++ {
		<-c.Pods()
	}

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
	c := k8s.NewPodCache(source, time.Second)
	c.Run(ctx)
	for i := 0; i < 1000; i++ {
		source.Add(testutil.NewPodWithRole("ns", fmt.Sprintf("name-%d", i), fmt.Sprintf("ip-%d", i), "Running", "foo_role"))
		<-c.Pods() // wait for delta
	}

	b.StartTimer()

	for n := 0; n < b.N; n++ {
		c.FindPodForIP("ip-500")
	}
}

func BenchmarkIsActiveRole(b *testing.B) {
	b.StopTimer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	c := k8s.NewPodCache(source, time.Second)
	c.Run(ctx)

	// we setup a simulation to approximate usage patterns: many pods with a handful using the same role:
	// 1000 pods but 10 pods per-role. the implementation of the cache degrades as the number of running
	// pods per role increases: there are more slice operations as the number of cache hits increases.
	for i := 0; i < 1000; i++ {
		role := i % 100
		source.Add(testutil.NewPodWithRole("ns", fmt.Sprintf("name-%d", i), fmt.Sprintf("ip-%d", i), "Running", fmt.Sprintf("role-%d", role)))
		<-c.Pods()
	}

	b.StartTimer()

	for n := 0; n < b.N; n++ {
		c.IsActivePodsForRole("role-0")
	}
}
