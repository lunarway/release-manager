package policy

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewAdd() *cobra.Command {
	var command = &cobra.Command{
		Use:   "add",
		Short: "",
		RunE: func(c *cobra.Command, args []string) error {
			c.HelpFunc()(c, args)
			return nil
		},
	}

	command.AddCommand(newRelease())
	return command
}

func newRelease() *cobra.Command {
	var command = &cobra.Command{
		Use:   "release",
		Short: "",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Printf("release command")
			return nil
		},
	}

	return command
}
