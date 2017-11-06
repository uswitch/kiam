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
	retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/uswitch/kiam/pkg/aws/sts"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/naming"
	"io/ioutil"
	"time"
)

// Client to interact with KiamServer, exposing k8s.RoleFinder and sts.CredentialsProvider interfaces
type KiamGateway struct {
	conn   *grpc.ClientConn
	client pb.KiamServiceClient
}

const (
	RetryInterval = 10 * time.Millisecond
)

// Creates a client suitable for interacting with a remote server. It can
// be closed cleanly
type GatewayConfig struct {
	Address string
	TLS     *TLSConfig
}

func NewGateway(address string, refresh time.Duration, caFile, certificateFile, keyFile string) (*KiamGateway, error) {
	callOpts := []retry.CallOption{
		retry.WithBackoff(retry.BackoffLinear(RetryInterval)),
	}

	certificate, err := tls.LoadX509KeyPair(certificateFile, keyFile)
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("error appending certs from ca")
	}
	creds := credentials.NewTLS(&tls.Config{
		ServerName:   address,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
	})

	resolver, err := naming.NewDNSResolverWithFreq(refresh)
	if err != nil {
		return nil, err
	}

	balancer := grpc.RoundRobin(resolver)
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(creds), grpc.WithUnaryInterceptor(retry.UnaryClientInterceptor(callOpts...)), grpc.WithBalancer(balancer)}
	conn, err := grpc.Dial(address, dialOpts...)
	if err != nil {
		return nil, err
	}

	client := pb.NewKiamServiceClient(conn)
	return &KiamGateway{conn: conn, client: ClientWithTelemetry(client)}, nil
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
	return nil, nil
}

func (g *KiamGateway) CredentialsForRole(ctx context.Context, role string) (*sts.Credentials, error) {
	credentials, err := g.client.GetRoleCredentials(ctx, &pb.GetRoleCredentialsRequest{&pb.Role{Name: role}})
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
