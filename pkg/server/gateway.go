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
	"k8s.io/client-go/pkg/api/v1"
)

// Handles interaction with KiamServer, exposing k8s.PodFinder and sts.CredentialsProvider interfaces
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

func (g *KiamGateway) FindPodForIP(ip string) (*v1.Pod, error) {
	log.Printf("finding pod for ip: %s", ip)
	g.client.GetPodRole(context.Background(), &pb.GetPodRoleRequest{Ip: ip})
	return nil, nil
}

func (g *KiamGateway) CredentialsForRole(role string) (*sts.Credentials, error) {
	log.Printf("credentials for role: %s", role)
	g.client.GetRoleCredentials(context.Background(), &pb.GetRoleCredentialsRequest{&pb.Role{Name: role}})
	return nil, nil
}
