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
	"regexp"
	"syscall"

	log "github.com/sirupsen/logrus"
	http "github.com/uswitch/kiam/pkg/aws/metadata"
	kiamserver "github.com/uswitch/kiam/pkg/server"
)

type agentCommand struct {
	logOptions
	telemetryOptions
	tlsOptions
	clientOptions

	port          int
	allowIPQuery  bool
	allowedRoutes *regexp.Regexp
	iptables      bool
	hostIP        string
	hostInterface string
}

func (cmd *agentCommand) Bind(parser parser) {
	cmd.logOptions.bind(parser)
	cmd.telemetryOptions.bind(parser)
	cmd.tlsOptions.bind(parser)
	cmd.clientOptions.bind(parser)

	parser.Flag("port", "HTTP port").Default("3100").IntVar(&cmd.port)
	parser.Flag("allow-ip-query", "Allow client IP to be specified with ?ip. Development use only.").Default("false").BoolVar(&cmd.allowIPQuery)
	parser.Flag("allowed-routes", "Proxy routes matching this regular expression").Default(".*").RegexpVar(&cmd.allowedRoutes)

	parser.Flag("iptables", "Add IPTables rules").Default("false").BoolVar(&cmd.iptables)
	parser.Flag("host", "Host IP address.").Envar("HOST_IP").Required().StringVar(&cmd.hostIP)
	parser.Flag("host-interface", "Network interface for pods to configure IPTables.").Default("docker0").StringVar(&cmd.hostInterface)
}

func (opts *agentCommand) Run() {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go opts.telemetryOptions.start(ctx, "agent")

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	signal.Notify(stopChan, syscall.SIGTERM)

	config := http.NewConfig(opts.port)
	config.AllowIPQuery = opts.allowIPQuery
	config.AllowedRoutes = opts.allowedRoutes

	ctxGateway, cancelCtxGateway := context.WithTimeout(context.Background(), opts.timeoutKiamGateway)
	defer cancelCtxGateway()

	gateway, err := kiamserver.NewGateway(ctxGateway, opts.serverAddress, opts.serverAddressRefresh, opts.caPath, opts.certificatePath, opts.keyPath)
	if err != nil {
		log.Fatalf("error creating server gateway: %s", err.Error())
	}

	server, err := http.NewWebServer(config, gateway)
	if err != nil {
		log.Fatalf("error creating agent http server: %s", err.Error())
	}

	go server.Serve()
	defer server.Stop(ctx)

	<-stopChan
	log.Infoln("stopped")
}
