package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/lunarway/release-manager/internal/log"
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
func (c *Client) GetLogs(podName, namespace string) (string, error) {
	numberOfLogLines := int64(25)
	podLogOpts := v1.PodLogOptions{
		TailLines: &numberOfLogLines,
	}
	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)

	podLogs, err := req.Stream()
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	str := buf.String()

	return str, nil
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
				continue
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

	// Just discard the event if there's no artifact id
	if pod.Annotations["lunarway.com/artifact-id"] == "" {
		log.Errorf("artifact-id missing in deployment: namespace '%s' pod '%s'", pod.Namespace, pod.Name)
		return
	}

	artifactId := pod.Annotations["lunarway.com/artifact-id"]

	switch e.Type {
	case watch.Modified:
		log.WithFields("pod", pod).Infof("Pod object: %s", pod.Name)
		switch pod.Status.Phase {

		// PodRunning means the pod has been bound to a node and all of the containers have been started.
		// At least one container is still running or is in the process of being restarted.
		case v1.PodRunning:
			var containers []Container
			message := ""
			discard := false
			for i, cst := range pod.Status.ContainerStatuses {
				// Container is in Waiting state and CrashLoopBackOff
				if cst.State.Waiting != nil && cst.State.Waiting.Reason == "CrashLoopBackOff" {
					containers = append(containers, Container{Name: cst.Name, State: cst.State.Waiting.Reason, Reason: cst.State.Waiting.Reason, Message: cst.State.Waiting.Message, Ready: cst.Ready, RestartCount: cst.RestartCount})
					if i == 0 {
						message += cst.Name + ": " + cst.State.Waiting.Message
					} else {
						message += ", " + cst.Name + ": " + cst.State.Waiting.Message
					}

					continue
				}
				// Container is Running and responding to Readiness checks
				if cst.State.Running != nil && cst.Ready {
					containers = append(containers, Container{Name: cst.Name, State: "Ready", Reason: "", Message: "", Ready: cst.Ready, RestartCount: cst.RestartCount})
					continue
				}

				// Container is Running but the container is not responding to Readiness checks
				if cst.State.Running != nil && !cst.Ready {
					containers = append(containers, Container{Name: cst.Name, State: "Running", Reason: "", Message: "", Ready: cst.Ready, RestartCount: cst.RestartCount})
					continue
				}
				discard = true
			}

			if discard && len(containers) == 0 {
				return
			}

			if containsState(containers, "CrashLoopBackOff") {
				failed(&PodEvent{
					Namespace:  pod.Namespace,
					Name:       pod.Name,
					ArtifactID: artifactId,
					State:      "CrashLoopBackOff",
					Containers: containers,
					Reason:     "CrashLoopBackOff",
					Message:    message,
				})
				return
			} else if allContainersReady(containers) {
				succeeded(&PodEvent{
					Namespace:  pod.Namespace,
					Name:       pod.Name,
					ArtifactID: artifactId,
					State:      "Ready",
					Containers: containers,
					Reason:     "",
					Message:    message,
				})
				return
			}
			succeeded(&PodEvent{
				Namespace:  pod.Namespace,
				Name:       pod.Name,
				ArtifactID: artifactId,
				State:      string(v1.PodRunning),
				Containers: containers,
				Reason:     "",
				Message:    message,
			})
			return

			// PodFailed means that all containers in the pod have terminated, and at least one container has
			// terminated in a failure (exited with a non-zero exit code or was stopped by the system).
		case v1.PodFailed:
			failed(&PodEvent{
				Namespace:  pod.Namespace,
				Name:       pod.Name,
				ArtifactID: artifactId,
				State:      string(v1.PodFailed),
				Reason:     pod.Status.Reason,
				Message:    pod.Status.Message,
			})
			return

			// PodPending means the pod has been accepted by the system, but one or more of the containers
			// has not been started. This includes time before being bound to a node, as well as time spent
			// pulling images onto the host.
		case v1.PodPending:
			var containers []Container
			discard := false
			message := ""
			for i, cst := range pod.Status.ContainerStatuses {
				// Container is in Waiting state and CreateContainerConfigError
				if cst.State.Waiting != nil && cst.State.Waiting.Reason == "CreateContainerConfigError" {
					containers = append(containers, Container{Name: cst.Name, State: cst.State.Waiting.Reason, Reason: cst.State.Waiting.Reason, Message: cst.State.Waiting.Message, Ready: cst.Ready, RestartCount: cst.RestartCount})
					if i == 0 {
						message += cst.Name + ": " + cst.State.Waiting.Message
					} else {
						message += ", " + cst.Name + ": " + cst.State.Waiting.Message
					}
					continue
				}
				// Container is Running and responding to Readiness checks
				if cst.State.Running != nil && cst.Ready {
					containers = append(containers, Container{Name: cst.Name, State: "Ready", Reason: "", Message: "", Ready: cst.Ready, RestartCount: cst.RestartCount})
					continue
				}

				// Container is Running but the container is not responding to Readiness checks
				if cst.State.Running != nil && !cst.Ready {
					containers = append(containers, Container{Name: cst.Name, State: "Running", Reason: "", Message: "", Ready: cst.Ready, RestartCount: cst.RestartCount})
					continue
				}
				discard = true
			}

			if discard && len(containers) == 0 {
				return
			}

			if containsState(containers, "CreateContainerConfigError") {
				failed(&PodEvent{
					Namespace:  pod.Namespace,
					Name:       pod.Name,
					ArtifactID: artifactId,
					State:      "CreateContainerConfigError",
					Containers: containers,
					Reason:     "CreateContainerConfigError",
					Message:    message,
				})
				return
			}
			return

			// PodUnknown means that for some reason the state of the pod could not be obtained, typically due
			// to an error in communicating with the host of the pod.
		case v1.PodUnknown:
			log.WithFields("pod", fmt.Sprintf("%v", pod)).Infof("PodUnknown: pod=%s, reason=%s, message=%s", pod.Name, pod.Status.Reason, pod.Status.Message)
			return

			// PodSucceeded means that all containers in the pod have voluntarily terminate	// with a container exit code of 0, and the system is not going to restart any of these containers.
		case v1.PodSucceeded:
			log.WithFields("pod", fmt.Sprintf("%v", pod)).Infof("PodSucceeded: pod=%s, reason=%s, message=%s", pod.Name, pod.Status.Reason, pod.Status.Message)
			return

		default:
			log.WithFields("pod", fmt.Sprintf("%v", pod)).Infof("Default case: pod=%s, reason=%s, message=%s", pod.Name, pod.Status.Reason, pod.Status.Message)
			return
		}
	}
}

func containsState(c []Container, x string) bool {
	for _, n := range c {
		if x == n.State {
			return true
		}
	}
	return false
}

func allContainersReady(c []Container) bool {
	ready := 0
	for _, n := range c {
		if n.Ready {
			ready += 1
		}
	}
	return len(c) == ready
}
