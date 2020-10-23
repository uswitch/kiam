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
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

type awsConfigCredentialsProvider interface {
	NewCredentials(cfg *aws.Config, assumeRoleARN string) *credentials.Credentials
}

type STSCredentialsProvider struct {
}

func (s *STSCredentialsProvider) NewCredentials(cfg *aws.Config, assumeRoleARN string) *credentials.Credentials {
	return stscreds.NewCredentials(session.Must(session.NewSession(cfg)), assumeRoleARN)
}

func NewSTSCredentialsProvider() *STSCredentialsProvider {
	return &STSCredentialsProvider{}
}

type configBuilder struct {
	config *aws.Config
}

// Builds the necessary AWS config for Kiam's server
func NewServerConfigBuilder() *configBuilder {
	return &configBuilder{config: aws.NewConfig().WithCredentialsChainVerboseErrors(true)}
}

// WithRegion configures the *aws.Config with a region and a custom endpoint resolver. With an empty string
// it will not configure.
func (c *configBuilder) WithRegion(region string) (*configBuilder, error) {
	if region == "" {
		return c, nil
	}

	resolver, err := newRegionalEndpointResolver(region)
	if err != nil {
		return nil, err
	}

	c.config.WithRegion(region).WithEndpointResolver(resolver)

	return c, nil
}

func (c *configBuilder) WithCredentialsFromAssumedRole(provider awsConfigCredentialsProvider, assumeRoleARN string) *configBuilder {
	if assumeRoleARN == "" {
		return c
	}

	c.config.WithCredentials(provider.NewCredentials(c.config, assumeRoleARN))
	return c
}

func (c *configBuilder) Config() *aws.Config {
	return c.config
}