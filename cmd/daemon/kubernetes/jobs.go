package kubernetes

import (
	"context"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type JobInformer struct {
	exporter Exporter
}

func RegisterJobInformer(informerFactory informers.SharedInformerFactory, exporter Exporter, handlerFactory ResourceEventHandlerFactory) {
	j := JobInformer{
		exporter: exporter,
	}

	informerFactory.
		Batch().
		V1().
		Jobs().
		Informer().
		AddEventHandler(handlerFactory(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				j.handle(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				j.handle(newObj)
			},
		}))
}

func (j JobInformer) handle(e interface{}) {
	job, ok := e.(*batchv1.Job)
	if !ok {
		return
	}

	// Check if we have all the annotations we need for the release-daemon
	if !isCorrectlyAnnotated(job.Annotations) {
		return
	}

	ctx := context.Background()

	if isJobFailed(job) {
		// Notify the release-manager with the job error event.
		err := j.exporter.SendJobErrorEvent(ctx, http.JobErrorEvent{
			JobName:     job.Name,
			Namespace:   job.Namespace,
			Errors:      jobErrorMessages(job),
			ArtifactID:  job.Annotations["lunarway.com/artifact-id"],
			AuthorEmail: job.Annotations["lunarway.com/author"],
		})
		if err != nil {
			log.Errorf("Failed to send job error event: %v", err)
			return
		}
	}
}

func isJobFailed(job *batchv1.Job) bool {
	if len(job.Status.Conditions) == 0 {
		return false
	}
	for _, condition := range job.Status.Conditions {
		if condition.Status == corev1.ConditionTrue && condition.Type == batchv1.JobFailed {
			return true
		}
	}
	return false
}

func jobErrorMessages(job *batchv1.Job) []http.JobConditionError {
	var errors []http.JobConditionError

	for _, condition := range job.Status.Conditions {
		if condition.Status == corev1.ConditionTrue && condition.Type == batchv1.JobFailed {
			errors = append(errors, http.JobConditionError{
				Reason:  condition.Reason,
				Message: condition.Message,
			})
		}
	}
	return errors
}
