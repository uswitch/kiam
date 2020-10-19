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

	"github.com/aws/aws-sdk-go/aws/awserr"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/k8sc/official"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/k8s"
	"github.com/uswitch/kiam/pkg/prefetch"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/security/advancedtls"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

// Config controls the setup of the gRPC server
type Config struct {
	BindAddress              string
	KubeConfig               string
	PodSyncInterval          time.Duration
	SessionName              string
	SessionDuration          time.Duration
	SessionRefresh           time.Duration
	RoleBaseARN              string
	AutoDetectBaseARN        bool
	TLS                      TLSConfig
	ParallelFetcherProcesses int
	PrefetchBufferSize       int
	AssumeRoleArn            string
	Region                   string
}

// TLSConfig controls TLS
type TLSConfig struct {
	ServerCert string
	ServerKey  string
	CA         string
}

// KiamServer is the gRPC server. Construct with NewServer.
type KiamServer struct {
	tlsConfig           *dynamicTLSConfig
	listener            net.Listener
	server              *grpc.Server
	pods                *k8s.PodCache
	namespaces          *k8s.NamespaceCache
	eventRecorder       record.EventRecorder
	manager             *prefetch.CredentialManager
	credentialsProvider sts.CredentialsProvider
	assumePolicy        AssumeRolePolicy
	parallelFetchers    int
}

func simplifyAWSErrorMessage(err error) string {
	e, ok := err.(awserr.Error)
	if !ok {
		return err.Error()
	}

	return fmt.Sprintf("%s: %s", e.Code(), e.Message())
}

// GetPodCredentials returns credentials for the Pod, according to the role it's
// annotated with. It will additionally check policy before returning credentials.
func (k *KiamServer) GetPodCredentials(ctx context.Context, req *pb.GetPodCredentialsRequest) (*pb.Credentials, error) {
	pod, err := k.pods.GetPodByIP(req.Ip)
	if err != nil {
		if err == k8s.ErrPodNotFound {
			return nil, ErrPodNotFound
		}

		return nil, err
	}
	logger := log.WithFields(k8s.PodFields(pod)).WithField("pod.iam.requestedRole", req.Role)

	decision, err := k.assumePolicy.IsAllowedAssumeRole(ctx, req.Role, req.Ip)
	if err != nil {
		logger.Errorf("error checking policy: %s", err.Error())
		return nil, err
	}

	if !decision.IsAllowed() {
		logger.WithField("policy.explanation", decision.Explanation()).Errorf("pod denied by policy")
		k.recordEvent(pod, v1.EventTypeWarning, "KiamRoleForbidden", fmt.Sprintf("failed assuming role %q: %s", req.Role, decision.Explanation()))
		return nil, ErrPolicyForbidden
	}

	creds, err := k.credentialsProvider.CredentialsForRole(ctx, &sts.CredentialsIdentity{Role: req.Role})
	if err != nil {
		logger.Errorf("error retrieving credentials: %s", err.Error())
		k.recordEvent(pod, v1.EventTypeWarning, "KiamCredentialError", fmt.Sprintf("failed retrieving credentials: %s", simplifyAWSErrorMessage(err)))
		return nil, err
	}

	return translateCredentialsToProto(creds), nil
}

// GetHealth returns ok to allow a command to ensure the sever is operating well
func (k *KiamServer) GetHealth(ctx context.Context, _ *pb.GetHealthRequest) (*pb.HealthStatus, error) {
	return &pb.HealthStatus{Message: "ok"}, nil
}

// GetPodRole determines which role a Pod is annotated with
func (k *KiamServer) GetPodRole(ctx context.Context, req *pb.GetPodRoleRequest) (*pb.Role, error) {
	logger := log.WithField("pod.ip", req.Ip)
	pod, err := k.pods.GetPodByIP(req.Ip)
	if err != nil {
		logger.Errorf("error finding pod: %s", err.Error())
		return nil, err
	}

	role := k8s.PodRole(pod)

	logger.WithField("pod.iam.role", role).Infof("found role")
	return &pb.Role{Name: role}, nil
}

func translateCredentialsToProto(credentials *sts.Credentials) *pb.Credentials {
	return &pb.Credentials{
		Code:            credentials.Code,
		Type:            credentials.Type,
		AccessKeyId:     credentials.AccessKeyId,
		SecretAccessKey: credentials.SecretAccessKey,
		Token:           credentials.Token,
		Expiration:      credentials.Expiration,
		LastUpdated:     credentials.LastUpdated,
	}
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

// NewServer constructs a new server.
func NewServer(config *Config) (_ *KiamServer, err error) {
	arnResolver, err := newRoleARNResolver(config)
	if err != nil {
		return nil, err
	}
	stsGateway, err := sts.DefaultGateway(arnResolver.Resolve(config.AssumeRoleArn), config.Region)
	if err != nil {
		return nil, err
	}
	credentialsCache := sts.DefaultCache(
		stsGateway,
		config.SessionName,
		config.SessionDuration,
		config.SessionRefresh,
		arnResolver,
	)

	client, err := official.NewClient(config.KubeConfig)
	if err != nil {
		return nil, err
	}
	podCache := k8s.NewPodCache(k8s.NewListWatch(client, k8s.ResourcePods), config.PodSyncInterval, config.PrefetchBufferSize)
	namespaceCache := k8s.NewNamespaceCache(k8s.NewListWatch(client, k8s.ResourceNamespaces), time.Minute)

	notifyFn := serverTLSMetrics.notifyFunc(x509.ExtKeyUsageServerAuth)
	tlsConfig, err := newDynamicTLSConfig(config.TLS.ServerCert, config.TLS.ServerKey, config.TLS.CA, notifyFn)
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
	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)

	listener, err := net.Listen("tcp", config.BindAddress)
	if err != nil {
		return nil, err
	}
	srv := &KiamServer{
		tlsConfig:           tlsConfig,
		listener:            listener,
		server:              grpcServer,
		pods:                podCache,
		namespaces:          namespaceCache,
		eventRecorder:       eventRecorder(client),
		manager:             prefetch.NewManager(credentialsCache, podCache),
		credentialsProvider: credentialsCache,
		assumePolicy: Policies(
			NewRequestingAnnotatedRolePolicy(podCache, arnResolver),
			NewNamespacePermittedRoleNamePolicy(namespaceCache, podCache),
		),
		parallelFetchers: config.ParallelFetcherProcesses,
	}
	pb.RegisterKiamServiceServer(grpcServer, srv)
	return srv, nil
}

// Serve starts the server, starting all components and listening for gRPC
func (k *KiamServer) Serve(ctx context.Context) {
	k.manager.Run(ctx, k.parallelFetchers)
	err := k.pods.Run(ctx)
	if err != nil {
		log.Fatalf("error starting pod cache: %s", err)
	}
	err = k.namespaces.Run(ctx)
	if err != nil {
		log.Fatalf("error starting namespace cache: %s", err)
	}
	k.server.Serve(k.listener)
}

// Stop performs a graceful shutdown of the gRPC server
func (k *KiamServer) Stop() {
	k.server.GracefulStop()
	k.listener.Close()
	k.tlsConfig.Close()
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

func (k *KiamServer) recordEvent(object runtime.Object, eventtype, reason, message string) {
	if k.eventRecorder == nil {
		return
	}
	k.eventRecorder.Event(object, eventtype, reason, message)
}
