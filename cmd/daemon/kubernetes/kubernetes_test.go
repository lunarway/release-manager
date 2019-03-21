package kubernetes

import (
	"testing"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func Test(t *testing.T) {
	log.Init()
	testCases := []struct {
		desc          string
		input         watch.Event
		successOutput *PodEvent
		failureOutput *PodEvent
	}{
		{
			desc: "Pod Running",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "product-77d79cf64-59mjj",
						Annotations: map[string]string{
							"lunarway.com/controlled-by-release-manager": "true",
						},
					},
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
					},
				},
			},
			successOutput: &PodEvent{
				PodName: "product-77d79cf64-59mjj",
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			var successCalled, failedCalled bool
			statusNotifier(tC.input, func(e *PodEvent) error {
				successCalled = true
				assert.Equal(t, tC.successOutput, e, "success callback not as expected")
				return nil
			}, func(e *PodEvent) error {
				failedCalled = true
				assert.Equal(t, tC.failureOutput, e, "failure callback not as expected")
				return nil
			})
			if tC.successOutput != nil {
				assert.True(t, successCalled, "expected success callback to be called")
			}
			if tC.failureOutput != nil {
				assert.True(t, failedCalled, "expected fail callback to be called")
			}
		})
	}
}
