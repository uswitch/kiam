package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/fortytw2/leaktest"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/k8s"
	"github.com/uswitch/kiam/pkg/testutil"
	pb "github.com/uswitch/kiam/proto"
	kt "k8s.io/client-go/tools/cache/testing"
)

const (
	defaultBuffer = 10
)

func TestErrorSimplification(t *testing.T) {
	e := awserr.NewRequestFailure(awserr.New("code", "message", fmt.Errorf("foo")), 403, "abcdef")
	simplified := simplifyAWSErrorMessage(e)

	if simplified != "code: message" {
		t.Errorf("unexpected: %s", simplified)
	}

	simplified = simplifyAWSErrorMessage(fmt.Errorf("foo"))
	if simplified != "foo" {
		t.Errorf("expected foo, got: %s", simplified)
	}
}

func TestReturnsErrorWhenPodNotFound(t *testing.T) {
	defer leaktest.Check(t)()

	source := kt.NewFakeControllerSource()
	defer source.Shutdown()

	podCache := k8s.NewPodCache(source, time.Second, defaultBuffer)
	server := &KiamServer{pods: podCache}

	_, err := server.GetPodCredentials(context.Background(), &pb.GetPodCredentialsRequest{})

	if err != ErrPodNotFound {
		t.Error("unexpected error:", err)
	}
}

func TestReturnsPolicyErrorWhenForbidden(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	defer source.Shutdown()
	source.Add(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Running", "running_role"))

	podCache := k8s.NewPodCache(source, time.Second, defaultBuffer)
	podCache.Run(ctx)
	server := &KiamServer{pods: podCache, assumePolicy: &forbidPolicy{}}

	_, err := server.GetPodCredentials(ctx, &pb.GetPodCredentialsRequest{Ip: "192.168.0.1"})

	if err != ErrPolicyForbidden {
		t.Error("unexpected error:", err)
	}
}

func TestReturnsCredentials(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := kt.NewFakeControllerSource()
	defer source.Shutdown()
	source.Add(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Running", "running_role"))

	podCache := k8s.NewPodCache(source, time.Second, defaultBuffer)
	podCache.Run(ctx)
	server := &KiamServer{pods: podCache, assumePolicy: &allowPolicy{}, credentialsProvider: &stubCredentialsProvider{accessKey: "A1234"}}

	creds, err := server.GetPodCredentials(ctx, &pb.GetPodCredentialsRequest{Ip: "192.168.0.1"})
	if err != nil {
		t.Error("unexpected error", err)
	}

	if creds == nil {
		t.Fatal("credentials were nil")
	}

	if creds.AccessKeyId != "A1234" {
		t.Error("unexpected access key", creds.AccessKeyId)
	}
}

type stubCredentialsProvider struct {
	accessKey string
}

func (c *stubCredentialsProvider) CredentialsForRole(ctx context.Context, identity *sts.CredentialsIdentity) (*sts.Credentials, error) {
	return &sts.Credentials{
		AccessKeyId: c.accessKey,
	}, nil
}

type forbidPolicy struct {
}

func (f *forbidPolicy) IsAllowedAssumeRole(ctx context.Context, roleName, podIP string) (Decision, error) {
	return &decision{allowed: false, explanation: "uh uh uh"}, nil
}

type allowPolicy struct {
}

func (a *allowPolicy) IsAllowedAssumeRole(ctx context.Context, roleName, podIP string) (Decision, error) {
	return &decision{allowed: true, explanation: "always"}, nil
}

type decision struct {
	allowed     bool
	explanation string
}

func (d *decision) IsAllowed() bool {
	return d.allowed
}

func (d *decision) Explanation() string {
	return d.explanation
}
