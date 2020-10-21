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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	b := NewServerConfigBuilder()

	if b.Config().Region != nil {
		t.Error("expected nil region, was", *b.Config().Region)
	}

	if !*b.Config().CredentialsChainVerboseErrors {
		t.Error("expected verbose errors")
	}
}

func TestConfigWithRegion(t *testing.T) {
	b, _ := NewServerConfigBuilder().WithRegion(endpoints.UsEast1RegionID)

	if *b.Config().Region != endpoints.UsEast1RegionID {
		t.Error("unexpected region", *b.Config().Region)
	}

	// it should also configure with our custom endpoint resolver
	_, ok := b.Config().EndpointResolver.(*regionalEndpointResolver)
	if !ok {
		t.Errorf("expected endpoint resolver to be castable to *regionalEndpointResolver, was %T", b.Config().EndpointResolver)
	}
}

func TestConfigDoesntUseRegionalResolverWithEmptyRegion(t *testing.T) {
	b, _ := NewServerConfigBuilder().WithRegion("")

	if b.Config().Region != nil {
		t.Error("expected no region, was", *b.Config().Region)
	}
}

func TestWithCredentials(t *testing.T) {
	const accessKeyID = "id"
	creds := credentials.NewStaticCredentials(accessKeyID, "secret", "token")
	p := newStubCredentialsProvider(creds)

	b := NewServerConfigBuilder()
	b.WithCredentialsFromAssumedRole(p, "my test role")

	if b.Config().Credentials != creds {
		t.Errorf("expected same credentials, was %v", b.Config().Credentials)
	}

	v, _ := b.Config().Credentials.Get()
	if v.AccessKeyID != accessKeyID {
		t.Error("unexpected access key", v.AccessKeyID)
	}
}

func TestWithEmptyAssumeRole(t *testing.T) {
	creds := credentials.NewStaticCredentials("foo", "secret", "token")
	p := newStubCredentialsProvider(creds)

	b := NewServerConfigBuilder()
	b.WithCredentialsFromAssumedRole(p, "")

	if p.calls != 0 {
		t.Error("shouldn't have called provider with empty role")
	}
}

func TestConfiguresWithCredentialsFromProvider(t *testing.T) {
	const accessKeyID = "AccessKeyID-example"
	creds := credentials.NewStaticCredentials(accessKeyID, "secret", "token")
	stubProvider := newStubCredentialsProvider(creds)

	builder := NewServerConfigBuilder()
	builder.WithCredentialsFromAssumedRole(stubProvider, "my test role")

	c, _ := builder.Config().Credentials.Get()
	if c.AccessKeyID != accessKeyID {
		t.Errorf("expected id as access key, was %s", c.AccessKeyID)
	}
}

func TestProvidesConfigurationToCredentialsProvider(t *testing.T) {
	creds := credentials.NewStaticCredentials("foo", "secret", "token")
	stubProvider := newStubCredentialsProvider(creds)

	builder := NewServerConfigBuilder()
	builder.WithCredentialsFromAssumedRole(stubProvider, "my test role")

	if stubProvider.requestedConfig != builder.Config() {
		t.Error("expected builder config to be passed to credentials provider")
	}
}


func newStubCredentialsProvider(creds *credentials.Credentials) *stubCredentialsProvider {
	return &stubCredentialsProvider{
		credentials: creds,
		requestedConfig: nil,
	}
}

type stubCredentialsProvider struct {
	credentials *credentials.Credentials
	requestedConfig *aws.Config
	calls int
}

func (s *stubCredentialsProvider) NewCredentials(cfg *aws.Config, assumeRoleARN string) *credentials.Credentials {
	s.requestedConfig = cfg
	s.calls += 1
	return s.credentials
}
