package policy

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func NewAdd(service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "add",
		Short: "Add a release policy for a service. See available commands for specific policies.",
		// make sure that only valid args are applied and that at least one
		// command is specified
		Args: func(c *cobra.Command, args []string) error {
			err := cobra.OnlyValidArgs(c, args)
			if err != nil {
				return err
			}
			if len(args) == 0 {
				return errors.New("please specify a policy.")
			}
			return nil
		},
		ValidArgs: []string{"auto-release"},
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(autoRelease(service))
	return command
}

func autoRelease(service *string) *cobra.Command {
	var branch, env string
	var command = &cobra.Command{
		Use:   "auto-release",
		Short: "Auto-release policy for releasing branch artifacts to an environment",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Printf("auto-release %s from %s to %s\n", *service, branch, env)
			return nil
		},
	}
	command.Flags().StringVar(&branch, "branch", "", "Branch to auto-release artifacts from")
	command.MarkFlagRequired("branch")
	command.Flags().StringVar(&env, "env", "", "Environment to release artifacts to")
	command.MarkFlagRequired("env")
	return command
}
