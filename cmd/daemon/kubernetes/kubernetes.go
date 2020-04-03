package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	deploymentutil "k8s.io/kubectl/pkg/util/deployment"
	podutil "k8s.io/kubectl/pkg/util/podutils"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset *kubernetes.Clientset
}

var (
	ErrWatcherClosed = errors.New("channel closed")
)

func NewClient(kubeConfigPath string) (*Client, error) {
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
		clientset: clientset,
	}, nil
}

// func (c *Client) GetLogs(podName, namespace string) (string, error) {
// 	numberOfLogLines := int64(8)
// 	podLogOpts := v1.PodLogOptions{
// 		TailLines: &numberOfLogLines,
// 	}
// 	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)

// 	podLogs, err := req.Stream()
// 	if err != nil {
// 		return "", err
// 	}
// 	defer podLogs.Close()

// 	buf := new(bytes.Buffer)
// 	_, err = io.Copy(buf, podLogs)
// 	if err != nil {
// 		return "", err
// 	}
// 	logs, err := parseToJSONAray(buf.String())
// 	if err != nil {
// 		return buf.String(), nil
// 	}
// 	message := ""
// 	for _, l := range logs {
// 		message += l.Message + "\n"
// 	}
// 	return message, nil
// }

func parseToJSONAray(str string) ([]Log, error) {
	str = strings.ReplaceAll(str, "}\n{", "},{")
	str = fmt.Sprintf("[%s]", str)

	var logs []Log
	err := json.Unmarshal([]byte(str), &logs)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal")
	}
	return logs, nil
}

func (c *Client) WatchDeployments(ctx context.Context, succeeded, failed NotifyFunc) error {
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
			detectSuccessfulDeployment(c.clientset, e, succeeded, failed)

		case <-ctx.Done():
			watcher.Stop()
			return ctx.Err()
		}
	}
}

func (c *Client) WatchPods(ctx context.Context, succeeded, failed NotifyFunc) error {
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
			detectPodErrors(c.clientset, e, succeeded, failed)

		case <-ctx.Done():
			watcher.Stop()
			return ctx.Err()
		}
	}
}

func detectPodErrors(kubectl *kubernetes.Clientset, e watch.Event, succeeded, failed NotifyFunc) {
	pod, ok := e.Object.(*corev1.Pod)
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

	if podutil.IsPodReady(pod) {
		return
	}

	if isPodInCrashLoopBackOff(pod) {
		log.Infof("Pod: %s is in CrashLoopBackOff", pod.Name)
		// TODO:
		// 1. Identify the pod or pods that is CrashLooping
		// 2. Extract the logs of the pod(s) that are in CrashLoop
		// 3. Generate event
		return
	}

	if isPodInCreateContainerConfigError(pod) {
		log.Infof("Pod: %s is in CreateContainerConfigError", pod.Name)
		// TODO:
		// 1. Identify the pod or pods that is CrashLooping
		// 2. Extract the message from kubernetes
		// 3. Generate event
		return
	}

	// TODO
	// Identify other possible errors
	//
	return
}

func detectSuccessfulDeployment(kubectl *kubernetes.Clientset, e watch.Event, succeeded, failed NotifyFunc) {
	deployment, ok := e.Object.(*appsv1.Deployment)
	if !ok {
		return
	}
	// Just continue if this pod is not controlled by the release manager
	if !(deployment.Annotations["lunarway.com/controlled-by-release-manager"] == "true") {
		return
	}

	// Just discard the event if there's no artifact id
	if deployment.Annotations["lunarway.com/artifact-id"] == "" {
		log.Errorf("artifact-id missing in deployment: namespace '%s' pod '%s'", deployment.Namespace, deployment.Name)
		return
	}

	if deployment.Generation <= deployment.Status.ObservedGeneration {
		cond := getDeploymentCondition(deployment.Status, appsv1.DeploymentProgressing)
		if cond != nil && cond.Reason == "ProgressDeadlineExceeded" {
			log.Errorf("deployment %q exceeded its progress deadline", deployment.Name)
			return
		}
		// Waiting for deployment %q rollout to finish: %d out of %d new replicas have been updated...
		if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
			return
		}
		// Waiting for deployment %q rollout to finish: %d old replicas are pending termination...
		if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
			return
		}
		// Waiting for deployment %q rollout to finish: %d of %d updated replicas are available...
		if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
			return
		}
		log.Infof("Deployment: %q successfully rolled out", deployment.Name)

		_, _, newRS, err := deploymentutil.GetAllReplicaSets(deployment, kubectl.AppsV1())
		if err != nil {
			log.Infof("Error: %v", err)
		}

		log.Infof("New ReplicaSet detected: %s", newRS.Name)

		return
	}
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
