package kubernetes

import (
	"context"
	"fmt"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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

func (c *Client) WatchPods(ctx context.Context, succeeded, failed NotifyFunc) error {
	// TODO; See if it's possible to use FieldSelector to limit events received.
	watcher, err := c.clientset.CoreV1().Pods("").Watch(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for {
		select {
		case e := <-watcher.ResultChan():
			if e.Object == nil {
				log.Errorf("Object not found: %v", e.Object)
				return errors.New("object not found")
			}
			statusNotifier(e, succeeded, failed)

		case <-ctx.Done():
			watcher.Stop()
			return ctx.Err()
		}
	}
	return nil
}

func statusNotifier(e watch.Event, succeeded, failed NotifyFunc) {
	pod, ok := e.Object.(*v1.Pod)
	if !ok {
		return
	}

	// Just continue if this pod is not controlled by the release manager
	if !(pod.Annotations["lunarway.com/controlled-by-release-manager"] == "true") {
		return
	}

	log.Infof("Event received: %s, pod=%s", e.Type, pod.Name)
	switch e.Type {
	case watch.Modified:
		log.Infof("In modified state: pod=%s, pod.Status.Phase=%s", pod.Name, pod.Status.Phase)

		switch pod.Status.Phase {

		// PodRunning means the pod has been bound to a node and all of the containers have been started.
		// At least one container is still running or is in the process of being restarted.
		case v1.PodRunning:
			log.WithFields("pod", fmt.Sprintf("%v", pod)).Infof("PodRunning: pod=%s, reason=%s, message=%s", pod.Name, pod.Status.Reason, pod.Status.Message)
			succeeded(&PodEvent{
				Namespace: pod.Namespace,
				PodName:   pod.Name,
				Reason:    "",
				Message:   "",
			})

		// PodFailed means that all containers in the pod have terminated, and at least one container has
		// terminated in a failure (exited with a non-zero exit code or was stopped by the system).
		case v1.PodFailed:
			log.WithFields("pod", fmt.Sprintf("%v", pod)).Infof("PodFailed: pod=%s, reason=%s, message=%s", pod.Name, pod.Status.Reason, pod.Status.Message)
			failed(&PodEvent{
				Namespace: pod.Namespace,
				PodName:   pod.Name,
				Reason:    pod.Status.Reason,
				Message:   pod.Status.Message,
			})

		// PodPending means the pod has been accepted by the system, but one or more of the containers
		// has not been started. This includes time before being bound to a node, as well as time spent
		// pulling images onto the host.
		case v1.PodPending:
			log.WithFields("pod", fmt.Sprintf("%v", pod)).Infof("PodPending: pod=%s, reason=%s, message=%s", pod.Name, pod.Status.Reason, pod.Status.Message)
			for _, cst := range pod.Status.ContainerStatuses {
				if cst.State.Waiting.Reason == "CreateContainerConfigError" {
					failed(&PodEvent{
						Namespace: pod.Namespace,
						PodName:   pod.Name,
						Reason:    cst.State.Waiting.Reason,
						Message:   cst.State.Waiting.Message,
					})
				}

			}

		// PodUnknown means that for some reason the state of the pod could not be obtained, typically due
		// to an error in communicating with the host of the pod.
		case v1.PodUnknown:
			log.WithFields("pod", fmt.Sprintf("%v", pod)).Infof("PodUnknown: pod=%s, reason=%s, message=%s", pod.Name, pod.Status.Reason, pod.Status.Message)

		// PodSucceeded means that all containers in the pod have voluntarily terminated
		// with a container exit code of 0, and the system is not going to restart any of these containers.
		case v1.PodSucceeded:
			log.WithFields("pod", fmt.Sprintf("%v", pod)).Infof("PodSucceeded: pod=%s, reason=%s, message=%s", pod.Name, pod.Status.Reason, pod.Status.Message)
		}
	}
}
