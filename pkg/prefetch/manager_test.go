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
package prefetch

import (
	"context"
	"github.com/uswitch/kiam/pkg/aws/sts"
	kt "github.com/uswitch/kiam/pkg/k8s/testing"
	"github.com/uswitch/kiam/pkg/testutil"
	"testing"
	"time"
)

func TestPrefetchRunningPods(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requestedRoles := make(chan string)
	finder := kt.NewStubFinder(nil)
	announcer := kt.NewStubAnnouncer()
	cache := testutil.NewStubCredentialsCache(func(role string) (*sts.Credentials, error) {
		requestedRoles <- role
		return &sts.Credentials{}, nil
	})
	manager := NewManager(cache, finder, announcer)
	go manager.Run(ctx, 1)

	announcer.Announce(testutil.NewPodWithRole("ns", "name", "ip", "Running", "role"))
	role := <-requestedRoles
	if role != "role" {
		t.Error("should have requested role")
	}

	announcer.Announce(testutil.NewPodWithRole("ns", "name", "ip", "Failed", "failed_role"))
	select {
	case role = <-requestedRoles:
		t.Error("didn't expect to request role, but was requested", role)
	case <-time.After(time.Second):
		return
	}
}
