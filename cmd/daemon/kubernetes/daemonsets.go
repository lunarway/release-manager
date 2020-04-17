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
)

func (c *Client) HandleNewDaemonSets(ctx context.Context) error {
	watcher, err := c.clientset.AppsV1().DaemonSets("").Watch(ctx, metav1.ListOptions{})
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
			ds, ok := e.Object.(*appsv1.DaemonSet)
			if !ok {
				continue
			}

			if !isDaemonSetCorrectlyAnnotated(ds) {
				continue
			}

			// Avoid reporting on pods that has been marked for termination
			if isDaemonSetMarkedForTermination(ds) {
				continue
			}

			log.Infof("Observed: %d, Gen: %d", ds.Status.ObservedGeneration, ds.Generation)

			// Verify if the DaemonSet fulfills the criterias for a succesful release
			ok = isDaemonSetSuccessful(ds)
			if !ok {
				continue
			}

			// Check if the DaemonSet is marked with the lunarway.com/released annotation
			if isDaemonSetMarkedAsReleased(ds) {
				continue
			}

			// Annotate the DaemonSet to be able to skip it next time
			err = annotateDaemonSet(ctx, c.clientset, ds)
			if err != nil {
				log.Errorf("Unable to annotate DaemonSet: %v", err)
				continue
			}

			// Notify the release-manager with the successful deployment event.
			err = c.exporter.SendSuccessfulReleaseEvent(ctx, http.ReleaseEvent{
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
				continue
			}
		}
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

func isDaemonSetCorrectlyAnnotated(ds *appsv1.DaemonSet) bool {
	if !(ds.Annotations["lunarway.com/controlled-by-release-manager"] == "true") {
		return false
	}
	if ds.Annotations["lunarway.com/artifact-id"] == "" {
		log.Errorf("artifact-id missing in deployment: namespace '%s' name '%s'", ds.Namespace, ds.Name)
		return false
	}
	if ds.Annotations["lunarway.com/author"] == "" {
		log.Errorf("author missing in deployment: namespace '%s' name '%s'", ds.Namespace, ds.Name)
		return false
	}
	return true
}

// Avoid reporting on pods that has been marked for termination
func isDaemonSetMarkedForTermination(ds *appsv1.DaemonSet) bool {
	return ds.DeletionTimestamp != nil
}

func isDaemonSetMarkedAsReleased(ds *appsv1.DaemonSet) bool {
	if ds.Annotations["lunarway.com/released"] == "" {
		return false
	}
	return true
}

func annotateDaemonSet(ctx context.Context, c *kubernetes.Clientset, ds *appsv1.DaemonSet) error {
	timestamp := time.Now().String()
	ds.Annotations["lunarway.com/released"] = timestamp
	_, err := c.AppsV1().DaemonSets(ds.Namespace).Update(ctx, ds, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	log.Infof("DaemonSet: %s annotated with released timestamp: %s", ds.Name, timestamp)
	return nil
}
