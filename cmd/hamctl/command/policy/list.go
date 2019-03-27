package policy

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewList(service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "list",
		Short: "List current policies",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Printf("List policies for %s\n", *service)
			return nil
		},
	}
	return command
}
