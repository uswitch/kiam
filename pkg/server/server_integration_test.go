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

package server

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/miekg/dns"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/k8s"
	"google.golang.org/grpc"
	kt "k8s.io/client-go/tools/cache/testing"
)

const (
	kServerAddress = "127.0.0.1:8899"
)

func TestManagesResolutionFailures(t *testing.T) {
	defer leaktest.Check(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, err := newServerAndListen(ctx)
	if err != nil {
		t.Fatal(err)
	}

	dnsServer := dns.Server{Addr: "127.0.0.1:5351", Net: "udp"}
	go dnsServer.ListenAndServe()
	defer dnsServer.ShutdownContext(ctx)
	dns.HandleFunc("kiam-server.localdomain.", func(w dns.ResponseWriter, r *dns.Msg) {
		a := &dns.A{
			Hdr: dns.RR_Header{Name: "kiam-server.localdomain.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 10},
			A:   net.ParseIP("127.0.0.1"),
		}
		m := new(dns.Msg)
		m.Answer = []dns.RR{a}
		m.SetReply(r)
		w.WriteMsg(m)
	})

	b := newClientBuilder().WithAddress("kiam-server:8899").WithDNSResolver("127.0.0.1:5351")
	client, err := b.Build(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Health(ctx)
	if err != nil {
		t.Error(err)
	}
}

func TestHealthReturnsOk(t *testing.T) {
	defer leaktest.Check(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, client, err := newSystemAndListen(ctx)
	if err != nil {
		t.Fatal(err)
	}

	ctxCall, cancelCall := context.WithTimeout(ctx, time.Second*5)
	defer cancelCall()
	health, err := client.Health(ctxCall)
	if err != nil {
		t.Error("error checking health: ", err)
	}

	if health != "ok" {
		t.Error("expected ok, was", health)
	}
}

func TestRetriesUntilServerAvailable(t *testing.T) {
	defer leaktest.Check(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, _, gateway, err := newSystemAndListen(ctx)
	if err != nil {
		t.Error(err)
	}
	server.Stop()

	ok := make(chan string)
	go func(ctx context.Context) {
		m, err := gateway.Health(ctx)
		if err == nil {
			ok <- m
		}
	}(ctx)

	server, _, err = newTestServer(ctx)
	go func(ctx context.Context) {
		server.Serve(ctx)
	}(ctx)
	defer server.Stop()

	select {
	case _ = <-ok:
		// all good!
	case <-time.After(time.Second * 30):
		t.Error("didn't complete in 5 seconds")
	}
}

func newServerAndListen(ctx context.Context) (*KiamServer, *kt.FakeControllerSource, error) {
	server, source, err := newTestServer(ctx)
	if err != nil {
		return nil, nil, err
	}
	go func() { server.Serve(ctx) }()
	go func() {
		<-ctx.Done()
		server.Stop()
	}()

	return server, source, nil
}

// newSystemAndListen creates a server and starts it listening, returning a gateway to connect to it
func newSystemAndListen(ctx context.Context) (*KiamServer, *kt.FakeControllerSource, *KiamGateway, error) {
	server, source, err := newServerAndListen(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	gateway, err := newClient(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	return server, source, gateway, err
}

func newClientBuilder() *KiamGatewayBuilder {
	return NewKiamGatewayBuilder().WithAddress(kServerAddress).WithDialOption(grpc.WithInsecure(), grpc.WithBlock()).WithMaxRetries(20).WithRetryInterval(time.Second)
}

func newClient(ctx context.Context) (*KiamGateway, error) {
	return newClientBuilder().Build(ctx)
}

// creates the server, returns the server and a source allowing objects to be added
// to the controllers: pods and namespaces
//
// e.g. source.Add(testutil.NewPodWithRole("ns", "name", "192.168.0.1", "Running", "running_role"))
func newTestServer(ctx context.Context) (*KiamServer, *kt.FakeControllerSource, error) {
	cfg := &Config{
		BindAddress: kServerAddress,
	}
	grpcServer := grpc.NewServer()

	source := kt.NewFakeControllerSource()
	defer source.Shutdown()

	podCache := k8s.NewPodCache(sts.DefaultResolver("arn:account:"), source, time.Second, defaultBuffer)
	podCache.Run(ctx)
	namespaceCache := k8s.NewNamespaceCache(source, time.Second)
	namespaceCache.Run(ctx)

	b := NewKiamServerBuilder(cfg).WithGRPCServer(grpcServer).WithCaches(podCache, namespaceCache)
	server, err := b.Build()
	if err != nil {
		return nil, nil, err
	}

	return server, source, err
}
