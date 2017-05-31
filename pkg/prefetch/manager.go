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
	"github.com/uswitch/kiam/pkg/creds"
	"github.com/uswitch/kiam/pkg/k8s"
	"k8s.io/client-go/pkg/api/v1"
)

type CredentialManager struct {
	issuer creds.CredentialsIssuer
	finder k8s.PodFinderAnnouncer
}

func NewManager(issuer creds.CredentialsIssuer, finder k8s.PodFinderAnnouncer) *CredentialManager {
	return &CredentialManager{issuer: issuer, finder: finder}
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
		logger.WithFields(creds.CredentialsFields(issued, role)).Infof("warming credentials")
	}
}

func (m *CredentialManager) fetchCredentialsForRole(role string) (*creds.Credentials, error) {
	return m.issuer.CredentialsForRole(role)
}

func (m *CredentialManager) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case pod := <-m.finder.Pods():
			log.WithFields(k8s.PodFields(pod)).Debugf("pod announced")
			m.fetchCredentials(pod)
		case expiring := <-m.issuer.Expiring():
			m.handleExpiring(expiring)
		}
	}
}

func (m *CredentialManager) handleExpiring(credentials *creds.RoleCredentials) {
	logger := log.WithFields(creds.CredentialsFields(credentials.Credentials, credentials.Role))

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
	m.fetchCredentialsForRole(credentials.Role)
}

func (m *CredentialManager) IsRoleActive(role string) (bool, error) {
	return m.finder.IsActivePodsForRole(role)
}
