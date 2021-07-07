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
	"regexp"
	"time"

	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/future"
)

type credentialsCache struct {
	cache           *cache.Cache
	expiring        chan *CachedCredentials
	sessionName     string
	sessionDuration time.Duration
	cacheTTL        time.Duration
	gateway         STSGateway
}

type CachedCredentials struct {
	Identity    *RoleIdentity
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
) *credentialsCache {
	c := &credentialsCache{
		expiring:        make(chan *CachedCredentials, 1),
		sessionName:     sessionName,
		sessionDuration: sessionDuration,
		cacheTTL:        sessionDuration - sessionRefresh,
		gateway:         gateway,
	}
	c.cache = cache.New(c.cacheTTL, DefaultPurgeInterval)
	c.cache.OnEvicted(c.evicted)

	return c
}

func (c *credentialsCache) evicted(key string, item interface{}) {
	cacheSize.Dec()

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

// CredentialsForRole looks for cached credentials or requests them from the STSGateway. Requested credentials
// must have their ARN set.
func (c *credentialsCache) CredentialsForRole(ctx context.Context, identity *RoleIdentity) (*Credentials, error) {
	logger := log.WithFields(identity.LogFields())
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
		sessionName := c.getSessionName(identity)

		stsIssueRequest := &STSIssueRequest{
			RoleARN:         identity.Role.ARN,
			SessionName:     sessionName,
			ExternalID:      identity.ExternalID,
			SessionDuration: c.sessionDuration,
		}

		credentials, err := c.gateway.Issue(ctx, stsIssueRequest)
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
	cacheSize.Inc()

	val, err := f.Get(ctx)
	if err != nil {
		c.cache.Delete(identity.String())
		return nil, err
	}

	cachedCreds := val.(*CachedCredentials)
	return cachedCreds.Credentials, nil
}

func (c *credentialsCache) getSessionName(identity *RoleIdentity) string {
	sessionName := c.sessionName

	if identity.SessionName != "" {
		sessionName = identity.SessionName
	}

	sessionName = fmt.Sprintf("kiam-%s", sessionName)
	return sanitizeSessionName(sessionName)
}

// Ensure the session name meets length requirements and
// also coercce any character that doens't meet the pattern
// requirements to a hyhen so that we ensure a valid session name.
func sanitizeSessionName(sessionName string) string {
	sanitize := regexp.MustCompile(`([^\w+=,.@-])`)

	if len(sessionName) > 64 {
		sessionName = sessionName[0:63]
	}

	return sanitize.ReplaceAllString(sessionName, "-")
}
