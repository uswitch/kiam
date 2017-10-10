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
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/patrickmn/go-cache"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"time"
)

type credentialsCache struct {
	baseARN        string
	cache          *cache.Cache
	expiring       chan *RoleCredentials
	sessionName    string
	meterCacheHit  metrics.Meter
	meterCacheMiss metrics.Meter
	session        *session.Session
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

func DefaultCache(roleBaseARN, sessionName string) *credentialsCache {
	c := &credentialsCache{
		baseARN:        roleBaseARN,
		expiring:       make(chan *RoleCredentials, 1),
		sessionName:    fmt.Sprintf("kiam-%s", sessionName),
		meterCacheHit:  metrics.GetOrRegisterMeter("credentialsCache.cacheHit", metrics.DefaultRegistry),
		meterCacheMiss: metrics.GetOrRegisterMeter("credentialsCache.cacheMiss", metrics.DefaultRegistry),
		session:        session.Must(session.NewSession()),
	}
	c.cache = cache.New(DefaultCacheTTL, DefaultPurgeInterval)
	c.cache.OnEvicted(c.evicted)

	metrics.NewRegisteredFunctionalGauge("credentialsCacheSize", metrics.DefaultRegistry, func() int64 { return int64(c.cache.ItemCount()) })

	return c
}

func (c *credentialsCache) evicted(role string, item interface{}) {
	creds, ok := item.(*Credentials)
	if !ok {
		log.Errorf("type error, something other than *Credentials in cache")
		return
	}

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
	item, found := c.cache.Get(role)

	if found {
		c.meterCacheHit.Mark(1)

		creds, _ := item.(*Credentials)
		return creds, nil
	}

	c.meterCacheMiss.Mark(1)

	arn := fmt.Sprintf("%s%s", c.baseARN, role)
	credentials, err := issueNewCredentials(c.session, arn, c.sessionName, DefaultCredentialsValidityPeriod)
	if err != nil {
		return nil, err
	}

	log.WithFields(CredentialsFields(credentials, role)).Infof("requested new credentials")

	c.cache.Set(role, credentials, DefaultCacheTTL)

	return credentials, nil
}
