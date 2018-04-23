package k8s

import (
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	// ResourcePods are Pod resources
	ResourcePods = "pods"
	// ResourceNamespaces are Namespace resources
	ResourceNamespaces = "namespaces"
)

// NewListWatch creates a ListWatch for the specified Resource
func NewListWatch(client *kubernetes.Clientset, resource string) *cache.ListWatch {
	return cache.NewListWatchFromClient(client.Core().RESTClient(), resource, "", fields.Everything())
}
