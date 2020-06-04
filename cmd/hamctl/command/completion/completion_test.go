package completion_test

import (
	"testing"

	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestFlagAnnotation(t *testing.T) {
	tt := []struct {
		name        string
		setup       func() *cobra.Command
		flag        string
		annotations map[string][]string
	}{
		{
			name: "required flag before completion",
			flag: "f",
			setup: func() *cobra.Command {
				c := cobra.Command{}
				c.Flags().String("f", "", "")
				c.MarkFlagRequired("f")
				completion.FlagAnnotation(&c, "f", "compFunc")
				return &c
			},
			annotations: map[string][]string{
				cobra.BashCompCustom:          {"compFunc"},
				cobra.BashCompOneRequiredFlag: {"true"},
			},
		},
		{
			name: "required flag after completion",
			flag: "f",
			setup: func() *cobra.Command {
				c := cobra.Command{}
				c.Flags().String("f", "", "")
				completion.FlagAnnotation(&c, "f", "compFunc")
				c.MarkFlagRequired("f")
				return &c
			},
			annotations: map[string][]string{
				cobra.BashCompCustom:          {"compFunc"},
				cobra.BashCompOneRequiredFlag: {"true"},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.setup()
			assert.Equal(t, tc.annotations, cmd.Flag(tc.flag).Annotations)
		})
	}
}
