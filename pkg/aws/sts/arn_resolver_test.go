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

func TestAddsPrefix(t *testing.T) {
	resolver := DefaultResolver("arn:aws:iam::account-id:role/")
	role := resolver.Resolve("myrole")

	if role != "arn:aws:iam::account-id:role/myrole" {
		t.Error("unexpected role, was:", role)
	}
}

func TestAddsPrefixWithRoleBeginningWithSlash(t *testing.T) {
	resolver := DefaultResolver("arn:aws:iam::account-id:role/")
	role := resolver.Resolve("/myrole")

	if role != "arn:aws:iam::account-id:role/myrole" {
		t.Error("unexpected role, was:", role)
	}
}
func TestAddsPrefixWithRoleBeginningWithPathWithoutSlash(t *testing.T) {
	resolver := DefaultResolver("arn:aws:iam::account-id:role/")
	role := resolver.Resolve("kiam/myrole")

	if role != "arn:aws:iam::account-id:role/kiam/myrole" {
		t.Error("unexpected role, was:", role)
	}
}
func TestAddsPrefixWithRoleBeginningWithSlashPath(t *testing.T) {
	resolver := DefaultResolver("arn:aws:iam::account-id:role/")
	role := resolver.Resolve("/kiam/myrole")

	if role != "arn:aws:iam::account-id:role/kiam/myrole" {
		t.Error("unexpected role, was:", role)
	}
}

func TestUsesAbsoluteARN(t *testing.T) {
	resolver := DefaultResolver("arn:aws:iam::account-id:role/")
	role := resolver.Resolve("arn:aws:iam::some-other-account:role/another-role")

	if role != "arn:aws:iam::some-other-account:role/another-role" {
		t.Error("unexpected role, was:", role)
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
