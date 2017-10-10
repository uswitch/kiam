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
	log "github.com/sirupsen/logrus"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
	"net"
)

type KiamServer struct {
	listener net.Listener
	server   *grpc.Server
}

func (k *KiamServer) GetPodRole(ctx context.Context, req *pb.GetPodRoleRequest) (*pb.Role, error) {
	log.Infof("GetPodRole: %+v", req)
	return nil, nil
}

func (k *KiamServer) GetRoleCredentials(ctx context.Context, req *pb.GetRoleCredentialsRequest) (*pb.Credentials, error) {
	log.Infof("GetRoleCredentials: %+v", req)
	return nil, nil
}

func NewServer(bind string) (*KiamServer, error) {
	server := &KiamServer{}

	listener, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, err
	}
	server.listener = listener

	var serverOpts []grpc.ServerOption
	grpcServer := grpc.NewServer(serverOpts...)
	pb.RegisterKiamServiceServer(grpcServer, server)

	server.server = grpcServer

	return server, nil
}

func (s *KiamServer) Serve() {
	s.server.Serve(s.listener)
}

func (s *KiamServer) Stop() {
	s.server.GracefulStop()
}
