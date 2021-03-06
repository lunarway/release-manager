package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewVersion(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Prints the version number of hamctl",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
}
