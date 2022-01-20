package kubernetes

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseToJSONLogs(t *testing.T) {
	testCases := []struct {
		desc   string
		input  string
		output []ContainerLog
		err    error
	}{
		{
			desc:  "2 log lines",
			input: "{\"level\":\"info\",\"message\":\"[ACTOR] server receives *actor.Terminated\"}\n{\"level\":\"error\",\"message\":\"Got unexpected termination from component. Will stop application\"}\n",
			output: []ContainerLog{
				{
					Level:   "info",
					Message: "[ACTOR] server receives *actor.Terminated",
				},
				{
					Level:   "error",
					Message: "Got unexpected termination from component. Will stop application",
				},
			},
		},
		{
			desc:   "non json",
			input:  "PANIC IN MAIN\nWith a stack",
			output: nil,
			err:    errors.New("invalid character 'P' looking for beginning of value"),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			logs, err := parseToJSONAray(tC.input)
			if tC.err != nil {
				assert.EqualError(t, errors.Cause(err), tC.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "no output error expected")
			}
			assert.Equal(t, tC.output, logs, "output logs not as expected")
		})
	}
}

func TestIsPodControlledByJob(t *testing.T) {
	testCases := []struct {
		desc     string
		pod      *corev1.Pod
		expected bool
	}{
		{
			desc:     "no owner reference",
			pod:      &corev1.Pod{},
			expected: false,
		},
		{
			desc: "owner reference is not job",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "not-job",
						},
					},
				},
			},
			expected: false,
		},
		{
			desc: "owner reference is job",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "Job",
						},
					},
				},
			},
			expected: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actual := isPodControlledByJob(tc.pod)

			assert.Equal(t, tc.expected, actual, "output not as expected")
		})
	}
}

func TestIsPodInCreateContainerConfigError(t *testing.T) {
	testCases := []struct {
		desc     string
		pod      *corev1.Pod
		expected bool
	}{
		{
			desc:     "no container status",
			pod:      &corev1.Pod{},
			expected: false,
		},
		{
			desc: "container status is not createConfigError",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			desc: "container status is CreateContainerConfigError",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason:  "CreateContainerConfigError",
									Message: "Missing secret",
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			desc: "container status is CreateContainerConfigError with failed to sync configmap cache message ",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason:  "CreateContainerConfigError",
									Message: "failed to sync configmap cache: timed out waiting for the condition",
								},
							},
						},
					},
				},
			},
			expected: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actual := isPodInCreateContainerConfigError(tc.pod)

			assert.Equal(t, tc.expected, actual, "output not as expected")
		})
	}
}
