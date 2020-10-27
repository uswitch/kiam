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
package testutil

import (
	"context"

	"github.com/uswitch/kiam/pkg/aws/sts"
)

type stubCache struct {
	issue func(identity *sts.RoleIdentity) (*sts.Credentials, error)
}

func (i *stubCache) CredentialsForRole(ctx context.Context, identity *sts.RoleIdentity) (*sts.Credentials, error) {
	return i.issue(identity)
}

func (i *stubCache) Expiring() chan *sts.CachedCredentials {
	return make(chan *sts.CachedCredentials)
}

func NewStubCredentialsCache(f func(identity *sts.RoleIdentity) (*sts.Credentials, error)) sts.CredentialsCache {
	return &stubCache{f}
}
