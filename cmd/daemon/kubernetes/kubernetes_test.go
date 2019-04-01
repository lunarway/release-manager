package kubernetes

import (
	"testing"
	"time"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestStatusNotifier(t *testing.T) {
	log.Init()
	testCases := []struct {
		desc          string
		input         watch.Event
		successOutput *PodEvent
		failureOutput *PodEvent
	}{
		{
			desc: "Pod with 2 containers in State: Running and NOT Ready",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
						ContainerStatuses: []v1.ContainerStatus{
							{
								Name:  "container1",
								Ready: false,
								State: v1.ContainerState{
									Running: &v1.ContainerStateRunning{
										StartedAt: metav1.Time{
											Time: time.Now(),
										},
									},
								},
							},
							{
								Name:  "container2",
								Ready: false,
								State: v1.ContainerState{
									Running: &v1.ContainerStateRunning{
										StartedAt: metav1.Time{
											Time: time.Now(),
										},
									},
								},
							},
						},
					},
				},
			},
			successOutput: &PodEvent{
				Name:       "product-77d79cf64-59mjj",
				Namespace:  "dev",
				State:      "Running",
				ArtifactID: "master-7039119b9c-6a95af9e3f",
				Containers: []http.Container{
					{
						Name:  "container1",
						State: "Running",
						Ready: false,
					},
					{
						Name:  "container2",
						State: "Running",
						Ready: false,
					},
				},
				Reason:  "",
				Message: "",
			},
			failureOutput: nil,
		},
		{
			desc: "Pod with 2 containers in State: Running and Ready",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
						ContainerStatuses: []v1.ContainerStatus{
							{
								Name:  "container1",
								Ready: true,
								State: v1.ContainerState{
									Running: &v1.ContainerStateRunning{
										StartedAt: metav1.Time{
											Time: time.Now(),
										},
									},
								},
							},
							{
								Name:  "container2",
								Ready: true,
								State: v1.ContainerState{
									Running: &v1.ContainerStateRunning{
										StartedAt: metav1.Time{
											Time: time.Now(),
										},
									},
								},
							},
						},
					},
				},
			},
			successOutput: &PodEvent{
				Name:       "product-77d79cf64-59mjj",
				Namespace:  "dev",
				State:      "Running",
				ArtifactID: "master-7039119b9c-6a95af9e3f",
				Containers: []http.Container{
					{
						Name:  "container1",
						State: "Ready",
						Ready: true,
					},
					{
						Name:  "container2",
						State: "Ready",
						Ready: true,
					},
				},
				Reason:  "",
				Message: "",
			},
			failureOutput: nil,
		},
		{
			desc: "Pod with 1 container in State: Running",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
						ContainerStatuses: []v1.ContainerStatus{
							{
								Name:  "container1",
								Ready: false,
								State: v1.ContainerState{
									Running: &v1.ContainerStateRunning{
										StartedAt: metav1.Time{
											Time: time.Now(),
										},
									},
								},
							},
						},
					},
				},
			},
			successOutput: &PodEvent{
				Name:       "product-77d79cf64-59mjj",
				Namespace:  "dev",
				State:      "Running",
				ArtifactID: "master-7039119b9c-6a95af9e3f",
				Containers: []http.Container{
					{
						Name:  "container1",
						State: "Running",
					},
				},
				Reason:  "",
				Message: "",
			},
			failureOutput: nil,
		},
		{
			desc: "Pod in State: Running with CrashLoopBackOff",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
						ContainerStatuses: []v1.ContainerStatus{
							{
								Name: "crash",
								State: v1.ContainerState{
									Waiting: &v1.ContainerStateWaiting{
										Message: "there should be something here",
										Reason:  "CrashLoopBackOff",
									},
								},
							},
						},
					},
				},
			},
			successOutput: nil,
			failureOutput: &PodEvent{
				Name:      "product-77d79cf64-59mjj",
				Namespace: "dev",
				State:     "Running",
				Containers: []http.Container{
					{
						Name:    "crash",
						State:   "CrashLoopBackOff",
						Reason:  "CrashLoopBackOff",
						Message: "there should be something here",
					},
				},
				ArtifactID: "master-7039119b9c-6a95af9e3f",
				Reason:     "CrashLoopBackOff",
				Message:    "crash: there should be something here",
			},
		},
		{
			desc: "Pod with 2 containers in State: Running with 1 container in CrashLoopBackOff and 1 container Running",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
						ContainerStatuses: []v1.ContainerStatus{
							{
								Name: "crash",
								State: v1.ContainerState{
									Waiting: &v1.ContainerStateWaiting{
										Message: "there should be something here",
										Reason:  "CrashLoopBackOff",
									},
								},
							},
							{
								Name: "container2",
								State: v1.ContainerState{
									Running: &v1.ContainerStateRunning{
										StartedAt: metav1.Time{
											Time: time.Now(),
										},
									},
								},
							},
						},
					},
				},
			},
			successOutput: nil,
			failureOutput: &PodEvent{
				Name:      "product-77d79cf64-59mjj",
				Namespace: "dev",
				State:     string(v1.PodRunning),
				Containers: []http.Container{
					{
						Name:    "crash",
						State:   "CrashLoopBackOff",
						Reason:  "CrashLoopBackOff",
						Message: "there should be something here",
					},
					{
						Name:  "container2",
						State: "Running",
					},
				},
				ArtifactID: "master-7039119b9c-6a95af9e3f",
				Reason:     "CrashLoopBackOff",
				Message:    "crash: there should be something here",
			},
		},
		{
			desc: "Pod with 2 containers in State: Running with both containers in CrashLoopBackOff",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
						ContainerStatuses: []v1.ContainerStatus{
							{
								Name: "crash",
								State: v1.ContainerState{
									Waiting: &v1.ContainerStateWaiting{
										Message: "there should be something here",
										Reason:  "CrashLoopBackOff",
									},
								},
							},
							{
								Name: "crash1",
								State: v1.ContainerState{
									Waiting: &v1.ContainerStateWaiting{
										Message: "there should be something here as well",
										Reason:  "CrashLoopBackOff",
									},
								},
							},
						},
					},
				},
			},
			successOutput: nil,
			failureOutput: &PodEvent{
				Name:      "product-77d79cf64-59mjj",
				Namespace: "dev",
				State:     string(v1.PodRunning),
				Containers: []http.Container{
					{
						Name:    "crash",
						State:   "CrashLoopBackOff",
						Reason:  "CrashLoopBackOff",
						Message: "there should be something here",
					},
					{
						Name:    "crash1",
						State:   "CrashLoopBackOff",
						Reason:  "CrashLoopBackOff",
						Message: "there should be something here as well",
					},
				},
				ArtifactID: "master-7039119b9c-6a95af9e3f",
				Reason:     "CrashLoopBackOff",
				Message:    "crash: there should be something here, crash1: there should be something here as well",
			},
		},
		{
			desc: "Pod in State: Running with State.Waiting nil",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
						ContainerStatuses: []v1.ContainerStatus{
							{
								State: v1.ContainerState{
									Waiting: nil,
								},
							},
						},
					},
				},
			},
			successOutput: nil,
			failureOutput: nil,
		},
		{
			desc: "Pod in State: Waiting with CreateContainerConfigError",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase: v1.PodPending,
						ContainerStatuses: []v1.ContainerStatus{
							{
								Name: "error",
								State: v1.ContainerState{
									Waiting: &v1.ContainerStateWaiting{
										Message: "some config did not match",
										Reason:  "CreateContainerConfigError",
									},
								},
							},
						},
					},
				},
			},
			successOutput: nil,
			failureOutput: &PodEvent{
				Name:      "product-77d79cf64-59mjj",
				Namespace: "dev",
				State:     string(v1.PodPending),
				Containers: []http.Container{
					{
						Name:    "error",
						State:   "CreateContainerConfigError",
						Reason:  "CreateContainerConfigError",
						Message: "some config did not match",
					},
				},
				ArtifactID: "master-7039119b9c-6a95af9e3f",
				Reason:     "CreateContainerConfigError",
				Message:    "error: some config did not match",
			},
		},
		{
			desc: "Pod with 2 containers in State: Waiting with 1 container in CreateContainerConfigError",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase: v1.PodPending,
						ContainerStatuses: []v1.ContainerStatus{
							{
								Name: "error",
								State: v1.ContainerState{
									Waiting: &v1.ContainerStateWaiting{
										Message: "some config did not match",
										Reason:  "CreateContainerConfigError",
									},
								},
							},
							{
								Name: "running",
								State: v1.ContainerState{
									Running: &v1.ContainerStateRunning{
										StartedAt: metav1.Time{
											Time: time.Now(),
										},
									},
								},
							},
						},
					},
				},
			},
			successOutput: nil,
			failureOutput: &PodEvent{
				Name:      "product-77d79cf64-59mjj",
				Namespace: "dev",
				State:     string(v1.PodPending),
				Containers: []http.Container{
					{
						Name:    "error",
						State:   "CreateContainerConfigError",
						Reason:  "CreateContainerConfigError",
						Message: "some config did not match",
					},
					{
						Name:  "running",
						State: "Running",
					},
				},
				ArtifactID: "master-7039119b9c-6a95af9e3f",
				Reason:     "CreateContainerConfigError",
				Message:    "error: some config did not match",
			},
		},
		{
			desc: "Pod in State: Waiting with State.Waiting=nil",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase: v1.PodPending,
						ContainerStatuses: []v1.ContainerStatus{
							{
								State: v1.ContainerState{
									Waiting: nil,
								},
							},
						},
					},
				},
			},
			successOutput: nil,
			failureOutput: nil,
		},
		{
			desc: "Pod in State: Failed",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase:   v1.PodFailed,
						Message: "Some message",
						Reason:  "Some reason",
					},
				},
			},
			successOutput: nil,
			failureOutput: &PodEvent{
				Name:       "product-77d79cf64-59mjj",
				Namespace:  "dev",
				State:      "Failed",
				ArtifactID: "master-7039119b9c-6a95af9e3f",
				Reason:     "Some reason",
				Message:    "Some message",
			},
		},
		{
			desc: "Pod in State: Unknown",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase: v1.PodUnknown,
					},
				},
			},
			successOutput: nil,
			failureOutput: nil,
		},
		{
			desc: "Pod in State: Succeeded",
			input: watch.Event{
				Type: watch.Modified,
				Object: &v1.Pod{
					ObjectMeta: defaultObjectMetaData(),
					Status: v1.PodStatus{
						Phase: v1.PodSucceeded,
					},
				},
			},
			successOutput: nil,
			failureOutput: nil,
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
			} else {
				assert.False(t, successCalled, "expected success callback NOT to be called")
			}
			if tC.failureOutput != nil {
				assert.True(t, failedCalled, "expected failure callback to be called")
			} else {
				assert.False(t, failedCalled, "expected failure callback NOT to be called")
			}
		})
	}
}

func defaultObjectMetaData() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      "product-77d79cf64-59mjj",
		Namespace: "dev",
		Annotations: map[string]string{
			"lunarway.com/controlled-by-release-manager": "true",
			"lunarway.com/artifact-id":                   "master-7039119b9c-6a95af9e3f",
		},
	}
}
