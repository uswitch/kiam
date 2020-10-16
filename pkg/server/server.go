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
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/k8s"
	"github.com/uswitch/kiam/pkg/prefetch"
	pb "github.com/uswitch/kiam/proto"
	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

// KiamServer is the gRPC server. Construct with KiamServerBuilder.
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
	arnResolver         sts.ARNResolver
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

	decision, err := k.assumePolicy.IsAllowedAssumeRole(ctx, req.Role, pod)
	if err != nil {
		logger.Errorf("error checking policy: %s", err.Error())
		return nil, err
	}

	if !decision.IsAllowed() {
		logger.WithField("policy.explanation", decision.Explanation()).Errorf("pod denied by policy")
		k.recordEvent(pod, v1.EventTypeWarning, "KiamRoleForbidden", fmt.Sprintf("failed assuming role %q: %s", req.Role, decision.Explanation()))
		return nil, ErrPolicyForbidden
	}

	resolvedRole, err := k.arnResolver.Resolve(req.Role)
	if err != nil {
		return nil, err
	}

	identity := &sts.RoleIdentity{
		Role:        *resolvedRole,
		SessionName: k8s.PodSessionName(pod),
		ExternalID:  k8s.PodExternalID(pod),
	}

	creds, err := k.credentialsProvider.CredentialsForRole(ctx, identity)
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
	log.Infof("listening")
	k.server.Serve(k.listener)
}

// Stop performs a graceful shutdown of the gRPC server
func (k *KiamServer) Stop() {
	k.server.GracefulStop()
	k.listener.Close()
	if k.tlsConfig != nil {
		k.tlsConfig.Close()
	}
}

func (k *KiamServer) recordEvent(object runtime.Object, eventtype, reason, message string) {
	if k.eventRecorder == nil {
		return
	}
	k.eventRecorder.Event(object, eventtype, reason, message)
}
