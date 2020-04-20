package kubernetes

import (
	"context"

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
			ok = isDeploymentSuccessful(deploy)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}

			// In-order to minimize messages and only return events when new releases is detected, we add
			// a new annotation to the Deployment. This annotations tells provides the daemon with some valuable
			// information about the artifacts running.
			// When we initially apply a Deployment the lunarway.com/artifact-id annotations SHOULD be set.
			// Further the observed-artifact-id is an annotation managed by the daemon and will initially be "".
			// In this state we annotate the Deployment with the current artifact-id as the observed.
			// When we update a Deployment we also update the artifact-id, e.g. now observed and actual artifact id
			// is different. In this case we want to notify, and update the observed with the current artifact id.
			// This also eliminates messages when a pod is deleted. As the two annotations will be equal.
			if deploy.Annotations["lunarway.com/observed-artifact-id"] == deploy.Annotations["lunarway.com/artifact-id"] {
				continue
			}

			// Annotate the Deployment to be able to skip it next time
			err = annotateDeployment(ctx, c.clientset, deploy)
			if err != nil {
				log.Errorf("Unable to annotate Deployment: %v", err)
				continue
			}

			// Notify the release-manager with the successful deployment event.
			err = c.exporter.SendSuccessfulReleaseEvent(ctx, http.ReleaseEvent{
				Name:          deploy.Name,
				Namespace:     deploy.Namespace,
				ResourceType:  "Deployment",
				ArtifactID:    deploy.Annotations["lunarway.com/artifact-id"],
				AuthorEmail:   deploy.Annotations["lunarway.com/author"],
				AvailablePods: deploy.Status.AvailableReplicas,
				DesiredPods:   *deploy.Spec.Replicas,
			})
		}
	}
}

func isDeploymentSuccessful(d *appsv1.Deployment) bool {
	return deploymentutil.DeploymentComplete(d, &d.Status)
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

// Annotates the DaemonSet with the observed-artifact-id. This is used to differentiate new releases from pod deletion events.
func annotateDeployment(ctx context.Context, c *kubernetes.Clientset, d *appsv1.Deployment) error {
	d.Annotations["lunarway.com/observed-artifact-id"] = d.Annotations["lunarway.com/artifact-id"]
	_, err := c.AppsV1().Deployments(d.Namespace).Update(ctx, d, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
