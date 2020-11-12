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

	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/pprof"
	"github.com/uswitch/kiam/pkg/prometheus"
	"google.golang.org/grpc/keepalive"
)

type logOptions struct {
	jsonLog  bool
	logLevel string
}

func (o *logOptions) bind(parser parser) {
	parser.Flag("json-log", "Output log in JSON").BoolVar(&o.jsonLog)
	parser.Flag("level", "Log level: debug, info, warn, error.").Default("info").EnumVar(&o.logLevel, "debug", "info", "warn", "error")
}

func (o *logOptions) configureLogger() {
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

type telemetryOptions struct {
	prometheusListen string
	prometheusSync   time.Duration
	pprofListen      string
}

func (o *telemetryOptions) bind(parser parser) {
	parser.Flag("prometheus-listen-addr", "Prometheus HTTP listen address. e.g. localhost:9620").StringVar(&o.prometheusListen)
	parser.Flag("prometheus-sync-interval", "How frequently to update Prometheus metrics").Default("5s").DurationVar(&o.prometheusSync)

	parser.Flag("pprof-listen-addr", "Address to bind pprof HTTP server. e.g. localhost:9990").Default("").StringVar(&o.pprofListen)
}

func (o telemetryOptions) start(ctx context.Context, identifier string) {
	if o.prometheusListen != "" {
		metrics := prometheus.NewServer(identifier, o.prometheusListen, o.prometheusSync)
		metrics.Listen(ctx)
	}

	if o.pprofListen != "" {
		log.Infof("pprof listen address specified, will listen on %s", o.pprofListen)
		server := pprof.NewServer(o.pprofListen)
		go pprof.ListenAndWait(ctx, server)
	}
}

type tlsOptions struct {
	certificatePath string
	keyPath         string
	caPath          string
}

func (o *tlsOptions) bind(parser parser) {
	parser.Flag("cert", "Certificate path").Required().ExistingFileVar(&o.certificatePath)
	parser.Flag("key", "Key path").Required().ExistingFileVar(&o.keyPath)
	parser.Flag("ca", "CA certificate path").Required().ExistingFileVar(&o.caPath)
}

type clientOptions struct {
	serverAddress        string
	serverAddressRefresh time.Duration
	timeoutKiamGateway   time.Duration
	keepaliveParams      keepalive.ClientParameters
}

func (o *clientOptions) bind(parser parser) {
	parser.Flag("grpc-keepalive-time-duration", "gRPC keepalive time").Default("10s").DurationVar(&o.keepaliveParams.Time)
	parser.Flag("grpc-keepalive-timeout-duration", "gRPC keepalive timeout").Default("2s").DurationVar(&o.keepaliveParams.Timeout)
	parser.Flag("grpc-keepalive-permit-without-stream", "gRPC keepalive ping even with no RPC").BoolVar(&o.keepaliveParams.PermitWithoutStream)
	parser.Flag("server-address", "gRPC address to Kiam server service").Default("localhost:9610").StringVar(&o.serverAddress)
	parser.Flag("server-address-refresh", "Interval to refresh server service endpoints ( deprecated )").Default("0s").DurationVar(&o.serverAddressRefresh)
	parser.Flag("gateway-timeout-creation", "Timeout to create the kiam gateway ").Default("1s").DurationVar(&o.timeoutKiamGateway)
	if o.serverAddressRefresh > 0 {
		log.Error("server-address-refresh is deprecated and not in use, please remove it from your configuration")
	}
}
