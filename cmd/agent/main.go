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
	http "github.com/uswitch/kiam/pkg/aws/metadata"
	"github.com/uswitch/kiam/pkg/prometheus"
	kiamserver "github.com/uswitch/kiam/pkg/server"
	"gopkg.in/alecthomas/kingpin.v2"
)

type options struct {
	jsonLog              bool
	logLevel             string
	port                 int
	allowIPQuery         bool
	statsD               string
	statsDInterval       time.Duration
	iptables             bool
	hostIP               string
	hostInterface        string
	serverAddress        string
	serverAddressRefresh time.Duration
	prometheusListen     string
	prometheusSync       time.Duration

	certificatePath string
	keyPath         string
	caPath          string
}

func (o *options) configureLogger() {
	if o.jsonLog {
		log.SetFormatter(&log.JSONFormatter{})
	}

	switch o.logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	}
}

func main() {
	opts := &options{}

	kingpin.Flag("json-log", "Output log in JSON").BoolVar(&opts.jsonLog)
	kingpin.Flag("level", "Log level: debug, info, warn, error.").Default("info").EnumVar(&opts.logLevel, "debug", "info", "warn", "error")

	kingpin.Flag("port", "HTTP port").Default("3100").IntVar(&opts.port)
	kingpin.Flag("allow-ip-query", "Allow client IP to be specified with ?ip. Development use only.").Default("false").BoolVar(&opts.allowIPQuery)

	kingpin.Flag("statsd", "UDP address to publish StatsD metrics. e.g. 127.0.0.1:8125").Default("").StringVar(&opts.statsD)
	kingpin.Flag("statsd-interval", "Interval to publish to StatsD").Default("10s").DurationVar(&opts.statsDInterval)

	kingpin.Flag("iptables", "Add IPTables rules").Default("false").BoolVar(&opts.iptables)
	kingpin.Flag("host", "Host IP address.").Envar("HOST_IP").Required().StringVar(&opts.hostIP)
	kingpin.Flag("host-interface", "Network interface for pods to configure IPTables.").Default("docker0").StringVar(&opts.hostInterface)

	kingpin.Flag("server-address", "gRPC address to Kiam server service").Default("localhost:9610").StringVar(&opts.serverAddress)
	kingpin.Flag("server-address-refresh", "Interval to refresh server service endpoints").Default("10s").DurationVar(&opts.serverAddressRefresh)
	kingpin.Flag("cert", "Agent certificate path").Required().ExistingFileVar(&opts.certificatePath)
	kingpin.Flag("key", "Agent key path").Required().ExistingFileVar(&opts.keyPath)
	kingpin.Flag("ca", "CA certificate path").Required().ExistingFileVar(&opts.caPath)

	kingpin.Flag("prometheus-listen-addr", "Prometheus HTTP listen address. e.g. localhost:9620").StringVar(&opts.prometheusListen)
	kingpin.Flag("prometheus-sync-interval", "How frequently to update Prometheus metrics").Default("5s").DurationVar(&opts.prometheusSync)

	kingpin.Parse()

	opts.configureLogger()

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
		go statsd.StatsD(metrics.DefaultRegistry, opts.statsDInterval, "kiam.agent", addr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if opts.prometheusListen != "" {
		metrics := prometheus.NewServer("agent", opts.prometheusListen, opts.prometheusSync)
		metrics.Listen(ctx)
	}

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	signal.Notify(stopChan, syscall.SIGTERM)

	config := http.NewConfig(opts.port)
	config.AllowIPQuery = opts.allowIPQuery

	gateway, err := kiamserver.NewGateway(opts.serverAddress, opts.serverAddressRefresh, opts.caPath, opts.certificatePath, opts.keyPath)
	if err != nil {
		log.Fatalf("error creating server gateway: %s", err.Error())
	}

	server, err := http.NewWebServer(config, gateway, gateway, gateway)
	if err != nil {
		log.Fatalf("error creating agent http server: %s", err.Error())
	}

	go server.Serve()
	defer server.Stop(ctx)

	<-stopChan
	log.Infoln("stopped")
}
