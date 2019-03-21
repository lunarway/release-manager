package kubernetes

import (
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Client struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

func NewClient() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	client := Client{
		clientset: clientset,
		config:    config,
	}

	return &client, nil
}

func (c *Client) WatchPods() error {
	watcher, err := c.clientset.CoreV1().Pods("").Watch(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for {
		e := <-watcher.ResultChan()
		if e.Object == nil {
			log.Errorf("Object not found: %v", e.Object)
			return errors.New("object not found")
		}

		pod, ok := e.Object.(*v1.Pod)
		if !ok {
			continue
		}

		log.WithFields("action", e.Type,
			"namespace", pod.Namespace,
			"name", pod.Name,
			"phase", pod.Status.Phase,
			"reason", pod.Status.Reason,
			"container#", len(pod.Status.ContainerStatuses)).Infof("Event received: Type=%s, Pod=%s", e.Type, pod.Name)
	}

	watcher.Stop()
	return nil
}
