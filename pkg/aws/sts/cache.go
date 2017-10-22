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
	"github.com/patrickmn/go-cache"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/future"
	"time"
)

type credentialsCache struct {
	baseARN        string
	cache          *cache.Cache
	expiring       chan *RoleCredentials
	sessionName    string
	meterCacheHit  metrics.Meter
	meterCacheMiss metrics.Meter
	gateway        STSGateway
}

type RoleCredentials struct {
	Role        string
	Credentials *Credentials
}

const (
	DefaultPurgeInterval             = 1 * time.Minute
	DefaultCredentialsValidityPeriod = 15 * time.Minute
	DefaultCacheTTL                  = 10 * time.Minute
)

func DefaultCache(gateway STSGateway, roleBaseARN, sessionName string) *credentialsCache {
	c := &credentialsCache{
		baseARN:        roleBaseARN,
		expiring:       make(chan *RoleCredentials, 1),
		sessionName:    fmt.Sprintf("kiam-%s", sessionName),
		meterCacheHit:  metrics.GetOrRegisterMeter("credentialsCache.cacheHit", metrics.DefaultRegistry),
		meterCacheMiss: metrics.GetOrRegisterMeter("credentialsCache.cacheMiss", metrics.DefaultRegistry),
		gateway:        gateway,
	}
	c.cache = cache.New(DefaultCacheTTL, DefaultPurgeInterval)
	c.cache.OnEvicted(c.evicted)

	metrics.NewRegisteredFunctionalGauge("credentialsCache.size", metrics.DefaultRegistry, func() int64 { return int64(c.cache.ItemCount()) })

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

func (c *credentialsCache) CredentialsForRole(ctx context.Context, role string) (*Credentials, error) {
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

		c.meterCacheHit.Mark(1)

		return val.(*Credentials), nil
	}

	c.meterCacheMiss.Mark(1)

	issue := func() (interface{}, error) {
		arn := fmt.Sprintf("%s%s", c.baseARN, role)
		credentials, err := c.gateway.Issue(ctx, arn, c.sessionName, DefaultCredentialsValidityPeriod)
		if err != nil {
			metrics.GetOrRegisterMeter("credentialsCache.errorIssuing", metrics.DefaultRegistry).Mark(1)
			logger.Errorf("error requesting credentials: %s", err.Error())
			return nil, err
		}

		log.WithFields(CredentialsFields(credentials, role)).Infof("requested new credentials")
		return credentials, err
	}
	f := future.New(issue)
	c.cache.Set(role, f, DefaultCacheTTL)

	val, err := f.Get(ctx)
	if err != nil {
		c.cache.Delete(role)
		return nil, err
	}

	return val.(*Credentials), nil
}
