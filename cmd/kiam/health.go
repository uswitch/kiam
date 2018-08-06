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
	logOptions
	tlsOptions
	clientOptions
	timeout time.Duration
}

func (cmd *healthCommand) Bind(parser parser) {
	cmd.logOptions.bind(parser)
	cmd.tlsOptions.bind(parser)
	cmd.clientOptions.bind(parser)

	parser.Flag("timeout", "Timeout for health check").Default("1s").DurationVar(&cmd.timeout)
}

func (opts *healthCommand) Run() {
	opts.configureLogger()

	ctxGateway, cancelCtxGateway := context.WithTimeout(context.Background(), opts.timeoutKiamGateway)
	defer cancelCtxGateway()

	gateway, err := kiamserver.NewGateway(ctxGateway, opts.serverAddress, opts.serverAddressRefresh, opts.caPath, opts.certificatePath, opts.keyPath)
	if err != nil {
		log.Fatalf("error creating server gateway: %s", err.Error())
	}
	defer gateway.Close()

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
