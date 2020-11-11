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
	"testing"
)

func TestResolvedRoleEquality(t *testing.T) {
	resolver := DefaultResolver("arn:aws:iam::account-id:role/")
	i1, _ := resolver.Resolve("foo")
	i2, _ := resolver.Resolve("foo")
	i3, _ := resolver.Resolve("bar")

	if !i1.Equals(i2) {
		t.Error("expected equal")
	}

	if i1.Equals(i3) {
		t.Error("unexpected equality")
	}
}

func TestAddsPrefix(t *testing.T) {
	resolver := DefaultResolver("arn:aws:iam::account-id:role/")
	resolvedRole, _ := resolver.Resolve("myrole")

	if resolvedRole.ARN != "arn:aws:iam::account-id:role/myrole" {
		t.Error("unexpected role, was:", resolvedRole.ARN)
	}
}

func TestReturnsErrorForEmptyRole(t *testing.T) {
	resolver := DefaultResolver("arn:aws:iam::account-id:role/")
	_, err := resolver.Resolve("")

	if err == nil {
		t.Error("should've returned an error for empty role")
	}
}

func TestAddsPrefixWithRoleBeginningWithSlash(t *testing.T) {
	resolver := DefaultResolver("arn:aws:iam::account-id:role/")
	resolvedRole, _ := resolver.Resolve("/myrole")

	if resolvedRole.ARN != "arn:aws:iam::account-id:role/myrole" {
		t.Error("unexpected role, was:", resolvedRole.ARN)
	}

	if resolvedRole.Name != "myrole" {
		t.Error("unexpected role, was", resolvedRole.Name)
	}
}
func TestAddsPrefixWithRoleBeginningWithPathWithoutSlash(t *testing.T) {
	resolver := DefaultResolver("arn:aws:iam::account-id:role/")
	resolvedRole, _ := resolver.Resolve("kiam/myrole")

	if resolvedRole.ARN != "arn:aws:iam::account-id:role/kiam/myrole" {
		t.Error("unexpected role, was:", resolvedRole.ARN)
	}

	if resolvedRole.Name != "kiam/myrole" {
		t.Error("unexpected role", resolvedRole.Name)
	}
}
func TestAddsPrefixWithRoleBeginningWithSlashPath(t *testing.T) {
	resolver := DefaultResolver("arn:aws:iam::account-id:role/")
	resolvedRole, _ := resolver.Resolve("/kiam/myrole")

	if resolvedRole.ARN != "arn:aws:iam::account-id:role/kiam/myrole" {
		t.Error("unexpected role, was:", resolvedRole.ARN)
	}
}

func TestUsesAbsoluteARN(t *testing.T) {
	resolver := DefaultResolver("arn:aws:iam::account-id:role/")
	resolvedRole, _ := resolver.Resolve("arn:aws:iam::some-other-account:role/path-prefix/another-role")

	if resolvedRole.ARN != "arn:aws:iam::some-other-account:role/path-prefix/another-role" {
		t.Error("unexpected role, was:", resolvedRole.ARN)
	}

	if resolvedRole.Name != "path-prefix/another-role" {
		t.Error("expected role to be set, was", resolvedRole.Name)
	}
}

func TestExtractsBaseFromInstanceArn(t *testing.T) {
	prefix, _ := BaseArn("arn:aws:iam::account-id:instance-profile/instance-role-name")
	if prefix != "arn:aws:iam::account-id:role/" {
		t.Error("unexpected prefix, was: ", prefix)
	}

	prefix, _ = BaseArn("arn:aws:iam::account-id:instance-profile/mypath/instance-role-name")
	if prefix != "arn:aws:iam::account-id:role/" {
		t.Error("unexpected prefix, was: ", prefix)
	}
}
