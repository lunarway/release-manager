package command

import (
	"testing"

	"github.com/lunarway/release-manager/internal/spec"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	testCases := []struct {
		desc   string
		spec   spec.Spec
		stage  spec.Stage
		output spec.Spec
	}{
		{
			desc: "empty stages",
			spec: spec.Spec{
				Stages: nil,
			},
			stage: spec.Stage{
				ID: "test",
			},
			output: spec.Spec{
				Stages: []spec.Stage{
					{
						ID: "test",
					},
				},
			},
		},
		{
			desc: "stage already created",
			spec: spec.Spec{
				Stages: []spec.Stage{
					{
						ID: "test",
					},
				},
			},
			stage: spec.Stage{
				ID: "test",
			},
			output: spec.Spec{
				Stages: []spec.Stage{
					{
						ID: "test",
					},
				},
			},
		},
		{
			desc: "multiple stages present",
			spec: spec.Spec{
				Stages: []spec.Stage{
					{
						ID: "test",
					},
					{
						ID: "build",
					},
				},
			},
			stage: spec.Stage{
				ID:   "test",
				Name: "New",
			},
			output: spec.Spec{
				Stages: []spec.Stage{
					{
						ID:   "test",
						Name: "New",
					},
					{
						ID: "build",
					},
				},
			},
		},
		{
			desc: "another stage already present",
			spec: spec.Spec{
				Stages: []spec.Stage{
					{
						ID: "test",
					},
					{
						ID: "build",
					},
				},
			},
			stage: spec.Stage{
				ID: "push",
			},
			output: spec.Spec{
				Stages: []spec.Stage{
					{
						ID: "test",
					},
					{
						ID: "build",
					},
					{
						ID: "push",
					},
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			updated := setStage(tC.spec, tC.stage)
			assert.Equal(t, tC.output, updated)
		})
	}
}
