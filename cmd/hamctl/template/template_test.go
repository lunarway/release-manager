package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTmplMaxLength(t *testing.T) {
	type item struct {
		Field string
	}
	tt := []struct {
		name   string
		list   interface{}
		key    string
		output int
	}{
		{
			name:   "empty list",
			list:   nil,
			key:    "Field",
			output: 0,
		},
		{
			name:   "empty value",
			list:   nil,
			key:    "Field",
			output: 0,
		},
		{
			name: "single item",
			list: []item{
				{
					Field: "hello",
				},
			},
			key:    "Field",
			output: 5,
		},
		{
			name: "multiple items",
			list: []item{
				{
					Field: "hello",
				},
				{
					Field: "largeField",
				},
			},
			key:    "Field",
			output: 10,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output, err := tmplMaxLength(tc.list, tc.key)

			assert.NoError(t, err)
			assert.Equal(t, tc.output, output, "output not as expected")
		})
	}
}
