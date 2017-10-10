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
	"github.com/uswitch/kiam/pkg/aws/sts"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
)

// Client to interact with KiamServer, exposing k8s.RoleFinder and sts.CredentialsProvider interfaces
type KiamGateway struct {
	conn   *grpc.ClientConn
	client pb.KiamServiceClient
}

// Creates a client suitable for interacting with a remote server. It can
// be closed cleanly
func NewGateway(address string) (*KiamGateway, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, err
	}

	client := pb.NewKiamServiceClient(conn)

	return &KiamGateway{conn: conn, client: client}, nil
}

func (g *KiamGateway) Close() {
	g.conn.Close()
}

func (g *KiamGateway) FindRoleFromIP(ctx context.Context, ip string) (string, error) {
	log.Printf("finding pod for ip: %s", ip)
	role, err := g.client.GetPodRole(ctx, &pb.GetPodRoleRequest{Ip: ip})
	if err != nil {
		return "", err
	}
	return role.Name, nil
}

func (g *KiamGateway) CredentialsForRole(ctx context.Context, role string) (*sts.Credentials, error) {
	log.Printf("credentials for role: %s", role)
	g.client.GetRoleCredentials(ctx, &pb.GetRoleCredentialsRequest{&pb.Role{Name: role}})
	return nil, nil
}
