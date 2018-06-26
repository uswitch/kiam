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
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pubnub/go-metrics-statsd"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/prometheus"
	serv "github.com/uswitch/kiam/pkg/server"
	"gopkg.in/alecthomas/kingpin.v2"
)

type options struct {
	jsonLog          bool
	logLevel         string
	statsd           string
	statsdInterval   time.Duration
	prometheusListen string
	prometheusSync   time.Duration
}

func (o *options) bind(parser *kingpin.Application) {
	parser.Flag("json-log", "Output log in JSON").BoolVar(&o.jsonLog)
	parser.Flag("level", "Log level: debug, info, warn, error.").Default("info").EnumVar(&o.logLevel, "debug", "info", "warn", "error")

	parser.Flag("statsd", "UDP address to publish StatsD metrics. e.g. 127.0.0.1:8125").Default("").StringVar(&o.statsd)
	parser.Flag("statsd-interval", "Interval to publish to StatsD").Default("10s").DurationVar(&o.statsdInterval)

	parser.Flag("prometheus-listen-addr", "Prometheus HTTP listen address. e.g. localhost:9620").StringVar(&o.prometheusListen)
	parser.Flag("prometheus-sync-interval", "How frequently to update Prometheus metrics").Default("5s").DurationVar(&o.prometheusSync)
}

type serverOptions struct {
	*serv.Config
}

func (o *serverOptions) bind(parser *kingpin.Application) {
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

	parser.Flag("cert", "Server certificate path").Required().ExistingFileVar(&o.TLS.ServerCert)
	parser.Flag("key", "Server private key path").Required().ExistingFileVar(&o.TLS.ServerKey)
	parser.Flag("ca", "CA path").Required().ExistingFileVar(&o.TLS.CA)
}

func main() {
	serverConfig := &serv.Config{TLS: &serv.TLSConfig{}}

	var opts options
	opts.bind(kingpin.CommandLine)

	serverOpts := serverOptions{serverConfig}
	serverOpts.bind(kingpin.CommandLine)

	kingpin.Parse()

	if !serverConfig.AutoDetectBaseARN && serverConfig.RoleBaseARN == "" {
		log.Fatal("role-base-arn not specified and not auto-detected. please specify or use --role-base-arn-autodetect")
	}

	if serverConfig.SessionDuration < sts.AWSMinSessionDuration {
		log.Fatal("session-duration should be at least 15 minutes")
	}

	if opts.jsonLog {
		log.SetFormatter(&log.JSONFormatter{})
	}

	switch opts.logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if opts.statsd != "" {
		addr, err := net.ResolveUDPAddr("udp", opts.statsd)
		if err != nil {
			log.Fatal("error parsing statsd address:", err.Error())
		}
		go statsd.StatsD(metrics.DefaultRegistry, opts.statsdInterval, "kiam.server", addr)
	}

	if opts.prometheusListen != "" {
		metrics := prometheus.NewServer("server", opts.prometheusListen, opts.prometheusSync)
		metrics.Listen(ctx)
	}

	log.Infof("starting server")
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	signal.Notify(stopChan, syscall.SIGTERM)

	server, err := serv.NewServer(serverConfig)
	if err != nil {
		log.Fatal("error creating listener: ", err.Error())
	}

	go func() {
		<-stopChan
		log.Infof("stopping server")
		server.Stop()
		cancel()
	}()

	log.Infof("will serve on %s", serverConfig.BindAddress)

	server.Serve(ctx)

	log.Infoln("stopped")
}
