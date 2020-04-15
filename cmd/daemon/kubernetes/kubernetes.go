package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	//deploymentutil "k8s.io/kubectl/pkg/util/deployment"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset              *kubernetes.Clientset
	exporter               Exporter
	moduloCrashReportNotif float64
}

var (
	ErrWatcherClosed            = errors.New("channel closed")
	ErrProgressDeadlineExceeded = errors.New("progress deadline exceeded")
)

func NewClient(kubeConfigPath string, moduloCrashReportNotif float64, e Exporter) (*Client, error) {
	if kubeConfigPath != "" {
		// we run outside a cluster
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, err
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		return &Client{
			clientset: clientset,
		}, nil
	}

	// we run within a cluster
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		clientset:              clientset,
		exporter:               e,
		moduloCrashReportNotif: moduloCrashReportNotif,
	}, nil
}

func (c *Client) HandleNewDeployments(ctx context.Context) error {
	watcher, err := c.clientset.AppsV1().Deployments("").Watch(ctx, metav1.ListOptions{})
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
			if e.Type == watch.Deleted {
				continue
			}
			deploy, ok := e.Object.(*appsv1.Deployment)
			if !ok {
				continue
			}
			if !isDeploymentCorrectlyAnnotated(deploy) {
				continue
			}

			// Avoid reporting on pods that has been marked for termination
			if isDeploymentMarkedForTermination(deploy) {
				continue
			}

			// Check the received event, and determine whether or not this was a successful deployment.
			event, ok, err := isDeploymentSuccessful(deploy)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			log.Infof("Event type: %v", e.Type)

			// Notify the release-manager with the successful deployment event.
			err = c.exporter.SendSuccessfulDeploymentEvent(ctx, event)
			if err != nil {
				log.Errorf("error SendSuccessfulDeploymentEvent")
				continue
			}
		case <-ctx.Done():
			watcher.Stop()
			return ctx.Err()
		}
	}
	return nil
}

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
				var errorContainers []httpinternal.ContainerError
				for _, cst := range pod.Status.ContainerStatuses {
					if cst.State.Waiting != nil && cst.State.Waiting.Reason == "CrashLoopBackOff" {

						logs, err := getLogs(ctx, c.clientset, pod.Name, cst.Name, pod.Namespace)
						if err != nil {
							log.Errorf("Error retrieving logs from pod: %s, container: %s, namespace: %s", pod.Name, cst.Name, pod.Namespace)
						}

						errorContainers = append(errorContainers, httpinternal.ContainerError{
							Name:         cst.Name,
							ErrorMessage: logs,
							Type:         "CrashLoopBackOff",
						})
					}
				}

				err = c.exporter.SendPodErrorEvent(ctx, httpinternal.PodErrorEvent{
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
				var errorContainers []httpinternal.ContainerError
				for _, cst := range pod.Status.ContainerStatuses {
					if cst.State.Waiting != nil && cst.State.Waiting.Reason == "CreateContainerConfigError" {
						errorContainers = append(errorContainers, httpinternal.ContainerError{
							Name:         cst.Name,
							ErrorMessage: cst.State.Waiting.Message,
							Type:         "CreateContainerConfigError",
						})
					}
				}

				err = c.exporter.SendPodErrorEvent(ctx, httpinternal.PodErrorEvent{
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

func isDeploymentSuccessful(deployment *appsv1.Deployment) (httpinternal.DeploymentEvent, bool, error) {

	if deployment.Generation <= deployment.Status.ObservedGeneration {
		cond := getDeploymentCondition(deployment.Status, appsv1.DeploymentProgressing)
		if cond != nil && cond.Reason == "ProgressDeadlineExceeded" {
			log.Errorf("deployment %q exceeded its progress deadline", deployment.Name)
			// TODO: Maybe return a specific error here
			return httpinternal.DeploymentEvent{}, false, nil
		}
		//
		if cond.LastUpdateTime == cond.LastTransitionTime {
			return httpinternal.DeploymentEvent{}, false, nil
		}
		// Waiting for deployment %q rollout to finish: %d out of %d new replicas have been updated...
		if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
			return httpinternal.DeploymentEvent{}, false, nil
		}
		// Waiting for deployment %q rollout to finish: %d old replicas are pending termination...
		if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
			return httpinternal.DeploymentEvent{}, false, nil
		}
		// Waiting for deployment %q rollout to finish: %d of %d updated replicas are available...
		if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
			return httpinternal.DeploymentEvent{}, false, nil
		}
		return httpinternal.DeploymentEvent{
			Name:          deployment.Name,
			Namespace:     deployment.Namespace,
			ArtifactID:    deployment.Annotations["lunarway.com/artifact-id"],
			AuthorEmail:   deployment.Annotations["lunarway.com/author"],
			AvailablePods: deployment.Status.AvailableReplicas,
			Replicas:      *deployment.Spec.Replicas,
		}, true, nil
	}
	return httpinternal.DeploymentEvent{}, false, nil
}

func getDeploymentCondition(status appsv1.DeploymentStatus, condType appsv1.DeploymentConditionType) *appsv1.DeploymentCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

func isDeploymentCorrectlyAnnotated(deploy *appsv1.Deployment) bool {
	// Just continue if this pod is not controlled by the release manager
	if !(deploy.Annotations["lunarway.com/controlled-by-release-manager"] == "true") {
		return false
	}

	// Just discard the event if there's no artifact id
	if deploy.Annotations["lunarway.com/artifact-id"] == "" {
		log.Errorf("artifact-id missing in deployment: namespace '%s' pod '%s'", deploy.Namespace, deploy.Name)
		return false
	}

	if deploy.Annotations["lunarway.com/author"] == "" {
		log.Errorf("author missing in deployment: namespace '%s' pod '%s'", deploy.Namespace, deploy.Name)
		return false
	}
	return true
}

// Avoid reporting on pods that has been marked for termination
func isDeploymentMarkedForTermination(deploy *appsv1.Deployment) bool {
	if deploy.DeletionTimestamp != nil {
		return true
	}
	return false
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
	if pod.DeletionTimestamp != nil {
		return true
	}
	return false
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

type containerLog struct {
	Level   string
	Message string
}

func parseToJSONAray(str string) ([]containerLog, error) {
	str = strings.ReplaceAll(str, "}\n{", "},{")
	str = fmt.Sprintf("[%s]", str)

	var logs []containerLog
	err := json.Unmarshal([]byte(str), &logs)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal")
	}
	return logs, nil
}
