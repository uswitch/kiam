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

	"github.com/fortytw2/leaktest"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/testutil"
	kt "k8s.io/client-go/tools/cache/testing"
)

const bufferSize = 10

func TestFindsRunningPod(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	arnResolver := sts.DefaultResolver("arn:account:")
	c := NewPodCache(arnResolver, source, time.Second, bufferSize)
	source.Add(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Failed", "failed_role"))
	source.Add(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Running", "running_role"))
	c.Run(ctx)
	defer source.Shutdown()

	found, _ := c.GetPodByIP("192.168.0.1")
	if found == nil {
		t.Error("should have found pod")
	}
	if found.ObjectMeta.Annotations["iam.amazonaws.com/role"] != "running_role" {
		t.Error("wrong role found")
	}
}

func TestFindRoleActive(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	arnResolver := sts.DefaultResolver("arn:account:")
	c := NewPodCache(arnResolver, source, time.Second, bufferSize)
	source.Add(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Failed", "failed_role"))
	source.Modify(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Failed", "running_role"))
	source.Modify(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Running", "running_role"))
	c.Run(ctx)
	defer source.Shutdown()

	identity, _ := sts.NewRoleIdentity(arnResolver, "failed_role", "", "")
	active, _ := c.IsActivePodsForRole(identity)
	if active {
		t.Error("expected no active pods in failed_role")
	}

	identity, _ = sts.NewRoleIdentity(arnResolver, "running_role", "", "")
	active, _ = c.IsActivePodsForRole(identity)
	if !active {
		t.Error("expected running pod")
	}
}

func TestFindRoleActiveWithSessionName(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	arnResolver := sts.DefaultResolver("arn:account:")
	c := NewPodCache(arnResolver, source, time.Second, bufferSize)
	source.Add(testutil.NewPodWithSessionName("ns", "active-reader", "192.168.0.1", "Running", "reader", "active-reader"))
	source.Add(testutil.NewPodWithSessionName("ns", "stopped-reader", "192.168.0.2", "Succeeded", "reader", "stopped-reader"))
	c.Run(ctx)
	defer source.Shutdown()

	identity, _ := sts.NewRoleIdentity(arnResolver, "reader", "active-reader", "")
	active, _ := c.IsActivePodsForRole(identity)
	if !active {
		t.Error("expected running pod for active-reader")
	}

	identity, _ = sts.NewRoleIdentity(arnResolver, "reader", "stopped-reader", "")
	active, _ = c.IsActivePodsForRole(identity)
	if active {
		t.Error("expected no active pods for stopped-reader")
	}
}

func TestFindRoleActiveWithExternalID(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	arnResolver := sts.DefaultResolver("arn:account:")
	c := NewPodCache(arnResolver, source, time.Second, bufferSize)
	source.Add(testutil.NewPodWithExternalID("ns", "active-reader", "192.168.0.1", "Running", "reader", "1234"))
	source.Add(testutil.NewPodWithExternalID("ns", "stopped-reader", "192.168.0.2", "Succeeded", "reader", "4321"))
	c.Run(ctx)
	defer source.Shutdown()

	identity, _ := sts.NewRoleIdentity(arnResolver, "reader", "", "1234")
	active, _ := c.IsActivePodsForRole(identity)
	if !active {
		t.Error("expected running pod for active-reader")
	}

	identity, _ = sts.NewRoleIdentity(arnResolver, "reader", "", "4321")
	active, _ = c.IsActivePodsForRole(identity)
	if active {
		t.Error("expected no active pods for stopped-reader")
	}
}

func BenchmarkFindPodsByIP(b *testing.B) {
	b.StopTimer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	arnResolver := sts.DefaultResolver("arn:account:")
	c := NewPodCache(arnResolver, source, time.Second, bufferSize)
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
	arnResolver := sts.DefaultResolver("arn:account:")
	c := NewPodCache(arnResolver, source, time.Second, bufferSize)
	c.Run(ctx)

	b.StartTimer()

	for n := 0; n < b.N; n++ {
		identity, _ := sts.NewRoleIdentity(arnResolver, "role-0", "", "")
		c.IsActivePodsForRole(identity)
	}
}
