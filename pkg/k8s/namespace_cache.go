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
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"time"
)

const (
	// AnnotationName hold the name of the annotation for the regex expressing the
	// roles that can be assumed by pods in that namespace.
	AnnotationName = "iam.amazonaws.com/permitted"
)

// NamespaceCache implements NamespaceFinder interface used to determine which roles
// can be assumed by pods
type NamespaceCache struct {
	store      cache.Store
	controller cache.Controller
}

func namespaceFields(n *v1.Namespace) log.Fields {
	return log.Fields{
		"namespace":           n.Name,
		"namespace.permitted": n.GetAnnotations()[AnnotationName],
	}
}

func (c *NamespaceCache) process(obj interface{}) error {
	d := obj.(cache.Deltas).Newest()

	ns := d.Object.(*v1.Namespace)
	fields := log.Fields{
		"cache.object":     "namespace",
		"cache.delta.type": d.Type,
	}
	log.WithFields(fields).WithFields(namespaceFields(ns)).Debugf("processing delta")

	switch d.Type {
	case cache.Sync:
		return c.store.Add(d.Object)
	case cache.Added:
		return c.store.Add(d.Object)
	case cache.Updated:
		return c.store.Update(d.Object)
	case cache.Deleted:
		return c.store.Delete(d.Object)
	}
	return nil
}

func NewNamespaceCache(source cache.ListerWatcher, syncInterval time.Duration) *NamespaceCache {
	c := &NamespaceCache{}
	c.store = cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	config := &cache.Config{
		Queue:            cache.NewDeltaFIFO(cache.MetaNamespaceKeyFunc, nil, c.store),
		ListerWatcher:    source,
		ObjectType:       &v1.Namespace{},
		FullResyncPeriod: syncInterval,
		RetryOnError:     false,
		Process:          c.process,
	}
	c.controller = cache.New(config)
	return c
}

func (c *NamespaceCache) Run(ctx context.Context) {
	go c.controller.Run(ctx.Done())
	log.Infof("started namespace cache controller")
}

func (c *NamespaceCache) FindNamespace(ctx context.Context, name string) (*v1.Namespace, error) {
	obj, exists, err := c.store.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return obj.(*v1.Namespace), nil
}
