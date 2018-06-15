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
	"testing"
	"time"

	"github.com/uswitch/kiam/pkg/aws/sts"
)

type stubGateway struct {
	c             *sts.Credentials
	issueCount    int
	requestedRole string
}

func (s *stubGateway) Issue(ctx context.Context, roleARN, sessionName string, expiry time.Duration) (*sts.Credentials, error) {
	s.issueCount = s.issueCount + 1
	s.requestedRole = roleARN
	return s.c, nil
}

func TestRequestsCredentialsFromGatewayWithEmptyCache(t *testing.T) {
	stubGateway := &stubGateway{c: &sts.Credentials{Code: "foo"}}
	cache := sts.DefaultCache(stubGateway, "session", 15*time.Minute, 5*time.Minute, sts.DefaultResolver("prefix:"))
	ctx := context.Background()

	creds, _ := cache.CredentialsForRole(ctx, "role")
	if creds.Code != "foo" {
		t.Error("didnt return expected credentials code, was", creds.Code)
	}

	cache.CredentialsForRole(ctx, "role")
	if stubGateway.issueCount != 1 {
		t.Error("expected creds to be cached")
	}

	if stubGateway.requestedRole != "prefix:role" {
		t.Error("unexpected role, was:", stubGateway.requestedRole)
	}
}
