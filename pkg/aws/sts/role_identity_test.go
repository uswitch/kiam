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
	"testing"
)

const rolePrefix = "arn:aws:iam::account-id:role/"

func TestRoleIdentity(t *testing.T) {
	testCases := []struct {
		role, sessionName, externalID string
		sessionTags                   map[string]string
		expectedIdentity              string
	}{
		{
			role:             "example-role",
			expectedIdentity: "arn:aws:iam::account-id:role/example-role||",
		},
		{
			sessionName:      "db-reader",
			role:             "example-role",
			expectedIdentity: "arn:aws:iam::account-id:role/example-role|db-reader|",
		},
		{
			sessionName:      "db-reader",
			externalID:       "07091992",
			role:             "example-role",
			expectedIdentity: "arn:aws:iam::account-id:role/example-role|db-reader|07091992",
		},
		{
			sessionName:      "db-reader",
			externalID:       "07091992",
			role:             "example-role",
			sessionTags:      map[string]string{"foo": "bar"},
			expectedIdentity: "arn:aws:iam::account-id:role/example-role|db-reader|07091992|foo:bar",
		},
		{
			sessionName: "db-reader",
			externalID:  "07091992",
			role:        "example-role",
			sessionTags: map[string]string{
				"foo":     "bar",
				"another": "example",
			},
			expectedIdentity: "arn:aws:iam::account-id:role/example-role|db-reader|07091992|another:example,foo:bar",
		},
	}
	for _, tc := range testCases {
		resolver := DefaultResolver(rolePrefix)
		desc := fmt.Sprintf("test role=%s, session name=%s, external ID=%s, tags= %v", tc.role, tc.sessionName, tc.externalID, tc.sessionTags)
		t.Run(desc, func(t *testing.T) {
			id, err := NewRoleIdentity(resolver, tc.role, tc.sessionName, tc.externalID, tc.sessionTags)
			if err != nil {
				t.Errorf("NewRoleIdentity() = %v, want nil error", err)
			}
			if id.String() != tc.expectedIdentity {
				t.Errorf("id.String() = %s, want = %s", id.String(), tc.expectedIdentity)
			}
		})
	}
}
