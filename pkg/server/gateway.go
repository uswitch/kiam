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
package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"github.com/cenkalti/backoff"
	retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/uswitch/kiam/pkg/aws/sts"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/naming"
)

// Client to interact with KiamServer, exposing k8s.RoleFinder and sts.CredentialsProvider interfaces
type KiamGateway struct {
	conn                *grpc.ClientConn
	client              pb.KiamServiceClient
	serviceAccountToken string
}

const (
	RetryInterval = 10 * time.Millisecond
)

type KiamGatewayConfig struct {
	Address             string
	Refresh             time.Duration
	CaFile              string
	CertificateFile     string
	KeyFile             string
	ServiceAccountToken string
}

func NewGateway(ctx context.Context, config *KiamGatewayConfig) (*KiamGateway, error) {
	callOpts := []retry.CallOption{
		retry.WithBackoff(retry.BackoffLinear(RetryInterval)),
	}

	certificate, err := tls.LoadX509KeyPair(config.CertificateFile, config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("error loading keypair: %v", err)
	}
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(config.CaFile)
	if err != nil {
		return nil, fmt.Errorf("error reading SSL cert: %v", err)
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("error appending certs from ca")
	}

	host, _, err := net.SplitHostPort(config.Address)
	if err != nil {
		return nil, fmt.Errorf("error parsing hostname: %v", err)
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName:   host,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
	})

	resolver, err := naming.NewDNSResolverWithFreq(config.Refresh)
	if err != nil {
		return nil, fmt.Errorf("error creating DNS resolver: %v", err)
	}

	balancer := grpc.RoundRobin(resolver)
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(retry.UnaryClientInterceptor(callOpts...)),
		grpc.WithBalancer(balancer),
	}
	conn, err := grpc.Dial(config.Address, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("error dialing grpc server: %v", err)
	}

	lookupAddress := func() error {
		// BlockingWait for the BalancerGetOptions appears to have no effect
		// so we wrap in a retry loop to provide the same behaviour
		_, _, err := balancer.Get(ctx, grpc.BalancerGetOptions{})
		if err != nil {
			return fmt.Errorf("error waiting for address to be available in the balancer: %v", err)
		}
		return nil
	}
	err = backoff.Retry(lookupAddress, backoff.WithContext(backoff.NewConstantBackOff(50*time.Millisecond), ctx))

	client := pb.NewKiamServiceClient(conn)
	return &KiamGateway{conn: conn, client: ClientWithTelemetry(client), serviceAccountToken: config.ServiceAccountToken}, nil
}

func (g *KiamGateway) Close() {
	g.conn.Close()
}

func (g *KiamGateway) FindRoleFromIP(ctx context.Context, ip string) (string, error) {
	role, err := g.client.GetPodRole(ctx, &pb.GetPodRoleRequest{Ip: ip})
	if err != nil {
		return "", err
	}
	return role.Name, nil
}

func (g *KiamGateway) Health(ctx context.Context) (string, error) {
	status, err := g.client.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return "", err
	}
	return status.Message, nil
}

func (g *KiamGateway) IsAllowedAssumeRole(ctx context.Context, role, podIP string) (Decision, error) {
	resp, err := g.client.IsAllowedAssumeRole(ctx, &pb.IsAllowedAssumeRoleRequest{Ip: podIP, Role: &pb.Role{Name: role}})
	if err != nil {
		return nil, err
	}
	return &adaptedDecision{resp.Decision}, nil
}

func (g *KiamGateway) CredentialsForRole(ctx context.Context, role string) (*sts.Credentials, error) {
	credentials, err := g.client.GetRoleCredentials(ctx, &pb.GetRoleCredentialsRequest{
		Role:                &pb.Role{Name: role},
		ServiceAccountToken: g.serviceAccountToken})
	if err != nil {
		return nil, err
	}
	return &sts.Credentials{
		Code:            credentials.Code,
		Type:            credentials.Type,
		AccessKeyId:     credentials.AccessKeyId,
		SecretAccessKey: credentials.SecretAccessKey,
		Token:           credentials.Token,
		Expiration:      credentials.Expiration,
		LastUpdated:     credentials.LastUpdated,
	}, nil
}
