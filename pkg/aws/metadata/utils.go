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
package metadata

import (
	"context"
	"github.com/cenkalti/backoff"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/server"
	"time"
)

const (
	retryInterval = time.Millisecond * 5
)

func findRole(ctx context.Context, client server.Client, ip string) (string, error) {
	logger := log.WithField("pod.ip", ip)

	roleCh := make(chan string, 1)
	op := func() error {
		role, err := client.GetRole(ctx, ip)
		if err != nil {
			logger.Warnf("error finding role for pod: %s", err.Error())
			return err
		}
		roleCh <- role
		return nil
	}

	strategy := backoff.NewExponentialBackOff()
	strategy.InitialInterval = retryInterval

	err := backoff.Retry(op, backoff.WithContext(strategy, ctx))
	if err != nil {
		return "", err
	}

	return <-roleCh, nil
}
