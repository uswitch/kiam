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
	baseARN         string
	cache           *cache.Cache
	expiring        chan *CachedCredentials
	sessionName     string
	sessionDuration time.Duration
	cacheTTL        time.Duration
	gateway         STSGateway
}

type CredentialsIdentity struct {
	Role string
}

type CachedCredentials struct {
	Identity    *CredentialsIdentity
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
		expiring:        make(chan *CachedCredentials, 1),
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

func (c *credentialsCache) evicted(key string, item interface{}) {
	f := item.(*future.Future)
	obj, err := f.Get(context.Background())

	if err != nil {
		log.WithField("cache.key", key).Debugf("evicted credentials future had error: %s", err.Error())
		return
	}

	cachedCreds := obj.(*CachedCredentials)
	select {
	case c.expiring <- cachedCreds:
		log.WithFields(CredentialsFields(cachedCreds.Identity, cachedCreds.Credentials)).Infof("notified credentials expire soon")
		return
	default:
		return
	}
}

func (c *credentialsCache) Expiring() chan *CachedCredentials {
	return c.expiring
}

func (c *credentialsCache) CredentialsForRole(ctx context.Context, identity *CredentialsIdentity) (*Credentials, error) {
	logger := log.WithFields(log.Fields{"pod.iam.role": identity.Role})
	item, found := c.cache.Get(identity.String())

	if found {
		future, _ := item.(*future.Future)
		val, err := future.Get(ctx)

		if err != nil {
			logger.Errorf("error retrieving credentials in cache from future: %s. will delete", err.Error())
			c.cache.Delete(identity.String())
			return nil, err
		}

		cacheHit.Inc()

		cachedCreds := val.(*CachedCredentials)
		return cachedCreds.Credentials, nil
	}

	cacheMiss.Inc()

	issue := func() (interface{}, error) {
		arn := c.arnResolver.Resolve(identity.Role)
		credentials, err := c.gateway.Issue(ctx, arn, c.sessionName, c.sessionDuration)
		if err != nil {
			errorIssuing.Inc()
			logger.Errorf("error requesting credentials: %s", err.Error())
			return nil, err
		}

		cachedCreds := &CachedCredentials{
			Identity:    identity,
			Credentials: credentials,
		}

		log.WithFields(CredentialsFields(identity, credentials)).Infof("requested new credentials")
		return cachedCreds, err
	}
	f := future.New(issue)
	c.cache.Set(identity.String(), f, c.cacheTTL)

	val, err := f.Get(ctx)
	if err != nil {
		c.cache.Delete(identity.String())
		return nil, err
	}

	cachedCreds := val.(*CachedCredentials)
	return cachedCreds.Credentials, nil
}

func (i *CredentialsIdentity) String() string {
	return i.Role
}
