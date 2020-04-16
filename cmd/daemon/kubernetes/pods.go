package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (c *Client) HandlePodErrors(ctx context.Context) error {
	// TODO; See if it's possible to use FieldSelector to limit events received.
	watcher, err := c.clientset.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for {
		select {
		case e, ok := <-watcher.ResultChan():
			if !ok {
				return ErrWatcherClosed
			}
			if e.Object == nil {
				continue
			}
			pod, ok := e.Object.(*corev1.Pod)
			if !ok {
				continue
			}
			// Is this a pod managed by the release manager - and does it contain the information needed?
			if !isPodCorrectlyAnnotated(pod) {
				continue
			}
			// Avoid reporting on pods that has been marked for termination
			if isPodMarkedForTermination(pod) {
				continue
			}

			if isPodInCrashLoopBackOff(pod) {
				log.Infof("Pod: %s is in CrashLoopBackOff", pod.Name)
				restartCount := pod.Status.ContainerStatuses[0].RestartCount
				if math.Mod(float64(restartCount), c.moduloCrashReportNotif) != 1 {
					continue
				}
				var errorContainers []http.ContainerError
				for _, cst := range pod.Status.ContainerStatuses {
					if cst.State.Waiting != nil && cst.State.Waiting.Reason == "CrashLoopBackOff" {

						logs, err := getLogs(ctx, c.clientset, pod.Name, cst.Name, pod.Namespace)
						if err != nil {
							log.Errorf("Error retrieving logs from pod: %s, container: %s, namespace: %s", pod.Name, cst.Name, pod.Namespace)
						}

						errorContainers = append(errorContainers, http.ContainerError{
							Name:         cst.Name,
							ErrorMessage: logs,
							Type:         "CrashLoopBackOff",
						})
					}
				}

				err = c.exporter.SendPodErrorEvent(ctx, http.PodErrorEvent{
					PodName:     pod.Name,
					Namespace:   pod.Namespace,
					Errors:      errorContainers,
					ArtifactID:  pod.Annotations["lunarway.com/artifact-id"],
					AuthorEmail: pod.Annotations["lunarway.com/author"],
				})
				if err != nil {
					log.Errorf("error SendCrashLoopBackOffEvent")
				}
				continue
			}

			if isPodInCreateContainerConfigError(pod) {
				log.Infof("Pod: %s is in CreateContainerConfigError", pod.Name)
				// Determine which container of the deployment has CreateContainerConfigError
				var errorContainers []http.ContainerError
				for _, cst := range pod.Status.ContainerStatuses {
					if cst.State.Waiting != nil && cst.State.Waiting.Reason == "CreateContainerConfigError" {
						errorContainers = append(errorContainers, http.ContainerError{
							Name:         cst.Name,
							ErrorMessage: cst.State.Waiting.Message,
							Type:         "CreateContainerConfigError",
						})
					}
				}

				err = c.exporter.SendPodErrorEvent(ctx, http.PodErrorEvent{
					PodName:     pod.Name,
					Namespace:   pod.Namespace,
					Errors:      errorContainers,
					ArtifactID:  pod.Annotations["lunarway.com/artifact-id"],
					AuthorEmail: pod.Annotations["lunarway.com/author"],
				})
				if err != nil {
					log.Errorf("error SendCreateContainerConfigError")
				}
				continue
			}

		case <-ctx.Done():
			watcher.Stop()
			return ctx.Err()
		}
	}
}

func isPodCorrectlyAnnotated(pod *corev1.Pod) bool {
	// Just continue if this pod is not controlled by the release manager
	if !(pod.Annotations["lunarway.com/controlled-by-release-manager"] == "true") {
		return false
	}

	// Just discard the event if there's no artifact id
	if pod.Annotations["lunarway.com/artifact-id"] == "" {
		log.Errorf("artifact-id missing in pod template: namespace '%s' pod '%s'", pod.Namespace, pod.Name)
		return false
	}
	return true
}

// Avoid reporting on pods that has been marked for termination
func isPodMarkedForTermination(pod *corev1.Pod) bool {
	return pod.DeletionTimestamp != nil
}

// isPodInCrashLoopBackOff - checks the PodStatus and indicates whether the Pod is in CrashLoopBackOff
func isPodInCrashLoopBackOff(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, cst := range pod.Status.ContainerStatuses {
		if cst.State.Waiting != nil && cst.State.Waiting.Reason == "CrashLoopBackOff" {
			return true
		}
	}
	return false
}

// isPodInCreateContainerConfigError - check the PodStatus and indicates whether the Pod is in CreateContainerConfigError
func isPodInCreateContainerConfigError(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodPending {
		return false
	}
	for _, cst := range pod.Status.ContainerStatuses {
		if cst.State.Waiting != nil && cst.State.Waiting.Reason == "CreateContainerConfigError" {
			return true
		}
	}
	return false
}

func getLogs(ctx context.Context, client *kubernetes.Clientset, podName, containerName, namespace string) (string, error) {
	numberOfLogLines := int64(8)
	podLogOpts := corev1.PodLogOptions{
		TailLines: &numberOfLogLines,
		Container: containerName,
	}
	req := client.CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)

	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	logs, err := parseToJSONAray(buf.String())
	if err != nil {
		return buf.String(), nil
	}
	message := ""
	for _, l := range logs {
		message += l.Message + "\n"
	}
	return message, nil
}

type ContainerLog struct {
	Level   string
	Message string
}

func parseToJSONAray(str string) ([]ContainerLog, error) {
	str = strings.ReplaceAll(str, "}\n{", "},{")
	str = fmt.Sprintf("[%s]", str)

	var logs []ContainerLog
	err := json.Unmarshal([]byte(str), &logs)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal")
	}
	return logs, nil
}
