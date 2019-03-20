package kubernetes

import (
	"fmt"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type K8s struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

func NewClient() (*K8s, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return &K8s{}, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return &K8s{}, err
	}

	return &K8s{clientset: clientset, config: config}, nil
}

func (c *K8s) WatchPods() error {
	watcher, err := c.clientset.CoreV1().Pods("").Watch(metav1.ListOptions{})
	if err != nil {
		errors.WithMessage(err, "cannot create pod watcher")
	}

	for {
		e := <-watcher.ResultChan()
		if e.Object == nil {
			return errors.WithMessage(err, "cannot read object")
		}
		p, ok := e.Object.(*v1.Pod)
		if !ok {
			continue
		}

		fmt.Printf("POD: %v", p)
	}
	watcher.Stop()
	return nil
}
