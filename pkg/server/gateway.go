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
	"github.com/uswitch/kiam/pkg/aws/sts"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
	status "google.golang.org/grpc/status"
	"time"
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
	retryInterval time.Time
}

// Close disconnects the connection
func (g *KiamGateway) Close() {
	g.conn.Close()
	g.tlsConfig.Close()
}

// GetRole returns the role for the identified Pod
func (g *KiamGateway) GetRole(ctx context.Context, ip string) (string, error) {
	role, err := g.client.GetPodRole(ctx, &pb.GetPodRoleRequest{Ip: ip})
	if err != nil {
		return "", err
	}
	return role.GetName(), nil
}

// GetCredentials returns the credentials for the identified Pod
func (g *KiamGateway) GetCredentials(ctx context.Context, ip, role string) (*sts.Credentials, error) {
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
	status, err := g.client.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return "", err
	}
	return status.Message, nil
}
