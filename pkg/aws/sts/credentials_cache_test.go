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
package sts

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

type stubGateway struct {
	c                    *Credentials
	issueCount           int
	requestedRole        string
	requestedSessionName string
	requestedExternalID  string
}

func (s *stubGateway) Issue(ctx context.Context, request *STSIssueRequest) (*Credentials, error) {
	s.issueCount = s.issueCount + 1
	s.requestedRole = request.RoleARN
	s.requestedSessionName = request.SessionName
	s.requestedExternalID = request.ExternalID

	return s.c, nil
}

func TestRequestsCredentialsFromGatewayWithEmptyCache(t *testing.T) {
	stubGateway := &stubGateway{c: &Credentials{Code: "foo"}}
	cache := DefaultCache(stubGateway, "session", 15*time.Minute, 5*time.Minute)
	ctx := context.Background()

	credentialsIdentity := &RoleIdentity{Role: ResolvedRole{Name: "role", ARN: "arn:account:role"}}
	creds, _ := cache.CredentialsForRole(ctx, credentialsIdentity)
	if creds.Code != "foo" {
		t.Error("didnt return expected credentials code, was", creds.Code)
	}
	if testutil.ToFloat64(cacheSize) != 1 {
		t.Error("expected to cache credential, was", testutil.ToFloat64(cacheSize))
	}

	cache.CredentialsForRole(ctx, credentialsIdentity)
	if stubGateway.issueCount != 1 {
		t.Error("expected creds to be cached")
	}

	if stubGateway.requestedRole != "arn:account:role" {
		t.Error("unexpected role, was:", stubGateway.requestedRole)
	}
}

func TestRequestsCredentialsWithSessionName(t *testing.T) {
	var tests = []struct {
		name                string
		sessionName         string
		expectedSessionName string
	}{
		{"Default", "testing", "kiam-testing"},
		{"InvalidCharsReplacedWithHyphen", "testing@#&-test%", "kiam-testing@---test-"},
		{"LongNameLimitedTo64Chars", "Unsplvku4rP9A71Zb5DUQtKviVKSENh0GlKxVRPXGvfDyXXXy8OGqTVfc05DCAhKT9oHXU", "kiam-Unsplvku4rP9A71Zb5DUQtKviVKSENh0GlKxVRPXGvfDyXXXy8OGqTVfc0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubGateway := &stubGateway{c: &Credentials{Code: "foo"}}
			cache := DefaultCache(stubGateway, "session", 15*time.Minute, 5*time.Minute)
			ctx := context.Background()

			credentialsIdentity := &RoleIdentity{
				Role:        ResolvedRole{Name: "role", ARN: "arn:account:role"},
				SessionName: tt.sessionName,
			}

			_, _ = cache.CredentialsForRole(ctx, credentialsIdentity)

			if stubGateway.requestedSessionName != tt.expectedSessionName {
				t.Error("unexpected session-name, was:", stubGateway.requestedSessionName)
			}
		})
	}
}

func TestRequestsCredentialsWithExternalID(t *testing.T) {
	stubGateway := &stubGateway{c: &Credentials{Code: "foo"}}
	cache := DefaultCache(stubGateway, "session", 15*time.Minute, 5*time.Minute)
	ctx := context.Background()

	credentialsIdentity := &RoleIdentity{
		Role:       ResolvedRole{Name: "role", ARN: "arn:account:role"},
		ExternalID: "123456",
	}

	_, _ = cache.CredentialsForRole(ctx, credentialsIdentity)

	if stubGateway.requestedExternalID != "123456" {
		t.Error("unexpected external-id, was:", stubGateway.requestedExternalID)
	}
}
