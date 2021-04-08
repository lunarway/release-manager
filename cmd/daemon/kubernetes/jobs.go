package kubernetes

import (
	"context"
	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func (c *Client) HandleJobErrors(ctx context.Context) error {
	watcher, err := c.clientset.BatchV1().Jobs("").Watch(ctx, metav1.ListOptions{})
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
			job, ok := e.Object.(*batchv1.Job)
			if !ok {
				continue
			}

			// Check if we have all the annotations we need for the release-daemon
			if !isCorrectlyAnnotated(job.Annotations) {
				continue
			}

			if isJobFailed(job) {
				// Notify the release-manager with the job error event.
				err = c.exporter.SendJobErrorEvent(ctx, http.JobErrorEvent{
					JobName:     job.Name,
					Namespace:   job.Namespace,
					Errors:      jobErrorMessages(job),
					ArtifactID:  job.Annotations["lunarway.com/artifact-id"],
					AuthorEmail: job.Annotations["lunarway.com/author"],
				})
				if err != nil {
					log.Errorf("Failed to send job error event: %v", err)
					continue
				}
			}
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
			errors = append(errors, http.JobConditionError {
				Reason: condition.Reason,
				ErrorMessage: condition.Message,
			})
		}
	}
	return errors
}
