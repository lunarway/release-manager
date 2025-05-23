package kubernetes

import (
	"context"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type StatefulSetInformer struct {
	clientset kubernetes.Interface
	exporter  Exporter
}

func RegisterStatefulSetInformer(informerFactory informers.SharedInformerFactory, exporter Exporter, handlerFactory ResourceEventHandlerFactory, clientset *kubernetes.Clientset) {
	s := StatefulSetInformer{
		clientset: clientset,
		exporter:  exporter,
	}

	informerFactory.
		Apps().
		V1().
		StatefulSets().
		Informer().
		AddEventHandler(handlerFactory(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				s.handle(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				s.handle(newObj)
			},
		}))
}

func (s *StatefulSetInformer) handle(e interface{}) {
	ss, ok := e.(*appsv1.StatefulSet)
	if !ok {
		return
	}

	// Check if we have all the annotations we need for the release-daemon
	if !isCorrectlyAnnotated(ss.Annotations) {
		return
	}

	// Avoid reporting on pods that has been marked for termination
	if isStefulSetMarkedForTermination(ss) {
		return
	}

	// Verify if the StatefulSet fulfills the criterias for a succesful release
	if !isStatefulSetSuccessful(ss) {
		return
	}

	// In-order to minimize messages and only return events when new releases is detected, we add
	// a new annotation to the StatefulSet.
	// When we initially apply a StatefulSet the lunarway.com/artifact-id annotations SHOULD be set.
	// Further the observed-artifact-id is an annotation managed by the daemon and will initially be "".
	// In this state we annotate the StatefulSet with the current artifact-id as the observed.
	// When we update a StatefulSet we also update the artifact-id, e.g. now observed and actual artifact id
	// is different. In this case we want to notify, and update the observed with the current artifact id.
	// This also eliminates messages when a pod is deleted. As the two annotations will be equal.
	if isObserved(ss.Annotations) {
		return
	}

	ctx := context.Background()

	// Notify the release-manager with the successful deployment event.
	err := s.exporter.SendSuccessfulReleaseEvent(ctx, http.ReleaseEvent{
		Name:          ss.Name,
		Namespace:     ss.Namespace,
		ResourceType:  "StatefulSet",
		ArtifactID:    ss.Annotations[artifactIDAnnotationKey],
		AuthorEmail:   ss.Annotations[authorAnnotationKey],
		AvailablePods: ss.Status.ReadyReplicas,
		DesiredPods:   ss.Status.Replicas,
	})
	if err != nil {
		log.Errorf("Failed to send successful statefulset event: %v", err)
		return
	}

	// Annotate the StatefulSet to be able to skip it next time
	err = annotateStatefulSet(ctx, s.clientset, ss)
	if err != nil {
		log.Errorf("Unable to annotate StatefulSet: %v", err)
		return
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

func annotateStatefulSet(ctx context.Context, c kubernetes.Interface, ss *appsv1.StatefulSet) error {
	var err error
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		// Get the latest version of the StatefulSet
		latest, err := c.AppsV1().StatefulSets(ss.Namespace).Get(ctx, ss.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// Apply our annotation changes to the latest version
		if latest.Annotations == nil {
			latest.Annotations = make(map[string]string)
		}
		// Copy the artifact ID from the original StatefulSet if it's not already set
		if latest.Annotations[artifactIDAnnotationKey] == "" {
			latest.Annotations[artifactIDAnnotationKey] = ss.Annotations[artifactIDAnnotationKey]
		}
		observe(latest.Annotations)

		// Update with the latest version
		_, err = c.AppsV1().StatefulSets(ss.Namespace).Update(ctx, latest, metav1.UpdateOptions{})
		if err == nil {
			return nil
		}
		// If it's not a conflict error, return immediately
		if !k8serrors.IsConflict(err) {
			return err
		}
	}
	return err
}
