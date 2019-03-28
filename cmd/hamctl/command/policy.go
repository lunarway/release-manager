package command

import (
	"github.com/lunarway/release-manager/cmd/hamctl/command/policy"
	"github.com/lunarway/release-manager/internal/http"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewPolicy(client *http.Client) *cobra.Command {
	var service string
	var command = &cobra.Command{
		Use:   "policy",
		Short: "Manage release policies for services.",
		// make sure that only valid args are applied and that at least one
		// command is specified
		Args: func(c *cobra.Command, args []string) error {
			err := cobra.OnlyValidArgs(c, args)
			if err != nil {
				return err
			}
			if len(args) == 0 {
				return errors.New("please specify a command.")
			}
			return nil
		},
		ValidArgs: []string{"apply", "list", "delete"},
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(policy.NewApply(client, &service))
	command.AddCommand(policy.NewList(client, &service))
	command.AddCommand(policy.NewDelete(client, &service))

	command.PersistentFlags().StringVar(&service, "service", "", "Service to manage policies for (required)")
	command.MarkPersistentFlagRequired("service")
	return command
}
