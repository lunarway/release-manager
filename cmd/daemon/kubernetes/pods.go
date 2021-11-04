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
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type PodInformer struct {
	clientset              *kubernetes.Clientset
	exporter               Exporter
	moduloCrashReportNotif float64
}

func RegisterPodInformer(informerFactory informers.SharedInformerFactory, exporter Exporter, handlerFactory ResourceEventHandlerFactory, moduloCrashReportNotif float64) {
	p := PodInformer{
		exporter:               exporter,
		moduloCrashReportNotif: moduloCrashReportNotif,
	}

	informerFactory.
		Core().
		V1().
		Pods().
		Informer().
		AddEventHandler(handlerFactory(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				p.handle(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				p.handle(newObj)
			},
		}))
}

func (p *PodInformer) handle(e interface{}) {
	pod, ok := e.(*corev1.Pod)
	if !ok {
		return
	}
	// Is this a pod managed by the release manager - and does it contain the information needed?
	if !isCorrectlyAnnotated(pod.Annotations) {
		return
	}
	// Avoid reporting on pods that has been marked for termination
	if isPodMarkedForTermination(pod) {
		return
	}

	ctx := context.Background()

	if isPodInCrashLoopBackOff(pod) {
		log.Infof("Pod: %s is in CrashLoopBackOff", pod.Name)
		restartCount := pod.Status.ContainerStatuses[0].RestartCount
		if math.Mod(float64(restartCount), p.moduloCrashReportNotif) != 1 {
			return
		}
		var errorContainers []http.ContainerError
		for _, cst := range pod.Status.ContainerStatuses {
			if cst.State.Waiting != nil && cst.State.Waiting.Reason == "CrashLoopBackOff" {

				logs, err := getLogs(ctx, p.clientset, pod.Name, cst.Name, pod.Namespace)
				if err != nil {
					log.Errorf("Error retrieving logs from pod: %s, container: %s, namespace: %s: %v", pod.Name, cst.Name, pod.Namespace, err)
				}

				errorContainers = append(errorContainers, http.ContainerError{
					Name:         cst.Name,
					ErrorMessage: logs,
					Type:         "CrashLoopBackOff",
				})
			}
		}

		err := p.exporter.SendPodErrorEvent(ctx, http.PodErrorEvent{
			PodName:     pod.Name,
			Namespace:   pod.Namespace,
			Errors:      errorContainers,
			ArtifactID:  pod.Annotations["lunarway.com/artifact-id"],
			AuthorEmail: pod.Annotations["lunarway.com/author"],
		})
		if err != nil {
			log.Errorf("Failed to send crash loop backoff event: %v", err)
		}
		return
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

		err := p.exporter.SendPodErrorEvent(ctx, http.PodErrorEvent{
			PodName:     pod.Name,
			Namespace:   pod.Namespace,
			Errors:      errorContainers,
			ArtifactID:  pod.Annotations["lunarway.com/artifact-id"],
			AuthorEmail: pod.Annotations["lunarway.com/author"],
		})
		if err != nil {
			log.Errorf("Failed to send create container config error: %v", err)
		}
	}
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
