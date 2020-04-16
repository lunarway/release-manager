package kubernetes

import (
	"context"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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

			ok = isDaemonSetSuccessful(ds)
			if !ok {
				continue
			}

			// Notify the release-manager with the successful deployment event.
			err = c.exporter.SendSuccessfulDeploymentEvent(ctx, http.DeploymentEvent{
				Name:          ds.Name,
				Namespace:     ds.Namespace,
				ArtifactID:    ds.Annotations["lunarway.com/artifact-id"],
				AuthorEmail:   ds.Annotations["lunarway.com/author"],
				AvailablePods: ds.Status.NumberAvailable,
				Replicas:      ds.Status.DesiredNumberScheduled,
			})
			if err != nil {
				log.Errorf("Failed to send successful deployment event: %v", err)
				continue
			}
		}
	}
}

func isDaemonSetSuccessful(ds *appsv1.DaemonSet) bool {
	return ds.Status.DesiredNumberScheduled != 0 && ds.Status.DesiredNumberScheduled == ds.Status.CurrentNumberScheduled && ds.Status.NumberReady == ds.Status.UpdatedNumberScheduled && ds.Status.UpdatedNumberScheduled == ds.Status.NumberAvailable && ds.Status.NumberUnavailable == 0 && ds.Status.NumberMisscheduled == 0
}

func isDaemonSetCorrectlyAnnotated(ds *appsv1.DaemonSet) bool {
	// Just continue if this pod is not controlled by the release manager
	if !(ds.Annotations["lunarway.com/controlled-by-release-manager"] == "true") {
		return false
	}

	// Just discard the event if there's no artifact id
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
