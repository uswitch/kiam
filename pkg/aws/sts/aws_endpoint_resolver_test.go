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
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"testing"
)

func TestUsesDefaultForOtherServices(t *testing.T) {
	r, _ := newRegionalEndpointResolver("eu-west-1")
	rd, err := r.EndpointFor(endpoints.S3ServiceID, endpoints.EuWest1RegionID)
	if err != nil {
		t.Error(err)
	}
	if rd.URL != "https://s3.eu-west-1.amazonaws.com" {
		t.Error("unexpected", rd.URL)
	}
}

func TestResolvesDefaultRegion(t *testing.T) {
	resolver, _ := newRegionalEndpointResolver("")

	resolved, err := resolver.EndpointFor(endpoints.StsServiceID, "")
	if err != nil {
		t.Error(err)
	}

	if resolved.URL != "https://sts.amazonaws.com" {
		t.Error("unexpected:", resolved.URL)
	}
}

func TestResolvesUsingSpecifiedRegion(t *testing.T) {
	resolver, _ := newRegionalEndpointResolver("us-west-2")
	resolved, err := resolver.EndpointFor(endpoints.StsServiceID, "")
	if err != nil {
		t.Error(err)
	}

	if resolved.URL != "https://sts.us-west-2.amazonaws.com" {
		t.Error("unexpected:", resolved.URL)
	}
}

func TestResolvesEURegion(t *testing.T) {
	resolver, _ := newRegionalEndpointResolver("eu-west-1")
	resolved, err := resolver.EndpointFor(endpoints.StsServiceID, "")
	if err != nil {
		t.Error(err)
	}

	if resolved.URL != "https://sts.eu-west-1.amazonaws.com" {
		t.Error("unexpected:", resolved.URL)
	}
}

func TestAddsChinaPrefixForChineseRegions(t *testing.T) {
	resolver, err := newRegionalEndpointResolver("cn-north-1")
	if err != nil {
		t.Error(err)
	}

	resolved, err := resolver.EndpointFor(endpoints.StsServiceID, "")
	if err != nil {
		t.Error(err)
	}

	if resolved.URL != "https://sts.cn-north-1.amazonaws.com.cn" {
		t.Error("unexpected:", resolved.URL)
	}
}

func TestUseDefaultForFIPS(t *testing.T) {
	r, e := newRegionalEndpointResolver("us-east-1-fips")
	if e != nil {
		t.Error(e)
	}

	rd, e := r.EndpointFor(endpoints.StsServiceID, "us-east-1-fips")
	if e != nil {
		t.Error(e)
	}

	if rd.URL != "https://sts-fips.us-east-1.amazonaws.com" {
		t.Error("unexpected", rd.URL)
	}
}

func TestGovGateway(t *testing.T) {
	r, e := newRegionalEndpointResolver("us-gov-east-1")
	if e != nil {
		t.Error(e)
	}

	rd, e := r.EndpointFor(endpoints.StsServiceID, "us-gov-east-1")
	if e != nil {
		t.Error(e)
	}

	if rd.URL != "https://sts.us-gov-east-1.amazonaws.com" {
		t.Error("unexpected", rd.URL)
	}
}

// https://github.com/uswitch/kiam/issues/410
func TestAirgappedRegion(t *testing.T) {
	r, e := newRegionalEndpointResolver("us-iso-east-1")
	if e != nil {
		t.Error(e)
	}

	rd, e := r.EndpointFor(endpoints.StsServiceID, "us-iso-east-1")
	if e != nil {
		t.Error(e)
	}

	if rd.URL != "https://sts.us-iso-east-1.c2s.ic.gov" {
		t.Error("unexpected", rd.URL)
	}
}
