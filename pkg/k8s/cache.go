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
package k8s

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"time"
)

type PodFinderAnnouncer interface {
	// Finds a uncompleted pod from its IP address
	FindPodForIP(ip string) (*v1.Pod, error)
	// Return whether there are still uncompleted pods in the specified role
	IsActivePodsForRole(role string) (bool, error)
	// Will receive a Pod whenever there's a change/addition for a Pod with a role.
	Pods() <-chan *v1.Pod
}

type service struct {
	store           cache.Store
	cacheController cache.Controller
	stop            chan struct{}
	pods            chan *v1.Pod
}

func (s *service) Pods() <-chan *v1.Pod {
	return s.pods
}

func (s *service) announceRole(pod *v1.Pod) {
	s.pods <- pod
}

var MultipleRunningPodsErr = fmt.Errorf("multiple running pods found")

func IsPodCompleted(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed
}

func (s *service) IsActivePodsForRole(role string) (bool, error) {
	indexer, _ := s.store.(cache.Indexer)
	items, err := indexer.ByIndex(indexPodRole, role)
	if err != nil {
		return false, err
	}

	for _, obj := range items {
		pod, _ := obj.(*v1.Pod)

		if !IsPodCompleted(pod) {
			return true, nil
		}
	}

	return false, nil
}

func (s *service) FindPodForIP(ip string) (*v1.Pod, error) {
	found := make([]*v1.Pod, 0)

	indexer, _ := s.store.(cache.Indexer)
	items, err := indexer.ByIndex(indexPodIP, ip)
	if err != nil {
		return nil, err
	}

	for _, obj := range items {
		pod := obj.(*v1.Pod)

		if IsPodCompleted(pod) {
			continue
		}

		if pod.Status.PodIP == ip {
			found = append(found, pod)
		}
	}

	if len(found) == 0 {
		return nil, nil
	}

	if len(found) == 1 {
		return found[0], nil
	}

	return nil, MultipleRunningPodsErr
}

// handles objects from the queue processed by the cache
func (s *service) process(obj interface{}) error {
	deltas := obj.(cache.Deltas)

	for _, delta := range deltas {
		pod := delta.Object.(*v1.Pod)
		logger := log.WithFields(log.Fields{"cache.delta.type": delta.Type}).WithFields(PodFields(pod))

		role := PodRole(pod)
		if role != "" {
			logger.Debugf("announcing pod")
			s.announceRole(pod)
		}

		logger.Debugf("processing delta")
		switch delta.Type {
		case cache.Sync:
			s.store.Add(delta.Object)
		case cache.Added:
			s.store.Add(delta.Object)
		case cache.Updated:
			s.store.Update(delta.Object)
		case cache.Deleted:
			s.store.Delete(delta.Object)
		}
	}

	return nil
}

func KubernetesSource(client *kubernetes.Clientset) *cache.ListWatch {
	return cache.NewListWatchFromClient(client.Core().RESTClient(), "pods", "", fields.Everything())
}

const (
	indexPodIP   = "byIP"
	indexPodRole = "byRole"
)

func podIPIndex(obj interface{}) ([]string, error) {
	pod := obj.(*v1.Pod)

	if pod.Status.PodIP == "" {
		return []string{}, nil
	}

	return []string{pod.Status.PodIP}, nil
}

func podRoleIndex(obj interface{}) ([]string, error) {
	pod := obj.(*v1.Pod)
	role := PodRole(pod)
	if role == "" {
		return []string{}, nil
	}

	return []string{role}, nil
}

const (
	announceBufferSize = 100
)

func PodCache(source cache.ListerWatcher, syncInterval time.Duration) *service {
	service := &service{stop: make(chan struct{}), pods: make(chan *v1.Pod, announceBufferSize)}
	indexers := cache.Indexers{
		indexPodIP:   podIPIndex,
		indexPodRole: podRoleIndex,
	}
	store := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, indexers)
	service.store = store
	config := &cache.Config{
		Queue:            cache.NewDeltaFIFO(cache.MetaNamespaceKeyFunc, nil, service.store),
		ListerWatcher:    source,
		ObjectType:       &v1.Pod{},
		FullResyncPeriod: syncInterval,
		RetryOnError:     false,
		Process:          service.process,
	}
	service.cacheController = cache.New(config)
	return service
}

func (s *service) Run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		log.Infof("stopping cache controller")
		close(s.stop)
	}()

	go s.cacheController.Run(s.stop)
	log.Infof("started cache controller")
}

func PodRole(pod *v1.Pod) string {
	return pod.ObjectMeta.Annotations[IAMRoleKey]
}

const IAMRoleKey = "iam.amazonaws.com/role"
