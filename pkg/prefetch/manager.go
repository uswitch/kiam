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
package prefetch

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/uswitch/kiam/pkg/aws/sts"
	"github.com/uswitch/kiam/pkg/k8s"
	"k8s.io/client-go/pkg/api/v1"
)

type CredentialManager struct {
	cache     sts.CredentialsCache
	finder    k8s.PodFinder
	announcer k8s.PodAnnouncer
}

func NewManager(cache sts.CredentialsCache, finder k8s.PodFinder, announcer k8s.PodAnnouncer) *CredentialManager {
	return &CredentialManager{cache: cache, finder: finder, announcer: announcer}
}

func (m *CredentialManager) fetchCredentials(pod *v1.Pod) {
	logger := log.WithFields(k8s.PodFields(pod))
	if k8s.IsPodCompleted(pod) {
		logger.Debugf("ignoring fetch credentials for completed pod")
		return
	}

	role := k8s.PodRole(pod)
	issued, err := m.fetchCredentialsForRole(role)
	if err != nil {
		logger.Errorf("error warming credentials: %s", err.Error())
	} else {
		logger.WithFields(sts.CredentialsFields(issued, role)).Infof("warming credentials")
	}
}

func (m *CredentialManager) fetchCredentialsForRole(role string) (*sts.Credentials, error) {
	return m.cache.CredentialsForRole(role)
}

func (m *CredentialManager) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case pod := <-m.announcer.Pods():
			log.WithFields(k8s.PodFields(pod)).Debugf("pod announced")
			m.fetchCredentials(pod)
		case expiring := <-m.cache.Expiring():
			m.handleExpiring(expiring)
		}
	}
}

func (m *CredentialManager) handleExpiring(credentials *sts.RoleCredentials) {
	logger := log.WithFields(sts.CredentialsFields(credentials.Credentials, credentials.Role))

	active, err := m.IsRoleActive(credentials.Role)
	if err != nil {
		logger.Errorf("error checking whether role active: %s", err.Error())
		return
	}

	if !active {
		logger.Infof("role no longer active")
		return
	}

	logger.Infof("expiring credentials, fetching updated")
	_, err = m.fetchCredentialsForRole(credentials.Role)
	if err != nil {
		logger.Errorf("error fetching updated credentials for expiring: %s", err.Error())
	}
}

func (m *CredentialManager) IsRoleActive(role string) (bool, error) {
	return m.finder.IsActivePodsForRole(role)
}
