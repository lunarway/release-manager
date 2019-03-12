package policy

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewList() *cobra.Command {
	var command = &cobra.Command{
		Use:   "list",
		Short: "",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Printf("List command")
			return nil
		},
	}
	return command
}
