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
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/uswitch/kiam/pkg/aws/sts"
	kt "github.com/uswitch/kiam/pkg/k8s/testing"
	"github.com/uswitch/kiam/pkg/testutil"
)

func TestPrefetchRunningPods(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requestedRoles := make(chan string)
	announcer := kt.NewStubAnnouncer()
	cache := testutil.NewStubCredentialsCache(func(identity *sts.RoleIdentity) (*sts.Credentials, error) {
		requestedRoles <- identity.Role.Name
		return &sts.Credentials{}, nil
	})
	manager := NewManager(cache, announcer, sts.DefaultResolver("prefix"))
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

func TestRenewsCredentialsForRunningPod(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requested := make(chan *sts.RoleIdentity, 2)
	credentials := &sts.Credentials{}
	cache := testutil.NewStubCredentialsCache(func(identity *sts.RoleIdentity) (*sts.Credentials, error) {
		requested <- identity
		return credentials, nil
	})
	announcer := kt.NewStubAnnouncer()
	manager := NewManager(cache, announcer, sts.DefaultResolver("prefix"))
	go manager.Run(ctx, 1)

	announcer.Announce(testutil.NewPodWithRole("ns", "name", "ip", "Running", "role"))
	identity := <-requested
	// we'll expire them, triggering them being re-requested
	cache.Expire(&sts.CachedCredentials{Identity: identity, Credentials: credentials})

	select {
	case _ = <-requested:
		// success, re-requested
	case <-time.After(time.Second):
		t.Error("fail, didn't re-request expiring credentials in time")
	}
}

func TestPodSessionName(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requested := make(chan *sts.RoleIdentity)
	announcer := kt.NewStubAnnouncer()
	cache := testutil.NewStubCredentialsCache(func(identity *sts.RoleIdentity) (*sts.Credentials, error) {
		requested <- identity
		return &sts.Credentials{}, nil
	})

	manager := NewManager(cache, announcer, sts.DefaultResolver("prefix"))
	go manager.Run(ctx, 1)

	announcer.Announce(testutil.NewPodWithSessionName("ns", "name", "ip", "Running", "role", "session-name"))
	identity := <-requested
	if identity.SessionName != "session-name" {
		t.Error("should have requested session-name")
	}
}

func TestPodExternalID(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requested := make(chan *sts.RoleIdentity)
	announcer := kt.NewStubAnnouncer()
	cache := testutil.NewStubCredentialsCache(func(identity *sts.RoleIdentity) (*sts.Credentials, error) {
		requested <- identity
		return &sts.Credentials{}, nil
	})

	manager := NewManager(cache, announcer, sts.DefaultResolver("prefix"))
	go manager.Run(ctx, 1)

	announcer.Announce(testutil.NewPodWithExternalID("ns", "name", "ip", "Running", "role", "external-id"))
	identity := <-requested
	if identity.ExternalID != "external-id" {
		t.Error("should have requested external-id")
	}
}
