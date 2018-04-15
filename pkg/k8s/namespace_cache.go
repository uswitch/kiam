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
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	// AnnotationPermittedKey hold the name of the annotation for the regex expressing the
	// roles that can be assumed by pods in that namespace.
	AnnotationPermittedKey = "iam.amazonaws.com/permitted"
)

// NamespaceCache implements NamespaceFinder interface used to determine which roles
// can be assumed by pods
type NamespaceCache struct {
	indexer    cache.Indexer
	controller cache.Controller
}

// NewNamespaceCache creates the cache storing Namespaces
func NewNamespaceCache(source cache.ListerWatcher, syncInterval time.Duration) *NamespaceCache {
	namespaceLogger := &namespaceLogger{}
	indexer, controller := cache.NewIndexerInformer(source, &v1.Namespace{}, syncInterval, namespaceLogger, cache.Indexers{})
	return &NamespaceCache{
		indexer:    indexer,
		controller: controller,
	}
}

// Run starts the cache processing updates. Blocks until cache has synced
func (c *NamespaceCache) Run(ctx context.Context) error {
	go c.controller.Run(ctx.Done())
	log.Infof("started namespace cache controller")

	ok := cache.WaitForCacheSync(ctx.Done(), c.controller.HasSynced)
	if !ok {
		return ErrWaitingForSync
	}

	return nil
}

// FindNamespace finds the Namespace by it's name
func (c *NamespaceCache) FindNamespace(ctx context.Context, name string) (*v1.Namespace, error) {
	obj, exists, err := c.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return obj.(*v1.Namespace), nil
}

type namespaceLogger struct {
}

func (o *namespaceLogger) OnAdd(obj interface{}) {
	namespace, isNamespace := obj.(*v1.Namespace)
	if !isNamespace {
		log.Errorf("OnAdd unexpected object: %+v", obj)
		return
	}
	log.WithFields(namespaceFields(namespace)).Debugf("added namespace")
}

func (o *namespaceLogger) OnDelete(obj interface{}) {
	namespace, isNamespace := obj.(*v1.Namespace)
	if !isNamespace {
		deletedObj, isDeleted := obj.(cache.DeletedFinalStateUnknown)
		if !isDeleted {
			log.Errorf("OnDelete unexpected object: %+v", obj)
			return
		}

		namespace, isNamespace = deletedObj.Obj.(*v1.Namespace)
		if !isNamespace {
			log.Errorf("OnDelete unexpected DeletedFinalStateUnknown object: %+v", deletedObj.Obj)
		}
		log.WithFields(namespaceFields(namespace)).Debugf("deleted namespace")
		return
	}

	log.WithFields(namespaceFields(namespace)).Debugf("deleted namespace")
	return
}

func (o *namespaceLogger) OnUpdate(old, new interface{}) {
	namespace, isNamespace := new.(*v1.Namespace)
	if !isNamespace {
		log.Errorf("OnUpdate unexpected object: %+v", new)
		return
	}

	log.WithFields(namespaceFields(namespace)).Debugf("updated namespace")
}
