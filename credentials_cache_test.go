package main

import (
	"context"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"testing"
	"time"
)

type stubGateway struct {
	c          *sts.Credentials
	issueCount int
}

func (s *stubGateway) Issue(roleARN, sessionName string, expiry time.Duration) (*sts.Credentials, error) {
	s.issueCount = s.issueCount + 1
	return s.c, nil
}

func TestRequestsCredentialsFromGatewayWithEmptyCache(t *testing.T) {
	stubGateway := &stubGateway{c: &sts.Credentials{Code: "foo"}}
	cache := sts.DefaultCache(stubGateway, "arn", "session")
	ctx := context.Background()

	creds, _ := cache.CredentialsForRole(ctx, "role")
	if creds.Code != "foo" {
		t.Error("didnt return expected credentials code, was", creds.Code)
	}

	cache.CredentialsForRole(ctx, "role")
	if stubGateway.issueCount != 1 {
		t.Error("expected creds to be cached")
	}
}
