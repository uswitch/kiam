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
	"github.com/uswitch/kiam/cmd/agent"
	"github.com/uswitch/kiam/cmd/health"
	"github.com/uswitch/kiam/cmd/server"
	serv "github.com/uswitch/kiam/pkg/server"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	rootParser := kingpin.CommandLine

	agentParser := rootParser.Command("agent", "run the agent")
	agentOpts := &agent.Options{}
	agentOpts.Bind(agentParser)

	serverParser := rootParser.Command("server", "run the server")
	serverOpts := &server.Options{Config: &serv.Config{TLS: &serv.TLSConfig{}}}
	serverOpts.Bind(serverParser)

	healthParser := rootParser.Command("health", "run the health check")
	healthOpts := &health.Options{}
	healthOpts.Bind(healthParser)

	switch kingpin.Parse() {
	case "agent":
		agentOpts.Run()
	case "server":
		serverOpts.Run()
	case "health":
		healthOpts.Run()
	}
}
