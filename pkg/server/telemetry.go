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
	pb "github.com/uswitch/kiam/proto"
	grpc "google.golang.org/grpc"
	"time"
)

// implements the KiamServiceClient interface with timers
type TelemetryClient struct {
	client pb.KiamServiceClient
}

func ClientWithTelemetry(client pb.KiamServiceClient) pb.KiamServiceClient {
	return &TelemetryClient{client}
}

func (c *TelemetryClient) GetPodRole(ctx context.Context, in *pb.GetPodRoleRequest, opts ...grpc.CallOption) (*pb.Role, error) {
	timer := metrics.GetOrRegisterTimer("client.GetPodRole", metrics.DefaultRegistry)
	startTime := time.Now()
	defer timer.UpdateSince(startTime)

	return c.client.GetPodRole(ctx, in, opts...)
}

func (c *TelemetryClient) GetRoleCredentials(ctx context.Context, in *pb.GetRoleCredentialsRequest, opts ...grpc.CallOption) (*pb.Credentials, error) {
	timer := metrics.GetOrRegisterTimer("client.GetRoleCredentials", metrics.DefaultRegistry)
	startTime := time.Now()
	defer timer.UpdateSince(startTime)

	return c.client.GetRoleCredentials(ctx, in, opts...)
}

func (c *TelemetryClient) GetHealth(ctx context.Context, in *pb.GetHealthRequest, opts ...grpc.CallOption) (*pb.HealthStatus, error) {
	timer := metrics.GetOrRegisterTimer("client.GetHealth", metrics.DefaultRegistry)
	startTime := time.Now()
	defer timer.UpdateSince(startTime)

	return c.client.GetHealth(ctx, in, opts...)
}

func (c *TelemetryClient) IsAllowedAssumeRole(ctx context.Context, in *pb.IsAllowedAssumeRoleRequest, opts ...grpc.CallOption) (*pb.IsAllowedAssumeRoleResponse, error) {
	timer := metrics.GetOrRegisterTimer("client.IsAllowedAssumeRole", metrics.DefaultRegistry)
	startTime := time.Now()
	defer timer.UpdateSince(startTime)

	return c.client.IsAllowedAssumeRole(ctx, in, opts...)
}

// implements the KiamServiceServer interface with timers
type TelemetryServer struct {
	server pb.KiamServiceServer
}

func ServerWithTelemetry(server pb.KiamServiceServer) pb.KiamServiceServer {
	return &TelemetryServer{server}
}

func (c *TelemetryServer) IsAllowedAssumeRole(ctx context.Context, in *pb.IsAllowedAssumeRoleRequest) (*pb.IsAllowedAssumeRoleResponse, error) {
	timer := metrics.GetOrRegisterTimer("server.IsAllowedAssumeRole", metrics.DefaultRegistry)
	startTime := time.Now()
	defer timer.UpdateSince(startTime)

	return c.server.IsAllowedAssumeRole(ctx, in)
}

func (c *TelemetryServer) GetPodRole(ctx context.Context, in *pb.GetPodRoleRequest) (*pb.Role, error) {
	timer := metrics.GetOrRegisterTimer("server.GetPodRole", metrics.DefaultRegistry)
	startTime := time.Now()
	defer timer.UpdateSince(startTime)

	return c.server.GetPodRole(ctx, in)
}

func (c *TelemetryServer) GetRoleCredentials(ctx context.Context, in *pb.GetRoleCredentialsRequest) (*pb.Credentials, error) {
	timer := metrics.GetOrRegisterTimer("server.GetRoleCredentials", metrics.DefaultRegistry)
	startTime := time.Now()
	defer timer.UpdateSince(startTime)

	return c.server.GetRoleCredentials(ctx, in)
}

func (c *TelemetryServer) GetHealth(ctx context.Context, in *pb.GetHealthRequest) (*pb.HealthStatus, error) {
	timer := metrics.GetOrRegisterTimer("server.GetHealth", metrics.DefaultRegistry)
	startTime := time.Now()
	defer timer.UpdateSince(startTime)

	return c.server.GetHealth(ctx, in)
}
