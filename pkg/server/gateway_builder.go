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
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/security/advancedtls"
	"net"
	"time"
)

const (
	kDefaultRetryInterval = 10 * time.Millisecond
	kMaxRetries           = 0
)

// KiamGatewayBuilder helps to construct the KiamGateway for interacting with the KiamServer
type KiamGatewayBuilder struct {
	address         string
	transportCreds  credentials.TransportCredentials
	keepaliveParams keepalive.ClientParameters
	tlsConfig       *dynamicTLSConfig
	dialOptions     []grpc.DialOption
	retryInterval   time.Duration
	maxRetries      uint
	dnsResolver     string
}

func NewKiamGatewayBuilder() *KiamGatewayBuilder {
	return &KiamGatewayBuilder{retryInterval: kDefaultRetryInterval, maxRetries: kMaxRetries}
}

func (b *KiamGatewayBuilder) WithAddress(address string) *KiamGatewayBuilder {
	b.address = address
	return b
}

func (b *KiamGatewayBuilder) WithDNSResolver(resolver string) *KiamGatewayBuilder {
	b.dnsResolver = resolver
	return b
}

func (b *KiamGatewayBuilder) WithRetryInterval(dur time.Duration) *KiamGatewayBuilder {
	b.retryInterval = dur
	return b
}

func (b *KiamGatewayBuilder) WithMaxRetries(n uint) *KiamGatewayBuilder {
	b.maxRetries = n
	return b
}

// WithTLS configures the gRPC client with dynamic TLS.
func (b *KiamGatewayBuilder) WithTLS(cert, key, ca string) (*KiamGatewayBuilder, error) {
	notifyFn := clientTLSMetrics.notifyFunc(x509.ExtKeyUsageClientAuth)
	tlsConfig, err := newDynamicTLSConfig(cert, key, ca, notifyFn)
	if err != nil {
		return nil, fmt.Errorf("error reading tls certificates: %v", err)
	}
	b.tlsConfig = tlsConfig
	defer func() {
		if err != nil {
			tlsConfig.Close()
		}
	}()

	hostName, _, err := net.SplitHostPort(b.address)
	if err != nil {
		return nil, fmt.Errorf("error parsing hostname: %v", err)
	}

	creds, err := advancedtls.NewClientCreds(&advancedtls.ClientOptions{
		GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return tlsConfig.LoadCert(), nil
		},
		RootCertificateOptions: advancedtls.RootCertificateOptions{
			GetRootCAs: func(_ *advancedtls.GetRootCAsParams) (*advancedtls.GetRootCAsResults, error) {
				return &advancedtls.GetRootCAsResults{TrustCerts: tlsConfig.LoadCACerts()}, nil
			},
		},
		ServerNameOverride: hostName,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating grpc credentials: %v", err)
	}

	b.transportCreds = creds
	return b, nil
}

func (b *KiamGatewayBuilder) WithKeepAlive(parameters keepalive.ClientParameters) *KiamGatewayBuilder {
	b.keepaliveParams = parameters
	return b
}

func (b *KiamGatewayBuilder) Build(ctx context.Context) (*KiamGateway, error) {
	dialOpts := []grpc.DialOption{
		grpc.WithKeepaliveParams(b.keepaliveParams),
		grpc.WithTransportCredentials(b.transportCreds),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			grpc_prometheus.UnaryClientInterceptor,
			retry.UnaryClientInterceptor(
				retry.WithMax(b.maxRetries),
				retry.WithBackoff(retry.BackoffLinear(b.retryInterval)),
			),
		)),
		grpc.WithBalancerName(roundrobin.Name),
		grpc.WithDisableServiceConfig(),
		grpc.WithBlock(),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	}
	if b.dialOptions != nil {
		dialOpts = append(dialOpts, b.dialOptions...)
	}

	dialTarget := fmt.Sprintf("dns://%s/%s", b.dnsResolver, b.address)
	conn, err := grpc.DialContext(ctx, dialTarget, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("error dialing grpc server: %v", err)
	}
	gw := &KiamGateway{
		conn:      conn,
		client:    pb.NewKiamServiceClient(conn),
		tlsConfig: b.tlsConfig,
	}
	return gw, nil
}

func (b *KiamGatewayBuilder) WithDialOption(opts ...grpc.DialOption) *KiamGatewayBuilder {
	b.dialOptions = opts
	return b
}
