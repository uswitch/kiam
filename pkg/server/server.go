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

	log "github.com/sirupsen/logrus"
	"github.com/uswitch/k8sc/official"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/k8s"
	"github.com/uswitch/kiam/pkg/prefetch"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
)

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
}

type TLSConfig struct {
	ServerCert string
	ServerKey  string
	CA         string
}

type KiamServer struct {
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

func (k *KiamServer) IsAllowedAssumeRole(ctx context.Context, req *pb.IsAllowedAssumeRoleRequest) (*pb.IsAllowedAssumeRoleResponse, error) {
	decision, err := k.assumePolicy.IsAllowedAssumeRole(ctx, req.Role.Name, req.Ip)
	if err != nil {
		return nil, err
	}

	return &pb.IsAllowedAssumeRoleResponse{
		Decision: &pb.Decision{
			IsAllowed:   decision.IsAllowed(),
			Explanation: decision.Explanation(),
		},
	}, nil
}

func (k *KiamServer) GetHealth(ctx context.Context, _ *pb.GetHealthRequest) (*pb.HealthStatus, error) {
	return &pb.HealthStatus{Message: "ok"}, nil
}

func (k *KiamServer) GetPodRole(ctx context.Context, req *pb.GetPodRoleRequest) (*pb.Role, error) {
	logger := log.WithField("pod.ip", req.Ip)
	pod, err := k.pods.FindPodForIP(req.Ip)
	if err != nil {
		logger.Errorf("error finding pod: %s", err.Error())
		return nil, err
	}

	role := k8s.PodRole(pod)
	ref, err := reference.GetReference(scheme.Scheme, pod)
	if err != nil {
		logger.Errorf("error getting reference for pod %q: %s", pod.Name, err)
		return nil, err
	}

	logger.WithField("pod.iam.role", role).Infof("found role")
	k.eventRecorder.Event(ref, v1.EventTypeNormal, "KiamRoleFound",
		fmt.Sprintf("Role: %q found for pod: %q", role, pod.Name))

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

func (k *KiamServer) GetRoleCredentials(ctx context.Context, req *pb.GetRoleCredentialsRequest) (*pb.Credentials, error) {
	logger := log.WithField("pod.iam.role", req.Role.Name)

	logger.Infof("requesting credentials")
	credentials, err := k.credentialsProvider.CredentialsForRole(ctx, req.Role.Name)
	if err != nil {
		logger.Errorf("error requesting credentials: %s", err.Error())
		return nil, err
	}

	return translateCredentialsToProto(credentials), nil
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

func NewServer(config *Config) (*KiamServer, error) {
	server := &KiamServer{parallelFetchers: config.ParallelFetcherProcesses}

	listener, err := net.Listen("tcp", config.BindAddress)
	if err != nil {
		return nil, err
	}
	server.listener = listener

	client, err := official.NewClient(config.KubeConfig)
	if err != nil {
		log.Fatalf("couldn't create kubernetes client: %s", err.Error())
	}

	server.pods = k8s.NewPodCache(k8s.NewListWatch(client, k8s.ResourcePods), config.PodSyncInterval, config.PrefetchBufferSize)
	server.namespaces = k8s.NewNamespaceCache(k8s.NewListWatch(client, k8s.ResourceNamespaces), time.Minute)
	server.eventRecorder = eventRecorder(client)

	stsGateway := sts.DefaultGateway(config.AssumeRoleArn)
	arnResolver, err := newRoleARNResolver(config)
	if err != nil {
		return nil, err
	}
	credentialsCache := sts.DefaultCache(
		stsGateway, config.SessionName,
		config.SessionDuration,
		config.SessionRefresh,
		arnResolver,
	)
	server.credentialsProvider = credentialsCache
	server.manager = prefetch.NewManager(credentialsCache, server.pods, server.pods)
	server.assumePolicy = Policies(NewRequestingAnnotatedRolePolicy(server.pods, arnResolver), NewNamespacePermittedRoleNamePolicy(server.namespaces, server.pods))

	certificate, err := tls.LoadX509KeyPair(config.TLS.ServerCert, config.TLS.ServerKey)
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	if err != nil {
		return nil, err
	}
	ca, err := ioutil.ReadFile(config.TLS.CA)
	if err != nil {
		return nil, err
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("failed to append CA cert to certPool")
	}
	creds := credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	})

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterKiamServiceServer(grpcServer, ServerWithTelemetry(server))
	server.server = grpcServer

	return server, nil
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
