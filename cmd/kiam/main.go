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
	serv "github.com/uswitch/kiam/pkg/server"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	rootParser := kingpin.CommandLine

	agent := &agentCommand{}
	agent.Bind(rootParser.Command("agent", "run the agent"))

	server := &serverCommand{Config: &serv.Config{TLS: &serv.TLSConfig{}}}
	server.Bind(rootParser.Command("server", "run the server"))

	health := &healthCommand{}
	health.Bind(rootParser.Command("health", "run the health check"))

	switch kingpin.Parse() {
	case "agent":
		agent.Run()
	case "server":
		server.Run()
	case "health":
		health.Run()
	}
}

type parser interface {
	Flag(name, help string) *kingpin.FlagClause
}
