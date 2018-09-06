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
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/aws/sts"
	serv "github.com/uswitch/kiam/pkg/server"
)

type serverCommand struct {
	logOptions
	telemetryOptions
	tlsOptions

	serv.Config
}

func (cmd *serverCommand) Bind(parser parser) {
	cmd.logOptions.bind(parser)
	cmd.telemetryOptions.bind(parser)
	cmd.tlsOptions.bind(parser)

	serverOpts := serverOptions{&cmd.Config}
	serverOpts.bind(parser)
}

type serverOptions struct {
	*serv.Config
}

func (o *serverOptions) bind(parser parser) {
	parser.Flag("fetchers", "Number of parallel fetcher go routines").Default("8").IntVar(&o.ParallelFetcherProcesses)
	parser.Flag("prefetch-buffer-size", "How many Pod events to hold in memory between the Pod watcher and Prefetch manager.").Default("1000").IntVar(&o.PrefetchBufferSize)
	parser.Flag("bind", "gRPC bind address").Default("localhost:9610").StringVar(&o.BindAddress)
	parser.Flag("kubeconfig", "Path to .kube/config (or empty for in-cluster)").Default("").StringVar(&o.KubeConfig)
	parser.Flag("sync", "Pod cache sync interval").Default("1m").DurationVar(&o.PodSyncInterval)
	parser.Flag("role-base-arn", "Base ARN for roles. e.g. arn:aws:iam::123456789:role/").StringVar(&o.RoleBaseARN)
	parser.Flag("role-base-arn-autodetect", "Use EC2 metadata service to detect ARN prefix.").BoolVar(&o.AutoDetectBaseARN)
	parser.Flag("session", "Session name used when creating STS Tokens.").Default("kiam").StringVar(&o.SessionName)
	parser.Flag("session-duration", "Requested session duration for STS Tokens.").Default("15m").DurationVar(&o.SessionDuration)
	parser.Flag("session-refresh", "How soon STS Tokens should be refreshed before their expiration.").Default("5m").DurationVar(&o.SessionRefresh)
	parser.Flag("assume-role-arn", "IAM Role to assume before processing requests").Default("").StringVar(&o.AssumeRoleArn)

	if o.RoleAliases == nil {
		o.RoleAliases = map[string]string{}
	}
	parser.Flag("role-alias", "Mapping of a role alias to the full role name. e.g., external-dns=stack-12345-external-dns-123456. Can be specified multiple times.").StringMapVar(&o.RoleAliases)
}

func (opts *serverCommand) Run() {
	opts.configureLogger()

	if !opts.AutoDetectBaseARN && opts.RoleBaseARN == "" {
		log.Fatal("role-base-arn not specified and not auto-detected. please specify or use --role-base-arn-autodetect")
	}

	if opts.SessionDuration < sts.AWSMinSessionDuration {
		log.Fatal("session-duration should be at least 15 minutes")
	}

	ctx, cancel := context.WithCancel(context.Background())

	opts.telemetryOptions.start(ctx, "server")

	log.Infof("starting server")
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	signal.Notify(stopChan, syscall.SIGTERM)

	opts.Config.TLS = serv.TLSConfig{ServerCert: opts.certificatePath, ServerKey: opts.keyPath, CA: opts.caPath}
	server, err := serv.NewServer(&opts.Config)
	if err != nil {
		log.Fatal("error creating listener: ", err.Error())
	}

	go func() {
		<-stopChan
		log.Infof("stopping server")
		server.Stop()
		cancel()
	}()

	log.Infof("will serve on %s", opts.BindAddress)

	server.Serve(ctx)

	log.Infoln("stopped")
}
