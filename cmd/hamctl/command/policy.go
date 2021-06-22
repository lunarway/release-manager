package command

import (
	"github.com/go-openapi/runtime"
	"github.com/lunarway/release-manager/cmd/hamctl/command/policy"
	"github.com/lunarway/release-manager/generated/http/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewPolicy(client *client.ReleaseManagerServerAPI, clientAuth *runtime.ClientAuthInfoWriter, service *string) *cobra.Command {
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
	command.AddCommand(policy.NewApply(client, clientAuth, service))
	command.AddCommand(policy.NewList(client, clientAuth, service))
	command.AddCommand(policy.NewDelete(client, clientAuth, service))
	return command
}
