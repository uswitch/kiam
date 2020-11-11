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
	"fmt"
	"strings"
)

type Resolver struct {
	prefix string
}

type ResolvedRole struct {
	Name string
	ARN  string
}

// DefaultResolver will add the prefix to any roles which
// don't start with arn:
func DefaultResolver(prefix string) *Resolver {
	return &Resolver{prefix: prefix}
}

// Resolve converts from a role string into the absolute role arn.
func (r *Resolver) Resolve(role string) (*ResolvedRole, error) {
	if role == "" {
		return nil, fmt.Errorf("role can't be empty")
	}

	if strings.HasPrefix(role, "arn:") {
		return &ResolvedRole{ARN: role, Name: roleFromArn(role)}, nil
	}

	if strings.HasPrefix(role, "/") {
		role = strings.TrimPrefix(role, "/")
	}

	return &ResolvedRole{ARN: fmt.Sprintf("%s%s", r.prefix, role), Name: role}, nil
}

// arn:aws:iam::account-id:role/role-name-with-path
func roleFromArn(arn string) string {
	splits := strings.SplitAfterN(arn, ":", 6)
	return strings.TrimPrefix(splits[5], "role/")
}

func (i *ResolvedRole) Equals(other *ResolvedRole) bool {
	return *i == *other
}
