package policy

import (
	"fmt"
	"strings"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewDelete(client *httpinternal.Client, service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "delete",
		Short: "Delete one or more policies",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Printf("Delete policies for %s: %s\n", *service, strings.Join(args, " "))
			return nil
		},
	}
	return command
}
