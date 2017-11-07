package kiam

import (
	"context"
	"github.com/uswitch/kiam/pkg/server"
	"github.com/uswitch/kiam/pkg/testutil"
	"testing"
)

func TestRequestedRolePolicy(t *testing.T) {
	p := testutil.NewPodWithRole("namespace", "name", "192.168.0.1", testutil.PhaseRunning, "myrole")
	f := testutil.NewStubFinder(p)

	policy := server.NewRequestingAnnotatedRolePolicy(f)
	decision, err := policy.IsAllowedAssumeRole(context.Background(), "myrole", "192.168.0.1")
	if err != nil {
		t.Fail()
	}

	if !decision.IsAllowed() {
		t.Error("role was same, should have been permitted:", decision.Explanation())
	}

	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "wrongrole", "192.168.0.1")
	if decision.IsAllowed() {
		t.Error("role is different, should be denied", decision.Explanation())
	}
}
