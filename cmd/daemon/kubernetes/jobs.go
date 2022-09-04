package kubernetes

import (
	"context"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type JobInformer struct {
	clientset *kubernetes.Clientset
	exporter  Exporter
	logger    *log.Logger
}

func RegisterJobInformer(logger *log.Logger, informerFactory informers.SharedInformerFactory, exporter Exporter, handlerFactory ResourceEventHandlerFactory, clientset *kubernetes.Clientset) {
	j := JobInformer{
		clientset: clientset,
		exporter:  exporter,
		logger:    logger,
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

	if isObserved(job.Annotations) {
		return
	}

	ctx := context.Background()

	if isJobFailed(job) {
		// Notify the release-manager with the job error event.
		err := j.exporter.SendJobErrorEvent(ctx, http.JobErrorEvent{
			JobName:     job.Name,
			Namespace:   job.Namespace,
			Errors:      jobErrorMessages(job),
			ArtifactID:  job.Annotations[artifactIDAnnotationKey],
			AuthorEmail: job.Annotations[authorAnnotationKey],
		})
		if err != nil {
			j.logger.Errorf("Failed to send job error event: %v", err)
			return
		}
		// Annotate the Deployment to be able to skip it next time
		err = annotateJob(ctx, j.clientset, job)
		if err != nil {
			j.logger.Errorf("Unable to annotate Job: %v", err)
			return
		}
	}
}

// Annotates the Job with the observed-artifact-id.
func annotateJob(ctx context.Context, c *kubernetes.Clientset, j *batchv1.Job) error {
	observe(j.Annotations)
	_, err := c.BatchV1().Jobs(j.Namespace).Update(ctx, j, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
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
