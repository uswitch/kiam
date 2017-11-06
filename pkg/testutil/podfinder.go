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
package testutil

import (
	"context"
	"github.com/uswitch/kiam/pkg/k8s"
	"github.com/uswitch/kiam/pkg/server"
	"k8s.io/api/core/v1"
)

type FailingFinder struct {
	Pod               *v1.Pod
	SucceedAfterCalls int
	calls             int
}

func (f *FailingFinder) FindPodForIP(ip string) (*v1.Pod, error) {
	if f.calls != f.SucceedAfterCalls {
		f.calls = f.calls + 1
		return nil, server.ErrPodNotFound
	}

	return f.Pod, nil
}

func (f *FailingFinder) FindRoleFromIP(ctx context.Context, ip string) (string, error) {
	pod, err := f.FindPodForIP(ip)
	if err != nil {
		return "", err
	}

	if pod == nil {
		return "", nil
	}

	return k8s.PodRole(pod), nil
}

func NewStubFinder(pod *v1.Pod) *stubFinder {
	return &stubFinder{pod: pod}
}

type stubFinder struct {
	pod *v1.Pod
}

func (f *stubFinder) FindPodForIP(ip string) (*v1.Pod, error) {
	if f.pod == nil {
		return nil, server.ErrPodNotFound
	}

	return f.pod, nil
}

func (f *stubFinder) FindRoleFromIP(ctx context.Context, ip string) (string, error) {
	pod, err := f.FindPodForIP(ip)
	if err != nil {
		return "", err
	}

	return k8s.PodRole(pod), nil
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
