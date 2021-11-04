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
)

type DaemonSetInformer struct {
	clientset *kubernetes.Clientset
	exporter  Exporter
}

func RegisterDaemonSetInformer(informerFactory informers.SharedInformerFactory, clientset *kubernetes.Clientset, exporter Exporter, handlerFactory ResourceEventHandlerFactory) {
	d := DaemonSetInformer{
		clientset: clientset,
		exporter:  exporter,
	}

	informerFactory.
		Apps().
		V1().
		DaemonSets().
		Informer().
		AddEventHandler(handlerFactory(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				d.handle(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				d.handle(newObj)
			},
		}))
}

func (d *DaemonSetInformer) handle(e interface{}) {
	ds, ok := e.(*appsv1.DaemonSet)
	if !ok {
		return
	}

	// Check if we have all the annotations we need for the release-daemon
	if !isCorrectlyAnnotated(ds.Annotations) {
		return
	}

	// Avoid reporting on pods that has been marked for termination
	if isDaemonSetMarkedForTermination(ds) {
		return
	}

	// Verify if the DaemonSet fulfills the criterias for a succesful release
	if !isDaemonSetSuccessful(ds) {
		return
	}

	// In-order to minimize messages and only return events when new releases is detected, we add
	// a new annotation to the DaemonSet.
	// When we initially apply a DaemonSet the lunarway.com/artifact-id annotations SHOULD be set.
	// Further the observed-artifact-id is an annotation managed by the daemon and will initially be "".
	// In this state we annotate the DaemonSet with the current artifact-id as the observed.
	// When we update a DaemonSet we also update the artifact-id, e.g. now observed and actual artifact id
	// is different. In this case we want to notify, and update the observed with the current artifact id.
	// This also eliminates messages when a pod is deleted. As the two annotations will be equal.
	if ds.Annotations["lunarway.com/observed-artifact-id"] == ds.Annotations["lunarway.com/artifact-id"] {
		return
	}

	ctx := context.Background()

	// Notify the release-manager with the successful deployment event.
	err := d.exporter.SendSuccessfulReleaseEvent(ctx, http.ReleaseEvent{
		Name:          ds.Name,
		Namespace:     ds.Namespace,
		ResourceType:  "DaemonSet",
		ArtifactID:    ds.Annotations["lunarway.com/artifact-id"],
		AuthorEmail:   ds.Annotations["lunarway.com/author"],
		AvailablePods: ds.Status.NumberAvailable,
		DesiredPods:   ds.Status.DesiredNumberScheduled,
	})
	if err != nil {
		log.Errorf("Failed to send successful daemonset event: %v", err)
		return
	}

	// Annotate the DaemonSet to be able to skip it next time
	err = annotateDaemonSet(ctx, d.clientset, ds)
	if err != nil {
		log.Errorf("Unable to annotate DaemonSet: %v", err)
		return
	}
}

func isDaemonSetSuccessful(ds *appsv1.DaemonSet) bool {
	return ds.Status.DesiredNumberScheduled != 0 &&
		ds.Status.DesiredNumberScheduled == ds.Status.CurrentNumberScheduled &&
		ds.Status.NumberReady == ds.Status.UpdatedNumberScheduled &&
		ds.Status.NumberAvailable == ds.Status.UpdatedNumberScheduled &&
		ds.Status.NumberUnavailable == 0 &&
		ds.Status.NumberMisscheduled == 0 &&
		ds.Status.ObservedGeneration >= ds.Generation
}

// Avoid reporting on daemon sets that has been marked for termination
func isDaemonSetMarkedForTermination(ds *appsv1.DaemonSet) bool {
	return ds.DeletionTimestamp != nil
}

// Annotates the DaemonSet with the observed-artifact-id. This is used to differentiate new releases from pod deletion events.
func annotateDaemonSet(ctx context.Context, c *kubernetes.Clientset, ds *appsv1.DaemonSet) error {
	ds.Annotations["lunarway.com/observed-artifact-id"] = ds.Annotations["lunarway.com/artifact-id"]
	_, err := c.AppsV1().DaemonSets(ds.Namespace).Update(ctx, ds, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
