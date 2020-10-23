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
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"net"
	"strings"
)

// regionalEndpointResolver will override behaviour when returning endpoints for STS endpoints
// allowing the return of non-global regions.
type regionalEndpointResolver struct {
	endpoint endpoints.ResolvedEndpoint
	resolver endpoints.Resolver
}

// regionalHostname generates a regional hostname for STS. It uses DNS to verify whether
// the calculated name is correct, and returns an error if not.
func regionalHostname(region string) (string, error) {
	// more endpoints:
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_enable-regions.html#id_credentials_temp_enable-regions_writing_code
	// most regional variations follow this pattern
	hostname := fmt.Sprintf("sts.%s.amazonaws.com", region)

	// chinese regions use .com.cn
	if strings.HasPrefix(region, "cn-") {
		hostname = fmt.Sprintf("%s.cn", hostname)
	}

	// iso regions are airgapped, https://github.com/uswitch/kiam/issues/410 has more context
	// but just follows a different pattern
	if strings.HasPrefix(region, "us-iso") {
		hostname = fmt.Sprintf("sts.%s.c2s.ic.gov", region)
	}

	if _, err := net.LookupHost(hostname); err != nil {
		return "", fmt.Errorf("Regional STS endpoint does not exist: %s", hostname)
	}

	return hostname, nil
}

func newRegionalEndpointResolver(region string) (endpoints.Resolver, error) {
	// Unspecified and FIPs regions follow default resolver pattern
	if region == "" || strings.Contains(region,"fips") {
		return endpoints.DefaultResolver(), nil
	}

	host, err := regionalHostname(region)

	if err != nil {
		return nil, err
	}

	return &regionalEndpointResolver{
		endpoint: endpoints.ResolvedEndpoint{URL: fmt.Sprintf("https://%s", host), SigningRegion: region},
		resolver: endpoints.DefaultResolver(),
	}, nil
}

func (r *regionalEndpointResolver) EndpointFor(svc, region string, opts ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
	if svc != endpoints.StsServiceID {
		return r.resolver.EndpointFor(svc, region, opts...)
	}

	return r.endpoint, nil
}