package official

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient returns an in-cluster or out of cluster clientset depending on
// whether kubecfg is set or empty.
func NewClient(kubecfg string) (*kubernetes.Clientset, error) {
	if kubecfg != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubecfg)
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
