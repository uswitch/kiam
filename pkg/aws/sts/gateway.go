package sts

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/rcrowley/go-metrics"
	"time"
)

type STSGateway interface {
	Issue(role, session string, expiry time.Duration) (*Credentials, error)
}

type DefaultSTSGateway struct {
	session *session.Session
}

func DefaultGateway() *DefaultSTSGateway {
	return &DefaultSTSGateway{session: session.Must(session.NewSession())}
}

func (g *DefaultSTSGateway) Issue(roleARN, sessionName string, expiry time.Duration) (*Credentials, error) {
	timer := metrics.GetOrRegisterTimer("aws.assumeRole", metrics.DefaultRegistry)
	started := time.Now()
	defer timer.UpdateSince(started)

	counter := metrics.GetOrRegisterCounter("stsAssumeRole.executingRequests", metrics.DefaultRegistry)
	counter.Inc(1)
	defer counter.Dec(1)

	svc := sts.New(g.session)
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
