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
package server

import (
	"context"
	"k8s.io/api/core/v1"
	"testing"

	"github.com/uswitch/kiam/pkg/aws/sts"
	kt "github.com/uswitch/kiam/pkg/k8s/testing"
	"github.com/uswitch/kiam/pkg/testutil"
)

type fakePolicy struct {
	decision Decision
	err      error
}

func (f fakePolicy) IsAllowedAssumeRole(ctx context.Context, roleName string, pod *v1.Pod) (Decision, error) {
	return f.decision, nil
}

func TestRequestedRolePolicy(t *testing.T) {
	p := testutil.NewPodWithRole("namespace", "name", "192.168.0.1", testutil.PhaseRunning, "myrole")
	f := kt.NewStubFinder(p)

	arnResolver := sts.DefaultResolver("arn:aws:iam::123456789012:role/")
	policy := NewRequestingAnnotatedRolePolicy(f, arnResolver)
	decision, err := policy.IsAllowedAssumeRole(context.Background(), "myrole", p)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !decision.IsAllowed() {
		t.Error("role was same, should have been permitted:", decision.Explanation())
	}

	policy = NewRequestingAnnotatedRolePolicy(f, arnResolver)
	decision, err = policy.IsAllowedAssumeRole(context.Background(), "/myrole", p)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !decision.IsAllowed() {
		t.Error("role was same, should have been permitted:", decision.Explanation())
	}

	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "wrongrole", p)
	if decision.IsAllowed() {
		t.Error("role is different, should be denied", decision.Explanation())
	}

	if decision.Explanation() != "requested 'arn:aws:iam::123456789012:role/wrongrole' but annotated with 'arn:aws:iam::123456789012:role/myrole', forbidden" {
		t.Error("unexpected explanation, was", decision.Explanation())
	}

	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "/wrongrole", p)
	if decision.IsAllowed() {
		t.Error("role is different, should be denied", decision.Explanation())
	}
}

func TestRequestedRolePolicyWithSlash(t *testing.T) {
	arnResolver := sts.DefaultResolver("arn:aws:iam::123456789012:role/")
	p := testutil.NewPodWithRole("namespace", "name", "192.168.0.1", testutil.PhaseRunning, "/myrole")
	f := kt.NewStubFinder(p)

	policy := NewRequestingAnnotatedRolePolicy(f, arnResolver)
	decision, err := policy.IsAllowedAssumeRole(context.Background(), "myrole", p)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !decision.IsAllowed() {
		t.Error("role was same, should have been permitted:", decision.Explanation())
	}

	policy = NewRequestingAnnotatedRolePolicy(f, arnResolver)
	decision, err = policy.IsAllowedAssumeRole(context.Background(), "/myrole", p)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !decision.IsAllowed() {
		t.Error("role was same, should have been permitted:", decision.Explanation())
	}

	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "wrongrole", p)
	if decision.IsAllowed() {
		t.Error("role is different, should be denied", decision.Explanation())
	}

	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "/wrongrole", p)
	if decision.IsAllowed() {
		t.Error("role is different, should be denied", decision.Explanation())
	}
}

func TestNamespacePolicy(t *testing.T) {
	n := testutil.NewNamespace("red", "^red.*$|^.red.*$")
	nf := kt.NewNamespaceFinder(n)
	p := testutil.NewPodWithRole("red", "foo", "192.168.0.1", testutil.PhaseRunning, "red_role")
	arnResolver := sts.DefaultResolver("")

	policy := NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, err := policy.IsAllowedAssumeRole(context.Background(), "red_role", p)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !decision.IsAllowed() {
		t.Errorf("expected to be allowed- pod in correct namespace")
	}

	policy = NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, err = policy.IsAllowedAssumeRole(context.Background(), "/red_role", p)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !decision.IsAllowed() {
		t.Errorf("expected to be allowed- pod in correct namespace")
	}

	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "orange_role", p)
	if decision.IsAllowed() {
		t.Errorf("expected to be forbidden- requesting role that fails regexp")
	}

	if decision.Explanation() != "namespace policy expression '^red.*$|^.red.*$' forbids role 'orange_role'" {
		t.Error("unexpected explanation, was", decision.Explanation())
	}

	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "/orange_role", p)
	if decision.IsAllowed() {
		t.Errorf("expected to be forbidden- requesting role that fails regexp")
	}
}

