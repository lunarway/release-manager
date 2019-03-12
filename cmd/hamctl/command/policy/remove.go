package policy

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRemove() *cobra.Command {
	var command = &cobra.Command{
		Use:   "remove",
		Short: "",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Printf("Remove command")
			return nil
		},
	}
	return command
}
