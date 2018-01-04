package kiam

import (
	"context"
	"fmt"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"testing"
	"time"
)

type stubGateway struct {
	c          *sts.Credentials
	issueCount int
}

func (s *stubGateway) Issue(ctx context.Context, roleARN, sessionName string, expiry time.Duration) (*sts.Credentials, error) {
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

type arnExpectingGateway struct {
	expectedARN string
	c           *sts.Credentials
}

func (s *arnExpectingGateway) Issue(ctx context.Context, roleARN, sessionName string, expiry time.Duration) (*sts.Credentials, error) {
	if roleARN != s.expectedARN {
		return nil, fmt.Errorf("expected %s but was %s", s.expectedARN, roleARN)
	}
	return s.c, nil
}

func TestGeneratesCorrectARN(t *testing.T) {
	g := &arnExpectingGateway{expectedARN: "arn:aws:iam::account-id:role/myrole", c: &sts.Credentials{}}
	cache := sts.DefaultCache(g, "arn:aws:iam::account-id:role/", "session")

	_, err := cache.CredentialsForRole(context.Background(), "myrole")
	if err != nil {
		t.Error(err)
	}
}

func TestUsesAbsoluteARN(t *testing.T) {
	g := &arnExpectingGateway{expectedARN: "arn:aws:iam::another-account:role/foorole", c: &sts.Credentials{}}
	cache := sts.DefaultCache(g, "arn:aws:iam::account-id:role/", "session")

	_, err := cache.CredentialsForRole(context.Background(), "arn:aws:iam::another-account:role/foorole")
	if err != nil {
		t.Error(err)
	}
}
