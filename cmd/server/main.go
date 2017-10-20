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
	"github.com/pubnub/go-metrics-statsd"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	serv "github.com/uswitch/kiam/pkg/server"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	serverConfig := &serv.Config{TLS: &serv.TLSConfig{}}
	var flags struct {
		jsonLog        bool
		logLevel       string
		statsd         string
		statsdInterval time.Duration
	}

	kingpin.Flag("json-log", "Output log in JSON").BoolVar(&flags.jsonLog)
	kingpin.Flag("level", "Log level: debug, info, warn, error.").Default("info").EnumVar(&flags.logLevel, "debug", "info", "warn", "error")

	kingpin.Flag("statsd", "UDP address to publish StatsD metrics. e.g. 127.0.0.1:8125").Default("").StringVar(&flags.statsd)
	kingpin.Flag("statsd-interval", "Interval to publish to StatsD").Default("10s").DurationVar(&flags.statsdInterval)

	kingpin.Flag("fetchers", "Number of parallel fetcher go routines").Default("8").IntVar(&serverConfig.ParallelFetcherProcesses)
	kingpin.Flag("prefetch-buffer-size", "How many Pod events to hold in memory between the Pod watcher and Prefetch manager.").Default("1000").IntVar(&serverConfig.PrefetchBufferSize)
	kingpin.Flag("bind", "gRPC bind address").Default("localhost:9610").StringVar(&serverConfig.BindAddress)
	kingpin.Flag("kubeconfig", "Path to .kube/config (or empty for in-cluster)").Default("").StringVar(&serverConfig.KubeConfig)
	kingpin.Flag("sync", "Pod cache sync interval").Default("1m").DurationVar(&serverConfig.PodSyncInterval)
	kingpin.Flag("role-base-arn", "Base ARN for roles. e.g. arn:aws:iam::123456789:role/").Required().StringVar(&serverConfig.RoleBaseARN)
	kingpin.Flag("session", "Session name used when creating STS Tokens.").Default("kiam").StringVar(&serverConfig.SessionName)

	kingpin.Flag("cert", "Server certificate path").Required().ExistingFileVar(&serverConfig.TLS.ServerCert)
	kingpin.Flag("key", "Server private key path").Required().ExistingFileVar(&serverConfig.TLS.ServerKey)
	kingpin.Flag("ca", "CA path").Required().ExistingFileVar(&serverConfig.TLS.CA)

	kingpin.Parse()

	if flags.jsonLog {
		log.SetFormatter(&log.JSONFormatter{})
	}

	switch flags.logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	}

	if flags.statsd != "" {
		addr, err := net.ResolveUDPAddr("udp", flags.statsd)
		if err != nil {
			log.Fatal("error parsing statsd address:", err.Error())
		}
		go statsd.StatsD(metrics.DefaultRegistry, flags.statsdInterval, "kiam.server", addr)
	}

	log.Infof("starting server")
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	signal.Notify(stopChan, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := serv.NewServer(serverConfig)
	if err != nil {
		log.Fatal("error creating listener:", err.Error())
	}

	go func() {
		<-stopChan
		log.Infof("stopping server")
		server.Stop()
	}()

	log.Infof("will serve on %s", serverConfig.BindAddress)

	server.Serve(ctx)

	log.Infoln("stopped")
}
