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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/k8sc/official"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/k8s"
	"github.com/uswitch/kiam/pkg/prefetch"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/security/advancedtls"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"net"
	"time"
)

// KiamServerBuilder helps construct the KiamServer
type KiamServerBuilder struct {
	config               *Config
	stsGateway           sts.STSGateway
	podCache             *k8s.PodCache
	namespaceCache       *k8s.NamespaceCache
	eventRecorder        record.EventRecorder
	transportCredentials credentials.TransportCredentials
	tlsConfig            *dynamicTLSConfig
	grpcServer           *grpc.Server
}

func NewKiamServerBuilder(c *Config) *KiamServerBuilder {
	return &KiamServerBuilder{config: c}
}

// WithAWSSTSGateway creates the server with an STS Gateway that interacts
// with AWS APIs. Use WithSTSGateway to provide a different implementation.
func (b *KiamServerBuilder) WithAWSSTSGateway() (*KiamServerBuilder, error) {
	cfg, err := sts.NewServerConfigBuilder().WithRegion(b.config.Region)
	if err != nil {
		return nil, err
	}
	cfg.WithCredentialsFromAssumedRole(sts.NewSTSCredentialsProvider(), b.config.AssumeRoleArn)
	stsGateway, err := sts.DefaultGateway(cfg.Config())
	if err != nil {
		return nil, err
	}

	b.WithSTSGateway(stsGateway)

	return b, nil
}

// WithSTSGateway specifies the STS Gateway to use when issuing credentials
func (b *KiamServerBuilder) WithSTSGateway(gateway sts.STSGateway) {
	b.stsGateway = gateway
}

func newRoleARNResolver(config *Config) (sts.ARNResolver, error) {
	if config.AutoDetectBaseARN {
		log.Infof("detecting arn prefix")
		prefix, err := sts.DetectARNPrefix()
		if err != nil {
			return nil, fmt.Errorf("error detecting arn prefix: %s", err)
		}
		log.Infof("using detected prefix: %s", prefix)
		return sts.DefaultResolver(prefix), nil
	}

	return sts.DefaultResolver(config.RoleBaseARN), nil
}

func eventRecorder(kubeClient *kubernetes.Clientset) record.EventRecorder {
	source := v1.EventSource{Component: "kiam.server"}
	sink := &typedcorev1.EventSinkImpl{
		Interface: kubeClient.CoreV1().Events(""),
	}
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(sink)

	return broadcaster.NewRecorder(scheme.Scheme, source)
}

// WithKubernetesClient configures the server to use the Kubernetes client to watch
// Pods and Namespaces.
func (b *KiamServerBuilder) WithKubernetesClient() (*KiamServerBuilder, error) {
	client, err := official.NewClient(b.config.KubeConfig)
	if err != nil {
		return nil, err
	}

	podCache := k8s.NewPodCache(k8s.NewListWatch(client, k8s.ResourcePods), b.config.PodSyncInterval, b.config.PrefetchBufferSize)
	nsCache := k8s.NewNamespaceCache(k8s.NewListWatch(client, k8s.ResourceNamespaces), time.Minute)

	b.WithCaches(podCache, nsCache)

	b.eventRecorder = eventRecorder(client)

	return b, nil
}

// WithCaches configures the Pod and Namespace caches used for watching for Kubernetes objects.
func (b *KiamServerBuilder) WithCaches(podCache *k8s.PodCache, nsCache *k8s.NamespaceCache) *KiamServerBuilder {
	b.podCache = podCache
	b.namespaceCache = nsCache

	return b
}

// WithTLS configures the Kiam server to use mutual TLS. Should always be used in production.
func (b *KiamServerBuilder) WithTLS() (*KiamServerBuilder, error) {
	notifyFn := serverTLSMetrics.notifyFunc(x509.ExtKeyUsageServerAuth)
	tlsConfig, err := newDynamicTLSConfig(b.config.TLS.ServerCert, b.config.TLS.ServerKey, b.config.TLS.CA, notifyFn)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			tlsConfig.Close()
		}
	}()

	creds, err := advancedtls.NewServerCreds(&advancedtls.ServerOptions{
		GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return tlsConfig.LoadCert(), nil
		},
		RootCertificateOptions: advancedtls.RootCertificateOptions{
			GetRootCAs: func(_ *advancedtls.GetRootCAsParams) (*advancedtls.GetRootCAsResults, error) {
				return &advancedtls.GetRootCAsResults{TrustCerts: tlsConfig.LoadCACerts()}, nil
			},
		},
		RequireClientCert: true,
	})
	if err != nil {
		return nil, err
	}

	b.transportCredentials = creds
	b.tlsConfig = tlsConfig

	b.grpcServer = grpc.NewServer(
		grpc.Creds(b.transportCredentials),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)

	return b, nil
}

// WithGRPCServer controls the gRPC Server that the KiamServer will use to listen.
func (b *KiamServerBuilder) WithGRPCServer(server *grpc.Server) *KiamServerBuilder {
	b.grpcServer = server
	return b
}

func (b *KiamServerBuilder) Build() (*KiamServer, error) {
	arnResolver, err := newRoleARNResolver(b.config)
	if err != nil {
		return nil, err
	}

	credentialsCache := sts.DefaultCache(
		b.stsGateway,
		b.config.SessionName,
		b.config.SessionDuration,
		b.config.SessionRefresh,
		arnResolver,
	)

	listener, err := net.Listen("tcp", b.config.BindAddress)
	if err != nil {
		return nil, err
	}

	srv := &KiamServer{
		tlsConfig:           b.tlsConfig,
		listener:            listener,
		server:              b.grpcServer,
		pods:                b.podCache,
		namespaces:          b.namespaceCache,
		eventRecorder:       b.eventRecorder,
		manager:             prefetch.NewManager(credentialsCache, b.podCache),
		credentialsProvider: credentialsCache,
		assumePolicy: Policies(
			NewRequestingAnnotatedRolePolicy(b.podCache, arnResolver),
			NewNamespacePermittedRoleNamePolicy(b.namespaceCache, b.podCache),
		),
		parallelFetchers: b.config.ParallelFetcherProcesses,
	}
	pb.RegisterKiamServiceServer(b.grpcServer, srv)
	return srv, nil
}
