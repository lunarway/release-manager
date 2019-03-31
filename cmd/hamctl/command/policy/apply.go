package policy

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewApply(client *httpinternal.Client, service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "apply",
		Short: "Apply a release policy for a service. See available commands for specific policies.",
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
	command.AddCommand(autoRelease(client, service))
	return command
}

func autoRelease(client *httpinternal.Client, service *string) *cobra.Command {
	var branch, env string
	var command = &cobra.Command{
		Use:   "auto-release",
		Short: "Auto-release policy for releasing branch artifacts to an environment",
		RunE: func(c *cobra.Command, args []string) error {
			committerName, committerEmail, err := git.CommitterDetails()
			if err != nil {
				return err
			}

			var resp httpinternal.ApplyPolicyResponse
			path, err := client.URL(path)
			if err != nil {
				return err
			}
			err = client.Do(http.MethodPatch, path, httpinternal.ApplyAutoReleasePolicyRequest{
				Service:        *service,
				Branch:         branch,
				Environment:    env,
				CommitterEmail: committerEmail,
				CommitterName:  committerName,
			}, &resp)
			if err != nil {
				return err
			}

			fmt.Printf("[✓] Applied auto-release policy '%s' to service '%s'\n", resp.ID, resp.Service)
			return nil
		},
	}
	command.Flags().StringVar(&branch, "branch", "", "Branch to auto-release artifacts from")
	command.MarkFlagRequired("branch")
	command.Flags().StringVar(&env, "env", "", "Environment to release artifacts to")
	command.MarkFlagRequired("env")
	return command
}
