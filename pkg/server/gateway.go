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
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/statsd"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/naming"
)

// Client is the Server's client interface
type Client interface {
	GetRole(ctx context.Context, ip string) (string, error)
	GetCredentials(ctx context.Context, ip, role string) (*sts.Credentials, error)
	Health(ctx context.Context) (string, error)
}

// KiamGateway is the client to interact with KiamServer
type KiamGateway struct {
	conn   *grpc.ClientConn
	client pb.KiamServiceClient
	statsd bool
}

const (
	RetryInterval = 10 * time.Millisecond
)

// NewGateway constructs a gRPC client to talk to the server
func NewGateway(ctx context.Context, address string, refresh time.Duration, caFile, certificateFile, keyFile string, statsd bool) (*KiamGateway, error) {
	callOpts := []retry.CallOption{
		retry.WithBackoff(retry.BackoffLinear(RetryInterval)),
	}

	certificate, err := tls.LoadX509KeyPair(certificateFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("error loading keypair: %v", err)
	}
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("error reading SSL cert: %v", err)
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("error appending certs from ca")
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("error parsing hostname: %v", err)
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName:   host,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
	})

	resolver, err := naming.NewDNSResolverWithFreq(refresh)
	if err != nil {
		return nil, fmt.Errorf("error creating DNS resolver: %v", err)
	}

	balancer := grpc.RoundRobin(resolver)
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(retry.UnaryClientInterceptor(callOpts...), grpc_prometheus.UnaryClientInterceptor)),
		grpc.WithBalancer(balancer),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	}
	conn, err := grpc.Dial(address, dialOpts...)
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
	return &KiamGateway{conn: conn, client: client, statsd: statsd}, nil
}

// Close disconnects the connection
func (g *KiamGateway) Close() {
	g.conn.Close()
}

// GetRole returns the role for the identified Pod
func (g *KiamGateway) GetRole(ctx context.Context, ip string) (string, error) {
	if g.statsd {
		defer statsd.Client.NewTiming().Send("gateway.rpc.GetRole")
	}
	role, err := g.client.GetPodRole(ctx, &pb.GetPodRoleRequest{Ip: ip})
	if err != nil {
		return "", err
	}
	return role.GetName(), nil
}

// GetCredentials returns the credentials for the identified Pod
func (g *KiamGateway) GetCredentials(ctx context.Context, ip, role string) (*sts.Credentials, error) {
	if g.statsd {
		defer statsd.Client.NewTiming().Send("gateway.rpc.GetCredentials")
	}
	credentials, err := g.client.GetPodCredentials(ctx, &pb.GetPodCredentialsRequest{Ip: ip, Role: role})
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

// Health is used to check the gRPC client connection
func (g *KiamGateway) Health(ctx context.Context) (string, error) {
	if g.statsd {
		defer statsd.Client.NewTiming().Send("gateway.rpc.Health")
	}
	status, err := g.client.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return "", err
	}
	return status.Message, nil
}
