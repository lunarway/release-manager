package kubernetes

import (
	"context"
	"time"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	deploymentutil "k8s.io/kubernetes/pkg/controller/deployment/util"
)

func (c *Client) HandleNewDeployments(ctx context.Context) error {
	watcher, err := c.clientset.AppsV1().Deployments("").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			watcher.Stop()
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
			event, ok, err := isDeploymentSuccessful(c.clientset, c.replicaSetTimeDiff, deploy)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}

			// Notify the release-manager with the successful deployment event.
			err = c.exporter.SendSuccessfulDeploymentEvent(ctx, event)
			if err != nil {
				log.Errorf("Failed to send successful deployment event: %v", err)
				continue
			}
		}
	}
}

func isDeploymentSuccessful(c *kubernetes.Clientset, replicaSetTimeDiff time.Duration, deployment *appsv1.Deployment) (http.DeploymentEvent, bool, error) {
	if !deploymentutil.DeploymentComplete(deployment, &deployment.Status) {
		return http.DeploymentEvent{}, false, nil
	}
	cond := deploymentutil.GetDeploymentCondition(deployment.Status, appsv1.DeploymentProgressing)
	if cond != nil {
		if cond.Reason == "ProgressDeadlineExceeded" {
			log.Errorf("deployment %s exceeded its progress deadline", deployment.Name)
			// TODO: Maybe return a specific error here
			return http.DeploymentEvent{}, false, nil
		}
		// It seems that this reduce unwanted messages when the release-daemon starts.
		if cond.LastUpdateTime == cond.LastTransitionTime {
			return http.DeploymentEvent{}, false, nil
		}
	}

	// Retrieve the new ReplicaSet to determince whether or not this is a new deployment event or not.
	newRs, err := deploymentutil.GetNewReplicaSet(deployment, c.AppsV1())
	if err != nil {
		return http.DeploymentEvent{}, false, nil
	}
	//We discard events if the difference between creation time of the ReplicaSet and now is greater than replicaSetTimeDiff
	diff := time.Since(newRs.CreationTimestamp.Time)
	if diff > replicaSetTimeDiff {
		return http.DeploymentEvent{}, false, nil
	}
	return http.DeploymentEvent{
		Name:          deployment.Name,
		Namespace:     deployment.Namespace,
		ArtifactID:    deployment.Annotations["lunarway.com/artifact-id"],
		AuthorEmail:   deployment.Annotations["lunarway.com/author"],
		AvailablePods: deployment.Status.AvailableReplicas,
		Replicas:      *deployment.Spec.Replicas,
	}, true, nil
}

func isDeploymentCorrectlyAnnotated(deploy *appsv1.Deployment) bool {
	// Just continue if this pod is not controlled by the release manager
	if !(deploy.Annotations["lunarway.com/controlled-by-release-manager"] == "true") {
		return false
	}

	// Just discard the event if there's no artifact id
	if deploy.Annotations["lunarway.com/artifact-id"] == "" {
		log.Errorf("artifact-id missing in deployment: namespace '%s' name '%s'", deploy.Namespace, deploy.Name)
		return false
	}

	if deploy.Annotations["lunarway.com/author"] == "" {
		log.Errorf("author missing in deployment: namespace '%s' name '%s'", deploy.Namespace, deploy.Name)
		return false
	}
	return true
}

// Avoid reporting on pods that has been marked for termination
func isDeploymentMarkedForTermination(deploy *appsv1.Deployment) bool {
	return deploy.DeletionTimestamp != nil
}
