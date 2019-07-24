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

// DefaultResolver will add the prefix to any roles which
// don't start with arn:
func DefaultResolver(prefix string) *Resolver {
	return &Resolver{prefix: prefix}
}

// Resolve converts from a role string into the absolute role arn.
func (r *Resolver) Resolve(role string) (string, string) {

	if role == "" {
		return "", ""
	}

	components := strings.Split(role, "|")

	externalID := ""

	if len(components) > 1 {
		externalID = components[1]
	}

	role = components[0]

	if strings.HasPrefix(role, "/") {
		role = strings.TrimPrefix(role, "/")
	}

	if strings.HasPrefix(role, "arn:") {
		return role, externalID
	}

	return fmt.Sprintf("%s%s", r.prefix, role), externalID
}
