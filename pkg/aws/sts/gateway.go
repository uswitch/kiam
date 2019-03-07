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
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/uswitch/kiam/pkg/statsd"
)

type STSGateway interface {
	Issue(ctx context.Context, role, session string, expiry time.Duration) (*Credentials, error)
}

type DefaultSTSGateway struct {
	session *session.Session
}

func DefaultGateway(assumeRoleArn, region string) *DefaultSTSGateway {
	config := aws.NewConfig()
	if assumeRoleArn != "" {
		config.WithCredentials(stscreds.NewCredentials(session.Must(session.NewSession()), assumeRoleArn))
	}

	if region != "" {
		config.WithRegion(region).WithEndpointResolver(endpoints.ResolverFunc(endpointFor))
	}

	session := session.Must(session.NewSession(config))
	return &DefaultSTSGateway{session: session}
}

func endpointFor(service, region string, opts ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
	var url string

	_, exists := endpoints.PartitionForRegion(endpoints.DefaultPartitions(), region)
	defaultResolver := endpoints.DefaultResolver()

	// if the region doesn't exist or it is a fips endpoint, fallback to the default resolver
	if !exists || strings.HasSuffix(region, "-fips") {
		return defaultResolver.EndpointFor(service, region, opts...)
	}

	if strings.HasPrefix(region, "cn-") {
		url = fmt.Sprintf("https://sts.%s.amazonaws.com.cn", region)
	} else {
		url = fmt.Sprintf("https://sts.%s.amazonaws.com", region)
	}

	return endpoints.ResolvedEndpoint{URL: url, SigningRegion: region}, nil
}

func (g *DefaultSTSGateway) Issue(ctx context.Context, roleARN, sessionName string, expiry time.Duration) (*Credentials, error) {
	timer := prometheus.NewTimer(assumeRole)
	defer timer.ObserveDuration()
	if statsd.Enabled {
		defer statsd.Client.NewTiming().Send("aws.assume_role")
	}

	assumeRoleExecuting.Inc()
	defer assumeRoleExecuting.Dec()

	svc := sts.New(g.session)
	in := &sts.AssumeRoleInput{
		DurationSeconds: aws.Int64(int64(expiry.Seconds())),
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String(sessionName),
	}
	resp, err := svc.AssumeRoleWithContext(ctx, in)
	if err != nil {
		return nil, err
	}

	return NewCredentials(*resp.Credentials.AccessKeyId, *resp.Credentials.SecretAccessKey, *resp.Credentials.SessionToken, *resp.Credentials.Expiration), nil
}
