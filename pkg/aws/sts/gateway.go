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
	"net"
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

type regionalResolver struct {
	endpoint endpoints.ResolvedEndpoint
}

func (r *regionalResolver) EndpointFor(svc, region string, opts ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
	return r.endpoint, nil
}

func newRegionalResolver(region string) (endpoints.Resolver, error) {
	var host string

	defaultResolver := endpoints.DefaultResolver()

	// if it is a FIPS region, let the default resolver give us a result.
	if strings.HasSuffix(region, "-fips") {
		endpoint, err := defaultResolver.EndpointFor("sts", region)
		if err != nil {
			return nil, err
		}
		return &regionalResolver{endpoint}, nil
	}

	if _, exists := endpoints.PartitionForRegion(endpoints.DefaultPartitions(), region); !exists {
		return nil, fmt.Errorf("Invalid region: %s", region)
	}

	if strings.HasPrefix(region, "cn-") {
		host = fmt.Sprintf("sts.%s.amazonaws.com.cn", region)
	} else {
		host = fmt.Sprintf("sts.%s.amazonaws.com", region)
	}

	if _, err := net.LookupHost(host); err != nil {
		return nil, fmt.Errorf("Regional STS endpoint does not exist: %s", host)
	}

	return &regionalResolver{endpoints.ResolvedEndpoint{
		URL:           fmt.Sprintf("https://%s", host),
		SigningRegion: region,
	}}, nil
}

type DefaultSTSGateway struct {
	session  *session.Session
	resolver endpoints.Resolver
}

func DefaultGateway(assumeRoleArn, region string) (*DefaultSTSGateway, error) {
        config := aws.NewConfig().WithCredentialsChainVerboseErrors(true)
	if assumeRoleArn != "" {
		config.WithCredentials(stscreds.NewCredentials(session.Must(session.NewSession()), assumeRoleArn))
	}

	if region != "" {
		resolver, err := newRegionalResolver(region)
		if err != nil {
			return nil, err
		}

		config.WithRegion(region).WithEndpointResolver(resolver)
	}

	session := session.Must(session.NewSession(config))
	return &DefaultSTSGateway{session: session}, nil
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
