package kubernetes

import (
	"context"
	"time"

	httpinternal "github.com/lunarway/release-manager/internal/http"
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
			event, ok, err := isDeploymentSuccessful(c.clientset, deploy)
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
		}
	}
	return nil
}

func isDeploymentSuccessful(c *kubernetes.Clientset, deployment *appsv1.Deployment) (httpinternal.DeploymentEvent, bool, error) {
	if deployment.Generation <= deployment.Status.ObservedGeneration {
		cond := deploymentutil.GetDeploymentCondition(deployment.Status, appsv1.DeploymentProgressing)
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

		newRs, err := deploymentutil.GetNewReplicaSet(deployment, c.AppsV1())
		if err != nil {
			return httpinternal.DeploymentEvent{}, false, nil
		}
		log.Infof("New Replicaset: %s, CreationTime: %s", newRs.Name, newRs.CreationTimestamp)
		//We discard events if the difference between creation and now is greater than 10s
		diff := time.Now().Sub(newRs.CreationTimestamp.Time)
		if diff > (time.Duration(10) * time.Second) {
			log.Infof("Timediff is greater than 10s and we should discard the event")
			return httpinternal.DeploymentEvent{}, false, nil
		}
		log.Infof("Revision: %s, RevisionHistory: %s", deployment.Annotations["deployment.kubernetes.io/revision"], deployment.Annotations["deployment.kubernetes.io/revision-history"])

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
