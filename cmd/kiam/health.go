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
	"time"

	"github.com/cenkalti/backoff"
	log "github.com/sirupsen/logrus"
	kiamserver "github.com/uswitch/kiam/pkg/server"
)

type healthCommand struct {
	jsonLog              bool
	logLevel             string
	serverAddress        string
	serverAddressRefresh time.Duration
	timeoutKiamGateway   time.Duration
	certificatePath      string
	keyPath              string
	caPath               string
	timeout              time.Duration
}

func (o *healthCommand) Bind(parser parser) {
	parser.Flag("json-log", "Output log in JSON").BoolVar(&o.jsonLog)
	parser.Flag("level", "Log level: debug, info, warn, error.").Default("info").EnumVar(&o.logLevel, "debug", "info", "warn", "error")

	parser.Flag("server-address", "gRPC address to Kiam server service").Default("localhost:9610").StringVar(&o.serverAddress)
	parser.Flag("server-address-refresh", "Interval to refresh server service endpoints").Default("10s").DurationVar(&o.serverAddressRefresh)
	parser.Flag("gateway-timeout-creation", "Timeout to create the kiam gateway ").Default("50ms").DurationVar(&o.timeoutKiamGateway)
	parser.Flag("cert", "Agent certificate path").Required().ExistingFileVar(&o.certificatePath)
	parser.Flag("key", "Agent key path").Required().ExistingFileVar(&o.keyPath)
	parser.Flag("ca", "CA certificate path").Required().ExistingFileVar(&o.caPath)

	parser.Flag("timeout", "Timeout for health check").Default("1s").DurationVar(&o.timeout)
}

func (opts *healthCommand) Run() {
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

	ctxGateway, cancelCtxGateway := context.WithTimeout(context.Background(), opts.timeoutKiamGateway)
	defer cancelCtxGateway()

	gateway, err := kiamserver.NewGateway(ctxGateway, opts.serverAddress, opts.serverAddressRefresh, opts.caPath, opts.certificatePath, opts.keyPath)
	if err != nil {
		log.Fatalf("error creating server gateway: %s", err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	op := func() error {
		message, err := gateway.Health(ctx)
		if err != nil {
			log.Warnf("error checking health: %s", err.Error())
			return err
		}

		log.Infof("healthy: %s", message)

		return nil
	}
	err = backoff.Retry(op, backoff.WithContext(backoff.NewConstantBackOff(100*time.Millisecond), ctx))

	if err != nil {
		log.Fatalf("error retrieving health: %s", err.Error())
	}
}
