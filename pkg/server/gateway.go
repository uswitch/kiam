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
	"net"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/statsd"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/security/advancedtls"

	status "google.golang.org/grpc/status"
)

// Client is the Server's client interface
type Client interface {
	GetRole(ctx context.Context, ip string) (string, error)
	GetCredentials(ctx context.Context, ip, role string) (*sts.Credentials, error)
	Health(ctx context.Context) (string, error)
}

// KiamGateway is the client to interact with KiamServer
type KiamGateway struct {
	conn      *grpc.ClientConn
	client    pb.KiamServiceClient
	tlsConfig *dynamicTLSConfig
}

const (
	RetryInterval = 10 * time.Millisecond
)

// NewGateway constructs a gRPC client to talk to the server
func NewGateway(ctx context.Context, address string, caFile, certificateFile, keyFile string, keepaliveParams keepalive.ClientParameters) (_ *KiamGateway, err error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("error parsing hostname: %v", err)
	}

	notifyFn := clientTLSMetrics.notifyFunc(x509.ExtKeyUsageClientAuth)
	tlsConfig, err := newDynamicTLSConfig(certificateFile, keyFile, caFile, notifyFn)
	if err != nil {
		return nil, fmt.Errorf("error reading tls certificates: %v", err)
	}
	defer func() {
		if err != nil {
			tlsConfig.Close()
		}
	}()
	creds, err := advancedtls.NewClientCreds(&advancedtls.ClientOptions{
		GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return tlsConfig.LoadCert(), nil
		},
		RootCertificateOptions: advancedtls.RootCertificateOptions{
			GetRootCAs: func(_ *advancedtls.GetRootCAsParams) (*advancedtls.GetRootCAsResults, error) {
				return &advancedtls.GetRootCAsResults{TrustCerts: tlsConfig.LoadCACerts()}, nil
			},
		},
		ServerNameOverride: host,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating grpc credentials: %v", err)
	}

	conn, err := grpc.DialContext(ctx, "dns:///"+address,
		grpc.WithKeepaliveParams(keepaliveParams),
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			retry.UnaryClientInterceptor(
				retry.WithBackoff(retry.BackoffLinear(RetryInterval)),
			),
			grpc_prometheus.UnaryClientInterceptor,
		)),
		grpc.WithBalancerName(roundrobin.Name),
		grpc.WithDisableServiceConfig(),
		grpc.WithBlock(),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	)
	if err != nil {
		return nil, fmt.Errorf("error dialing grpc server: %v", err)
	}
	gw := &KiamGateway{
		conn:      conn,
		client:    pb.NewKiamServiceClient(conn),
		tlsConfig: tlsConfig,
	}
	return gw, nil
}

// Close disconnects the connection
func (g *KiamGateway) Close() {
	g.conn.Close()
	g.tlsConfig.Close()
}

// GetRole returns the role for the identified Pod
func (g *KiamGateway) GetRole(ctx context.Context, ip string) (string, error) {
	if statsd.Enabled {
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
	if statsd.Enabled {
		defer statsd.Client.NewTiming().Send("gateway.rpc.GetCredentials")
	}
	credentials, err := g.client.GetPodCredentials(ctx, &pb.GetPodCredentialsRequest{Ip: ip, Role: role})
	if err != nil {
		if grpcStatus, ok := status.FromError(err); ok {
			switch grpcStatus.Message() {
			case ErrPolicyForbidden.Error():
				return nil, ErrPolicyForbidden
			case ErrPodNotFound.Error():
				return nil, ErrPodNotFound
			}
		}

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
	if statsd.Enabled {
		defer statsd.Client.NewTiming().Send("gateway.rpc.Health")
	}
	status, err := g.client.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return "", err
	}
	return status.Message, nil
}
