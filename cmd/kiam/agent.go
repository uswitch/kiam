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
	http "github.com/uswitch/kiam/pkg/aws/metadata"
	kiamserver "github.com/uswitch/kiam/pkg/server"
)

type agentCommand struct {
	logOptions
	telemetryOptions
	tlsOptions
	clientOptions
	*http.ServerOptions

	iptables       bool
	iptablesRemove bool
	hostIP         string
	hostInterface  string
}

func (cmd *agentCommand) Bind(parser parser) {
	cmd.logOptions.bind(parser)
	cmd.telemetryOptions.bind(parser)
	cmd.tlsOptions.bind(parser)
	cmd.clientOptions.bind(parser)

	cmd.ServerOptions = http.DefaultOptions()

	parser.Flag("port", "HTTP port").Default("3100").IntVar(&cmd.ListenPort)
	parser.Flag("allow-ip-query", "Allow client IP to be specified with ?ip. Development use only.").Default("false").BoolVar(&cmd.AllowIPQuery)
	parser.Flag("whitelist-route-regexp", "Proxy routes matching this regular expression").Default("^$").RegexpVar(&cmd.WhitelistRouteRegexp)

	parser.Flag("iptables", "Add IPTables rules").Default("false").BoolVar(&cmd.iptables)
	parser.Flag("iptables-remove", "Remove iptables rules at shutdown").Default("true").BoolVar(&cmd.iptablesRemove)
	parser.Flag("host", "Host IP address.").Envar("HOST_IP").Required().StringVar(&cmd.hostIP)
	parser.Flag("host-interface", "Network interface for pods to configure IPTables.").Default("docker0").StringVar(&cmd.hostInterface)
}

// run is the actual run implementation.
func (opts *agentCommand) run() error {
	opts.configureLogger()

	if opts.iptables {
		log.Infof("configuring iptables")
		rules := newIPTablesRules(opts.hostIP, opts.ListenPort, opts.hostInterface)
		err := rules.Add()
		if err != nil {
			log.Errorf("error configuring iptables: %s", err.Error())
			return err
		}
		if opts.iptablesRemove {
			defer func() {
				log.Infof("undoing iptables changes")
				rules.Remove()
			}()
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go opts.telemetryOptions.start(ctx, "agent")

	stopChan := make(chan os.Signal, 8)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	ctxGateway, cancelCtxGateway := context.WithTimeout(context.Background(), opts.timeoutKiamGateway)
	defer cancelCtxGateway()

	gateway, err := kiamserver.NewGateway(ctxGateway, opts.serverAddress, opts.caPath, opts.certificatePath, opts.keyPath, opts.keepaliveParams)
	if err != nil {
		log.Errorf("error creating server gateway: %s", err.Error())
		return err
	}
	defer gateway.Close()

	server, err := http.NewWebServer(opts.ServerOptions, gateway)
	if err != nil {
		log.Errorf("error creating agent http server: %s", err.Error())
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve()
	}()

	select {
	case err := <-errCh:
		if err != nil {
			log.Errorf("error running server: %s", err.Error())
			return err
		}
	case sig := <-stopChan:
		log.Infof("received signal (%s): starting server shutdown", sig.String())
		if err := server.Stop(ctx); err != nil {
			log.Errorf("error shutting down server: %s", err.Error())
			return err
		}
		log.Infoln("gracefully shutdown server")
	}
	log.Infoln("stopped")
	return nil
}

func (opts *agentCommand) Run() {
	if err := opts.run(); err != nil {
		log.Fatalf("fatal error: %s", err.Error())
	}
}
