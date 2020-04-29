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
package sts

import (
	"context"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/future"
)

type credentialsCache struct {
	arnResolver     ARNResolver
	cache           *cache.Cache
	expiring        chan *RoleCredentials
	sessionName     string
	sessionDuration time.Duration
	cacheTTL        time.Duration
	gateway         STSGateway
}

type RoleCredentials struct {
	Role        string
	ExternalID  string
	Credentials *Credentials
}

const (
	DefaultPurgeInterval = 1 * time.Minute
)

func DefaultCache(
	gateway STSGateway,
	sessionName string,
	sessionDuration time.Duration,
	sessionRefresh time.Duration,
	resolver ARNResolver,
) *credentialsCache {
	c := &credentialsCache{
		arnResolver:     resolver,
		expiring:        make(chan *RoleCredentials, 1),
		sessionName:     fmt.Sprintf("kiam-%s", sessionName),
		sessionDuration: sessionDuration,
		cacheTTL:        sessionDuration - sessionRefresh,
		gateway:         gateway,
	}
	c.cache = cache.New(c.cacheTTL, DefaultPurgeInterval)
	c.cache.OnEvicted(c.evicted)

	// TODO: Not do this inline
	cacheSize := prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Namespace: "kiam",
			Subsystem: "sts",
			Name:      "cacheSize",
			Help:      "Current size of the metadata cache",
		},
		func() float64 { return float64(c.cache.ItemCount()) },
	)
	prometheus.MustRegister(cacheSize)

	return c
}

func (c *credentialsCache) evicted(role string, item interface{}) {
	f := item.(*future.Future)
	obj, err := f.Get(context.Background())

	if err != nil {
		log.WithField("pod.iam.role", role).Debugf("evicted credentials future had error: %s", err.Error())
		return
	}

	creds := obj.(*Credentials)
	select {
	case c.expiring <- &RoleCredentials{Role: role, Credentials: creds}:
		log.WithFields(CredentialsFields(creds, role)).Infof("notified credentials expire soon")
		return
	default:
		return
	}
}

func (c *credentialsCache) Expiring() chan *RoleCredentials {
	return c.expiring
}

func (c *credentialsCache) CredentialsForRole(ctx context.Context, role string, externalID string) (*Credentials, error) {
	logger := log.WithFields(log.Fields{"pod.iam.role": role})
	item, found := c.cache.Get(role)

	if found {
		future, _ := item.(*future.Future)
		val, err := future.Get(ctx)

		if err != nil {
			logger.Errorf("error retrieving credentials in cache from future: %s. will delete", err.Error())
			c.cache.Delete(role)
			return nil, err
		}

		cacheHit.Inc()

		return val.(*Credentials), nil
	}

	cacheMiss.Inc()

	issue := func() (interface{}, error) {
		arn := c.arnResolver.Resolve(role)
		credentials, err := c.gateway.Issue(ctx, &STSGatewayRequest{
			roleARN:     arn,
			externalID:  externalID,
			sessionName: c.sessionName,
			expiry:      c.sessionDuration})

		if err != nil {
			errorIssuing.Inc()
			logger.Errorf("error requesting credentials: %s", err.Error())
			return nil, err
		}

		log.WithFields(CredentialsFields(credentials, role)).Infof("requested new credentials")
		return credentials, err
	}
	f := future.New(issue)
	c.cache.Set(role, f, c.cacheTTL)

	val, err := f.Get(ctx)
	if err != nil {
		c.cache.Delete(role)
		return nil, err
	}

	return val.(*Credentials), nil
}
