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
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/rcrowley/go-metrics"
	"time"
)

func IssueNewCredentials(roleARN, sessionName string, expiry time.Duration) (*Credentials, error) {
	session, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	timer := metrics.GetOrRegisterTimer("aws.assumeRole", metrics.DefaultRegistry)
	started := time.Now()
	defer timer.UpdateSince(started)

	svc := sts.New(session)
	in := &sts.AssumeRoleInput{
		DurationSeconds: aws.Int64(int64(expiry.Seconds())),
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String(sessionName),
	}
	resp, err := svc.AssumeRole(in)
	if err != nil {
		return nil, err
	}

	return NewCredentials(*resp.Credentials.AccessKeyId, *resp.Credentials.SecretAccessKey, *resp.Credentials.SessionToken, *resp.Credentials.Expiration), nil
}
