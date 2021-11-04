package kubernetes

import (
	"context"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	deploymentutil "k8s.io/kubernetes/pkg/controller/deployment/util"
)

type DeploymentInformer struct {
	clientset *kubernetes.Clientset
	exporter  Exporter
}

func RegisterDeploymentInformer(informerFactory informers.SharedInformerFactory, clientset *kubernetes.Clientset, exporter Exporter, handlerFactory ResourceEventHandlerFactory) {
	d := DeploymentInformer{
		clientset: clientset,
		exporter:  exporter,
	}

	informerFactory.
		Apps().
		V1().
		Deployments().
		Informer().
		AddEventHandler(handlerFactory(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				d.handleDeployment(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				d.handleDeployment(newObj)
			},
		}))
}

func (d *DeploymentInformer) handleDeployment(e interface{}) {
	deploy, ok := e.(*appsv1.Deployment)
	if !ok {
		return
	}

	ctx := context.Background()

	// Check if we have all the annotations we need for the release-daemon
	if !isCorrectlyAnnotated(deploy.Annotations) {
		return
	}

	// Avoid reporting on pods that has been marked for termination
	if isDeploymentMarkedForTermination(deploy) {
		return
	}

	// Check the received event, and determine whether or not this was a successful deployment.
	if !isDeploymentSuccessful(deploy) {
		return
	}

	// In-order to minimize messages and only return events when new releases is detected, we add
	// a new annotation to the Deployment.
	// When we initially apply a Deployment the lunarway.com/artifact-id annotations SHOULD be set.
	// Further the observed-artifact-id is an annotation managed by the daemon and will initially be "".
	// In this state we annotate the Deployment with the current artifact-id as the observed.
	// When we update a Deployment we also update the artifact-id, e.g. now observed and actual artifact id
	// is different. In this case we want to notify, and update the observed with the current artifact id.
	// This also eliminates messages when a pod is deleted. As the two annotations will be equal.
	if deploy.Annotations["lunarway.com/observed-artifact-id"] == deploy.Annotations["lunarway.com/artifact-id"] {
		return
	}

	// Notify the release-manager with the successful deployment event.
	err := d.exporter.SendSuccessfulReleaseEvent(ctx, http.ReleaseEvent{
		Name:          deploy.Name,
		Namespace:     deploy.Namespace,
		ResourceType:  "Deployment",
		ArtifactID:    deploy.Annotations["lunarway.com/artifact-id"],
		AuthorEmail:   deploy.Annotations["lunarway.com/author"],
		AvailablePods: deploy.Status.AvailableReplicas,
		DesiredPods:   *deploy.Spec.Replicas,
	})
	if err != nil {
		log.Errorf("Failed to send successful deployment event: %v", err)
		return
	}

	// Annotate the Deployment to be able to skip it next time
	err = annotateDeployment(ctx, d.clientset, deploy)
	if err != nil {
		log.Errorf("Unable to annotate Deployment: %v", err)
		return
	}
}

func isDeploymentSuccessful(d *appsv1.Deployment) bool {
	return deploymentutil.DeploymentComplete(d, &d.Status)
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
