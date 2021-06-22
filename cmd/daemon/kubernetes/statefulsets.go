package kubernetes

import (
	"context"

	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/prometheus/common/log"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

func (c *Client) HandleNewStatefulSets(ctx context.Context) error {
	watcher, err := c.clientset.AppsV1().StatefulSets("").Watch(ctx, metav1.ListOptions{})
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
			ss, ok := e.Object.(*appsv1.StatefulSet)
			if !ok {
				continue
			}

			// Check if we have all the annotations we need for the release-daemon
			if !isCorrectlyAnnotated(ss.Annotations) {
				continue
			}

			// Avoid reporting on pods that has been marked for termination
			if isStefulSetMarkedForTermination(ss) {
				continue
			}

			// Verify if the StatefulSet fulfills the criterias for a succesful release
			if !isStatefulSetSuccessful(ss) {
				continue
			}

			// In-order to minimize messages and only return events when new releases is detected, we add
			// a new annotation to the StatefulSet.
			// When we initially apply a StatefulSet the lunarway.com/artifact-id annotations SHOULD be set.
			// Further the observed-artifact-id is an annotation managed by the daemon and will initially be "".
			// In this state we annotate the StatefulSet with the current artifact-id as the observed.
			// When we update a StatefulSet we also update the artifact-id, e.g. now observed and actual artifact id
			// is different. In this case we want to notify, and update the observed with the current artifact id.
			// This also eliminates messages when a pod is deleted. As the two annotations will be equal.
			if ss.Annotations["lunarway.com/observed-artifact-id"] == ss.Annotations["lunarway.com/artifact-id"] {
				continue
			}

			// Notify the release-manager with the successful deployment event.
			err = c.exporter.SendSuccessfulReleaseEvent(ctx, models.DaemonKubernetesDeploymentWebhookRequest{
				Name:          ss.Name,
				Namespace:     ss.Namespace,
				ResourceType:  "StatefulSet",
				ArtifactID:    ss.Annotations["lunarway.com/artifact-id"],
				AuthorEmail:   ss.Annotations["lunarway.com/author"],
				AvailablePods: int64(ss.Status.ReadyReplicas),
				DesiredPods:   int64(ss.Status.Replicas),
			})
			if err != nil {
				log.Errorf("Failed to send successful statefulset event: %v", err)
				continue
			}

			// Annotate the StatefulSet to be able to skip it next time
			err = annotateStatefulSet(ctx, c.clientset, ss)
			if err != nil {
				log.Errorf("Unable to annotate StatefulSet: %v", err)
				continue
			}
		}
	}
}

// Avoid reporting on StatefulSets that has been marked for termination
func isStefulSetMarkedForTermination(ss *appsv1.StatefulSet) bool {
	return ss.DeletionTimestamp != nil
}

func isStatefulSetSuccessful(ss *appsv1.StatefulSet) bool {
	return ss.Status.Replicas == ss.Status.ReadyReplicas &&
		ss.Status.Replicas == ss.Status.CurrentReplicas &&
		ss.Status.Replicas == ss.Status.UpdatedReplicas &&
		ss.Status.ObservedGeneration >= ss.Generation
}

func annotateStatefulSet(ctx context.Context, c *kubernetes.Clientset, ss *appsv1.StatefulSet) error {
	ss.Annotations["lunarway.com/observed-artifact-id"] = ss.Annotations["lunarway.com/artifact-id"]
	_, err := c.AppsV1().StatefulSets(ss.Namespace).Update(ctx, ss, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
