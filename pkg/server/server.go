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
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/k8sc/official"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/k8s"
	"github.com/uswitch/kiam/pkg/prefetch"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
	"net"
	"time"
)

type Config struct {
	BindAddress     string
	KubeConfig      string
	PodSyncInterval time.Duration
	SessionName     string
	RoleBaseARN     string
}

type KiamServer struct {
	listener            net.Listener
	server              *grpc.Server
	cache               *k8s.PodCache
	manager             *prefetch.CredentialManager
	credentialsProvider sts.CredentialsProvider
}

func (k *KiamServer) GetPodRole(ctx context.Context, req *pb.GetPodRoleRequest) (*pb.Role, error) {
	roleTimer := metrics.GetOrRegisterTimer("GetPodRole", metrics.DefaultRegistry)
	startTime := time.Now()
	defer roleTimer.UpdateSince(startTime)

	logger := log.WithField("pod.ip", req.Ip)
	role, err := k.cache.FindRoleFromIP(ctx, req.Ip)
	if err != nil {
		logger.Errorf("error finding role: %s", err.Error())
		return nil, err
	}

	logger.WithField("pod.iam.role", role).Infof("successfully found role")

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

func NewServer(config *Config) (*KiamServer, error) {
	server := &KiamServer{}

	listener, err := net.Listen("tcp", config.BindAddress)
	if err != nil {
		return nil, err
	}
	server.listener = listener

	client, err := official.NewClient(config.KubeConfig)
	if err != nil {
		log.Fatalf("couldn't create kubernetes client: %s", err.Error())
	}
	server.cache = k8s.NewPodCache(k8s.KubernetesSource(client), config.PodSyncInterval)

	credentials := sts.DefaultCache(config.RoleBaseARN, config.SessionName)
	server.credentialsProvider = credentials
	server.manager = prefetch.NewManager(credentials, server.cache, server.cache)

	var serverOpts []grpc.ServerOption
	grpcServer := grpc.NewServer(serverOpts...)
	pb.RegisterKiamServiceServer(grpcServer, server)
	server.server = grpcServer

	return server, nil
}

func (s *KiamServer) Serve(ctx context.Context) {
	s.cache.Run(ctx)
	go s.manager.Run(ctx)
	s.server.Serve(s.listener)
}

func (s *KiamServer) Stop() {
	s.server.GracefulStop()
}