func TestNamespacePolicyWithSlash(t *testing.T) {
	n := testutil.NewNamespace("red", "^red.*$|^.red.*$")
	nf := kt.NewNamespaceFinder(n)
	p := testutil.NewPodWithRole("red", "foo", "192.168.0.1", testutil.PhaseRunning, "/red_role")
	arnResolver := sts.DefaultResolver("")

	policy := NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, err := policy.IsAllowedAssumeRole(context.Background(), "red_role", p)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !decision.IsAllowed() {
		t.Errorf("expected to be allowed- pod in correct namespace: %s", decision.Explanation())
	}

	policy = NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, err = policy.IsAllowedAssumeRole(context.Background(), "/red_role", p)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !decision.IsAllowed() {
		t.Errorf("expected to be allowed- pod in correct namespace: %s", decision.Explanation())
	}

	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "orange_role", p)
	if decision.IsAllowed() {
		t.Errorf("expected to be forbidden- requesting role that fails regexp")
	}

	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "/orange_role", p)
	if decision.IsAllowed() {
		t.Errorf("expected to be forbidden- requesting role that fails regexp")
	}
}

func TestNotAllowedWithoutNamespaceAnnotation(t *testing.T) {
	n := testutil.NewNamespace("red", "")
	nf := kt.NewNamespaceFinder(n)
	p := testutil.NewPodWithRole("red", "foo", "192.168.0.1", testutil.PhaseRunning, "red_role")
	arnResolver := sts.DefaultResolver("arn:aws:iam::123456789012:role/")

	policy := NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ := policy.IsAllowedAssumeRole(context.Background(), "red_role", p)

	if decision.IsAllowed() {
		t.Error("expected failure, empty namespace policy annotation")
	}

	policy = NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "/red_role", p)

	if decision.IsAllowed() {
		t.Error("expected failure, empty namespace policy annotation")
	}
}

func TestNotAllowedWithoutNamespaceAnnotationWithSlash(t *testing.T) {
	n := testutil.NewNamespace("red", "")
	nf := kt.NewNamespaceFinder(n)
	p := testutil.NewPodWithRole("red", "foo", "192.168.0.1", testutil.PhaseRunning, "/red_role")
	arnResolver := sts.DefaultResolver("arn:aws:iam::123456789012:role/")

	policy := NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ := policy.IsAllowedAssumeRole(context.Background(), "red_role", p)

	if decision.IsAllowed() {
		t.Error("expected failure, empty namespace policy annotation")
	}

	policy = NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "/red_role", p)

	if decision.IsAllowed() {
		t.Error("expected failure, empty namespace policy annotation")
	}
}

func TestAllowedWithARNResolverBaseRole(t *testing.T) {
	n := testutil.NewNamespace("red", "arn:aws:iam::123456789012:role/.*")
	nf := kt.NewNamespaceFinder(n)
	p := testutil.NewPodWithRole("red", "foo", "192.168.0.1", testutil.PhaseRunning, "/red_role")
	arnResolver := sts.DefaultResolver("arn:aws:iam::123456789012:role/")

	policy := NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ := policy.IsAllowedAssumeRole(context.Background(), "red_role", p)

	if !decision.IsAllowed() {
		t.Errorf("expected to be allowed- namespace base role regex match resolver base role: %s", decision.Explanation())
	}

	policy = NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "/red_role", p)

	if !decision.IsAllowed() {
		t.Errorf("expected to be allowed- namespace base role regex match resolver base role: %s", decision.Explanation())
	}
}

