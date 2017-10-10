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

type options struct {
	bind string
}

func main() {
	opts := &options{}
	kingpin.Flag("bind", "gRPC bind address").Default("localhost:9610").StringVar(&opts.bind)
	kingpin.Parse()

	log.Infof("starting server")
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	signal.Notify(stopChan, syscall.SIGTERM)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := serv.NewServer(opts.bind)
	if err != nil {
		log.Fatal("error creating listener:", err.Error())
	}

	go func() {
		<-stopChan
		log.Infof("stopping server")
		server.Stop()
	}()

	log.Infof("will serve on %s", opts.bind)
	server.Serve()

	log.Infoln("stopped")
}
