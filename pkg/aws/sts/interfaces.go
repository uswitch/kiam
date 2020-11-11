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
	"context"
)

type CredentialsProvider interface {
	CredentialsForRole(ctx context.Context, identity *RoleIdentity) (*Credentials, error)
}

type CredentialsCache interface {
	CredentialsForRole(ctx context.Context, identity *RoleIdentity) (*Credentials, error)
	Expiring() chan *CachedCredentials
}

// ARNResolver encapsulates resolution of roles into ARNs.
type ARNResolver interface {
	Resolve(role string) (*ResolvedRole, error)
}
