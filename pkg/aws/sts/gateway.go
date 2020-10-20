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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/prometheus/client_golang/prometheus"
)

type STSGateway interface {
	Issue(ctx context.Context, role, session string, expiry time.Duration) (*Credentials, error)
}

type DefaultSTSGateway struct {
	session  *session.Session
}

func DefaultGateway(config *aws.Config) (*DefaultSTSGateway, error) {
	return &DefaultSTSGateway{session: session.Must(session.NewSession(config))}, nil
}

func (g *DefaultSTSGateway) Issue(ctx context.Context, roleARN, sessionName string, expiry time.Duration) (*Credentials, error) {
	timer := prometheus.NewTimer(assumeRole)
	defer timer.ObserveDuration()

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