func TestNotAllowedWithoutARNResolverBaseRole(t *testing.T) {
	n := testutil.NewNamespace("red", "arn:aws:iam::123456789012:role/*")
	nf := kt.NewNamespaceFinder(n)
	p := testutil.NewPodWithRole("red", "foo", "192.168.0.1", testutil.PhaseRunning, "/red_role")
	arnResolver := sts.DefaultResolver("")

	policy := NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ := policy.IsAllowedAssumeRole(context.Background(), "red_role", p)

	if decision.IsAllowed() {
		t.Error("expected to be forbidden- requesting role that fails base role regexp")
	}

	policy = NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "/red_role", p)

	if decision.IsAllowed() {
		t.Error("expected to be forbidden- requesting role that fails base role regexp")
	}
}

func TestAllowedWithSubPathRegexInNamespace(t *testing.T) {
	n := testutil.NewNamespace("red", ".*/subpath/.*")
	nf := kt.NewNamespaceFinder(n)
	p := testutil.NewPodWithRole("red", "foo", "192.168.0.1", testutil.PhaseRunning, "red_role")
	arnResolver := sts.DefaultResolver("arn:aws:iam::account-id:role/subpath/")

	policy := NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ := policy.IsAllowedAssumeRole(context.Background(), "red_role", p)

	if !decision.IsAllowed() {
		t.Error("expected to be allowed- namespace regex matches role subpath", decision.Explanation())
	}

	policy = NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "/red_role", p)

	if !decision.IsAllowed() {
		t.Error("expected to be allowed- namespace regex matches role subpath", decision.Explanation())
	}
}

func TestNotAllowedWithoutSubPathRegexInNamespace(t *testing.T) {
	n := testutil.NewNamespace("red", "arn:aws:iam::account-id:role/red.*")
	nf := kt.NewNamespaceFinder(n)
	p := testutil.NewPodWithRole("red", "foo", "192.168.0.1", testutil.PhaseRunning, "red_role")
	arnResolver := sts.DefaultResolver("arn:aws:iam::account-id:role/subpath/")

	policy := NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ := policy.IsAllowedAssumeRole(context.Background(), "red_role", p)

	if decision.IsAllowed() {
		t.Error("expected to be forbidden- namespace regex DOES NOT match role subpath")
	}

	policy = NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "/red_role", p)

	if decision.IsAllowed() {
		t.Error("expected to be forbidden- namespace regex DOES NOT match role subpath")
	}
}

func TestAllowedWithExactSubPathInNamespace(t *testing.T) {
	n := testutil.NewNamespace("red", "arn:aws:iam::account-id:role/subpath/red_role")
	nf := kt.NewNamespaceFinder(n)
	p := testutil.NewPodWithRole("red", "foo", "192.168.0.1", testutil.PhaseRunning, "red_role")
	arnResolver := sts.DefaultResolver("arn:aws:iam::account-id:role/subpath/")

	policy := NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ := policy.IsAllowedAssumeRole(context.Background(), "red_role", p)

	if !decision.IsAllowed() {
		t.Error("expected to be allowed- namespace matches role subpath", decision.Explanation())
	}

	policy = NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "/red_role", p)

	if !decision.IsAllowed() {
		t.Error("expected to be allowed- namespace matches role subpath", decision.Explanation())
	}
}

func TestNotAllowedWithoutExactSubPathInNamespace(t *testing.T) {
	n := testutil.NewNamespace("red", "arn:aws:iam::account-id:role/subpath/blue_role")
	nf := kt.NewNamespaceFinder(n)
	p := testutil.NewPodWithRole("red", "foo", "192.168.0.1", testutil.PhaseRunning, "red_role")
	arnResolver := sts.DefaultResolver("arn:aws:iam::account-id:role/subpath/")

	policy := NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ := policy.IsAllowedAssumeRole(context.Background(), "red_role", p)

	if decision.IsAllowed() {
		t.Error("expected to be forbidden- namespace role DOES NOT match role subpath")
	}

	policy = NewNamespacePermittedRoleNamePolicy(nf, arnResolver)
	decision, _ = policy.IsAllowedAssumeRole(context.Background(), "/red_role", p)

	if decision.IsAllowed() {
		t.Error("expected to be forbidden- namespace role DOES NOT match role subpath")
	}
}
