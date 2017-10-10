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
	log "github.com/sirupsen/logrus"
	serv "github.com/uswitch/kiam/pkg/server"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	opts := &serv.Config{}

	kingpin.Flag("bind", "gRPC bind address").Default("localhost:9610").StringVar(&opts.BindAddress)
	kingpin.Flag("kubeconfig", "Path to .kube/config (or empty for in-cluster)").Default("").StringVar(&opts.KubeConfig)
	kingpin.Flag("sync", "Pod cache sync interval").Default("1m").DurationVar(&opts.PodSyncInterval)
	kingpin.Flag("role-base-arn", "Base ARN for roles. e.g. arn:aws:iam::123456789:role/").Required().StringVar(&opts.RoleBaseARN)
	kingpin.Flag("host", "Host IP address.").Envar("HOST_IP").Required().StringVar(&opts.Host)

	kingpin.Parse()

	log.Infof("starting server")
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	signal.Notify(stopChan, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := serv.NewServer(opts)
	if err != nil {
		log.Fatal("error creating listener:", err.Error())
	}

	go func() {
		<-stopChan
		log.Infof("stopping server")
		server.Stop()
	}()

	log.Infof("will serve on %s", opts.BindAddress)
	server.Serve(ctx)

	log.Infoln("stopped")
}
