package policy

import (
	"fmt"
	"strings"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewRemove(client *httpinternal.Client, service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "remove",
		Short: "Remove policy",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Printf("Remove policies for %s: %s\n", *service, strings.Join(args, " "))
			return nil
		},
	}
	return command
}
