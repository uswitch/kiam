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
	"github.com/uswitch/k8sc/official"
	"github.com/uswitch/kiam/pkg/creds"
	http "github.com/uswitch/kiam/pkg/http"
	"github.com/uswitch/kiam/pkg/k8s"
	"github.com/uswitch/kiam/pkg/prefetch"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type options struct {
	jsonLog        bool
	debug          bool
	kubeconfig     string
	port           int
	allowIPQuery   bool
	syncInterval   time.Duration
	roleBaseARN    string
	statsD         string
	statsDInterval time.Duration
	iptables       bool
	hostIP         string
	hostInterface  string
}

func main() {
	opts := &options{}

	kingpin.Flag("json-log", "Output log in JSON").BoolVar(&opts.jsonLog)
	kingpin.Flag("debug", "Log at Debug level").Short('d').BoolVar(&opts.debug)
	kingpin.Flag("kubeconfig", "Path to kube config").StringVar(&opts.kubeconfig)
	kingpin.Flag("port", "HTTP port").Default("3100").IntVar(&opts.port)
	kingpin.Flag("sync-interval", "Interval to refresh pod state from API server").Default("2m").DurationVar(&opts.syncInterval)
	kingpin.Flag("allow-ip-query", "Allow client IP to be specified with ?ip. Development use only.").Default("false").BoolVar(&opts.allowIPQuery)
	kingpin.Flag("role-base-arn", "Base ARN for roles. e.g. arn:aws:iam::123456789:role/").Required().StringVar(&opts.roleBaseARN)

	kingpin.Flag("statsd", "UDP address to publish StatsD metrics. e.g. 127.0.0.1:8125").Default("").StringVar(&opts.statsD)
	kingpin.Flag("statsd-interval", "Interval to publish to StatsD").Default("10s").DurationVar(&opts.statsDInterval)

	kingpin.Flag("iptables", "Add IPTables rules").Default("false").BoolVar(&opts.iptables)
	kingpin.Flag("host", "Host IP address.").Envar("HOST_IP").Required().StringVar(&opts.hostIP)
	kingpin.Flag("host-interface", "Network interface for pods to configure IPTables.").Default("docker0").StringVar(&opts.hostInterface)
	kingpin.Parse()

	if opts.jsonLog {
		log.SetFormatter(&log.JSONFormatter{})
	}
	if opts.debug {
		log.SetLevel(log.DebugLevel)
	}

	if opts.iptables {
		log.Infof("configuring iptables")
		rules := newIPTablesRules(opts.hostIP, opts.port, opts.hostInterface)
		err := rules.Add()
		if err != nil {
			log.Fatal("error configuring iptables:", err.Error())
		}
		defer rules.Remove()
	}

	if opts.statsD != "" {
		addr, err := net.ResolveUDPAddr("udp", opts.statsD)
		if err != nil {
			log.Fatal("error parsing statsd address:", err.Error())
		}
		go statsd.StatsD(metrics.DefaultRegistry, opts.statsDInterval, "kiam", addr)
	}

	client, err := official.NewClient(opts.kubeconfig)
	if err != nil {
		log.Fatalf("couldn't create kubernetes client: %s", err.Error())
	}

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	signal.Notify(stopChan, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := http.NewConfig(opts.port)
	config.AllowIPQuery = opts.allowIPQuery

	finder := k8s.PodCache(k8s.KubernetesSource(client), opts.syncInterval)
	finder.Run(ctx)

	credentials := creds.Default(opts.roleBaseARN, opts.hostIP)
	manager := prefetch.NewManager(credentials, finder)
	go manager.Run(ctx)

	server := http.NewWebServer(config, finder, credentials)
	go server.Serve()
	defer server.Stop(ctx)

	<-stopChan
	log.Infoln("terminating...")
}
