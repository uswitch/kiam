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
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"time"
)

type PodCache struct {
	store           cache.Store
	cacheController cache.Controller
	stop            chan struct{}
	pods            chan *v1.Pod
}

func (s *PodCache) Pods() <-chan *v1.Pod {
	return s.pods
}

var MultipleRunningPodsErr = fmt.Errorf("multiple running pods found")

func IsPodCompleted(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed
}

func (s *PodCache) IsActivePodsForRole(role string) (bool, error) {
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

var (
	ErrPodNotFound = fmt.Errorf("pod not found")
)

func (s *PodCache) FindPodForIP(ip string) (*v1.Pod, error) {
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

	for idx, pod := range found {
		log.WithFields(PodFields(pod)).Debugf("found %d/%d pods for ip %s", len(found), idx+1, ip)
	}

	if len(found) == 0 {
		return nil, ErrPodNotFound
	}

	if len(found) == 1 {
		return found[0], nil
	}

	return nil, MultipleRunningPodsErr
}

func (s *PodCache) FindRoleFromIP(ctx context.Context, ip string) (string, error) {
	pod, err := s.FindPodForIP(ip)
	if err != nil {
		return "", err
	}

	if pod == nil {
		return "", nil
	}

	return PodRole(pod), nil
}

func (s *PodCache) GetPodByIP(ctx context.Context, ip string) (*v1.Pod, error) {
	return s.FindPodForIP(ip)
}

// handles objects from the queue processed by the cache
func (s *PodCache) process(obj interface{}) error {
	deltas := obj.(cache.Deltas)
	deltaMeter := metrics.GetOrRegisterMeter("PodCache.processDelta", metrics.DefaultRegistry)

	for _, delta := range deltas {
		pod := delta.Object.(*v1.Pod)
		fields := log.Fields{
			"cache.delta.type": delta.Type,
			"cache.object":     "pod",
		}
		logger := log.WithFields(fields).WithFields(PodFields(pod))

		role := PodRole(pod)
		if role != "" {
			select {
			case s.pods <- pod:
				logger.Debugf("announced pod")
			default:
				metrics.GetOrRegisterMeter("PodCache.dropAnnounce", metrics.DefaultRegistry).Mark(1)
				logger.Warnf("pods announcement full, dropping")
			}
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

		deltaMeter.Mark(1)
	}

	return nil
}

const (
	ResourcePods       = "pods"
	ResourceNamespaces = "namespaces"
)

func KubernetesSource(client *kubernetes.Clientset, resource string) *cache.ListWatch {
	return cache.NewListWatchFromClient(client.Core().RESTClient(), resource, "", fields.Everything())
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

// Creates the cache object that uses a watcher to listen for Pod events. The cache indexes pods by their
// IP address so that Kiam can identify which role a Pod should assume. It periodically syncs the list of
// pods and can announce Pods. When announcing Pods via the channel it will drop events if the buffer
// is full- bufferSize determines how many.
func NewPodCache(source cache.ListerWatcher, syncInterval time.Duration, bufferSize int) *PodCache {
	PodCache := &PodCache{stop: make(chan struct{}), pods: make(chan *v1.Pod, bufferSize)}
	indexers := cache.Indexers{
		indexPodIP:   podIPIndex,
		indexPodRole: podRoleIndex,
	}
	store := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, indexers)
	PodCache.store = store
	config := &cache.Config{
		Queue:            cache.NewDeltaFIFO(cache.MetaNamespaceKeyFunc, nil, PodCache.store),
		ListerWatcher:    source,
		ObjectType:       &v1.Pod{},
		FullResyncPeriod: syncInterval,
		RetryOnError:     false,
		Process:          PodCache.process,
	}
	PodCache.cacheController = cache.New(config)
	return PodCache
}

func (s *PodCache) Run(ctx context.Context) {
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
