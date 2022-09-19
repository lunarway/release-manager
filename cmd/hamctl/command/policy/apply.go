package policy

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

//go:generate moq -rm -out config_mock.go . GitConfigAPI
type GitConfigAPI interface {
	CommitterDetails() (*git.CommitterDetails, error)
}

func NewApply(client *httpinternal.Client, service *string, gitConfigAPI GitConfigAPI) *cobra.Command {
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
				return errors.New("please specify a policy")
			}
			return nil
		},
		ValidArgs: []string{"auto-release", "branch-restriction"},
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(autoRelease(client, service, gitConfigAPI))
	command.AddCommand(branchRestriction(client, service, gitConfigAPI))
	return command
}

func autoRelease(client *httpinternal.Client, service *string, gitConfigAPI GitConfigAPI) *cobra.Command {
	var branch, env string
	var command = &cobra.Command{
		Use:   "auto-release",
		Short: "Auto-release policy for releasing branch artifacts to an environment",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			committer, err := gitConfigAPI.CommitterDetails()
			if err != nil {
				return err
			}

			var resp httpinternal.ApplyPolicyResponse
			path, err := client.URL(pathAutoRelease)
			if err != nil {
				return err
			}
			err = client.Do(http.MethodPatch, path, httpinternal.ApplyAutoReleasePolicyRequest{
				Service:        *service,
				Branch:         branch,
				Environment:    env,
				CommitterEmail: committer.Email,
				CommitterName:  committer.Name,
			}, &resp)
			if err != nil {
				return err
			}

			fmt.Printf("[✓] Applied auto-release policy '%s' to service '%s'\n", resp.ID, resp.Service)
			return nil
		},
	}
	command.Flags().StringVarP(&branch, "branch", "b", "", "Branch to auto-release artifacts from")
	// errors are skipped here as the only case they can occour are if thee flag
	// does not exist on the command.
	//nolint:errcheck
	command.MarkFlagRequired("branch")
	completion.FlagAnnotation(command, "branch", "__hamctl_get_branches")
	command.Flags().StringVarP(&env, "env", "e", "", "Environment to release artifacts to")
	//nolint:errcheck
	command.MarkFlagRequired("env")
	completion.FlagAnnotation(command, "env", "__hamctl_get_environments")
	return command
}

func branchRestriction(client *httpinternal.Client, service *string, gitConfigAPI GitConfigAPI) *cobra.Command {
	var branchRegex, env string
	var command = &cobra.Command{
		Use:   "branch-restriction",
		Short: "Branch restriction policy for limiting releases by their origin branch",
		Long:  "Branch restriction policy for limiting releases of artifacts by their origin branch to specific environments",
		Args:  cobra.ExactArgs(0),
		RunE: func(c *cobra.Command, args []string) error {
			committer, err := gitConfigAPI.CommitterDetails()
			if err != nil {
				return err
			}

			var resp httpinternal.ApplyBranchRestrictionPolicyResponse
			path, err := client.URL(pathBranchRestrction)
			if err != nil {
				return err
			}
			err = client.Do(http.MethodPatch, path, httpinternal.ApplyBranchRestrictionPolicyRequest{
				Service:        *service,
				BranchRegex:    branchRegex,
				Environment:    env,
				CommitterEmail: committer.Email,
				CommitterName:  committer.Name,
			}, &resp)
			if err != nil {
				return err
			}

			fmt.Printf("[✓] Applied branch restriction policy '%s' to service '%s'\n", resp.ID, resp.Service)
			return nil
		},
	}
	command.Flags().StringVar(&branchRegex, "branch-regex", "", "Regular expression defining allowed branch names")
	// errors are skipped here as the only case they can occur are if the flag
	// does not exist on the command.
	//nolint:errcheck
	command.MarkFlagRequired("branch-regex")
	completion.FlagAnnotation(command, "branch-regex", "__hamctl_get_branches")
	command.Flags().StringVarP(&env, "env", "e", "", "Environment to apply restriction to")
	//nolint:errcheck
	command.MarkFlagRequired("env")
	completion.FlagAnnotation(command, "env", "__hamctl_get_environments")
	return command
}
