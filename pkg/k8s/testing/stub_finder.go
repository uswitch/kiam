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
package testing

import (
	"context"
	"github.com/uswitch/kiam/pkg/k8s"
	"k8s.io/api/core/v1"
)

func NewStubFinder(pod *v1.Pod) *StubFinder {
	return &StubFinder{pod: pod}
}

type StubFinder struct {
	pod *v1.Pod
}

func (f *StubFinder) GetPodByIP(ip string) (*v1.Pod, error) {
	if f.pod == nil {
		return nil, k8s.ErrPodNotFound
	}

	return f.pod, nil
}

type stubAnnouncer struct {
	pods chan *v1.Pod
}

func NewStubAnnouncer() *stubAnnouncer {
	return &stubAnnouncer{pods: make(chan *v1.Pod)}
}

func (f *stubAnnouncer) Announce(pod *v1.Pod) {
	f.pods <- pod
}

func (f *stubAnnouncer) Pods() <-chan *v1.Pod {
	return f.pods
}

func (f *stubAnnouncer) IsActivePodsForRole(role string) (bool, error) {
	return true, nil
}

type stubNSFinder struct {
	n *v1.Namespace
}

func NewNamespaceFinder(n *v1.Namespace) *stubNSFinder {
	return &stubNSFinder{
		n: n,
	}
}

func (f *stubNSFinder) FindNamespace(ctx context.Context, name string) (*v1.Namespace, error) {
	return f.n, nil
}
