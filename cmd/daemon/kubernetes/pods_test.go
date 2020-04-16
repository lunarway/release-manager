package kubernetes

import (
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
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
