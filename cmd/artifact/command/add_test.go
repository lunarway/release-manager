package command

import (
	"testing"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	testCases := []struct {
		desc   string
		spec   artifact.Spec
		stage  artifact.Stage
		output artifact.Spec
	}{
		{
			desc: "empty stages",
			spec: artifact.Spec{
				Stages: nil,
			},
			stage: artifact.Stage{
				ID: "test",
			},
			output: artifact.Spec{
				Stages: []artifact.Stage{
					{
						ID: "test",
					},
				},
			},
		},
		{
			desc: "stage already created",
			spec: artifact.Spec{
				Stages: []artifact.Stage{
					{
						ID: "test",
					},
				},
			},
			stage: artifact.Stage{
				ID: "test",
			},
			output: artifact.Spec{
				Stages: []artifact.Stage{
					{
						ID: "test",
					},
				},
			},
		},
		{
			desc: "multiple stages present",
			spec: artifact.Spec{
				Stages: []artifact.Stage{
					{
						ID: "test",
					},
					{
						ID: "build",
					},
				},
			},
			stage: artifact.Stage{
				ID:   "test",
				Name: "New",
			},
			output: artifact.Spec{
				Stages: []artifact.Stage{
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
			spec: artifact.Spec{
				Stages: []artifact.Stage{
					{
						ID: "test",
					},
					{
						ID: "build",
					},
				},
			},
			stage: artifact.Stage{
				ID: "push",
			},
			output: artifact.Spec{
				Stages: []artifact.Stage{
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
