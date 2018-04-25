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
	"fmt"
	"github.com/coreos/go-iptables/iptables"
	"strings"
)

type rules struct {
	host          string
	kiamPort      int
	hostInterface string
}

const (
	metadataAddress = "169.254.169.254"
)

func newIPTablesRules(host string, kiamPort int, hostInterface string) *rules {
	return &rules{host: host, kiamPort: kiamPort, hostInterface: hostInterface}
}

func (r *rules) Add() error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}

	return ipt.AppendUnique("nat", "PREROUTING", r.ruleSpec()...)
}

func (r *rules) ruleSpec() []string {
	rules := []string{
		"-p", "tcp",
		"-d", metadataAddress,
		"--dport", "80",
		"-j", "DNAT",
		"--to-destination", r.kiamAddress(),
	}
	if strings.HasPrefix(r.hostInterface, "!") {
		rules = append(rules, "!")
	}
	rules = append(rules, "-i", strings.TrimPrefix(r.hostInterface, "!"))
	
	return rules
}

func (r *rules) Remove() error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}

	return ipt.Delete("nat", "PREROUTING", r.ruleSpec()...)
}

func (r *rules) kiamAddress() string {
	return fmt.Sprintf("%s:%d", r.host, r.kiamPort)
}
