package policy

import (
	"errors"
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/lunarway/release-manager/generated/http/client"
	"github.com/lunarway/release-manager/generated/http/client/policies"
	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/spf13/cobra"
)

func NewDelete(client *client.ReleaseManagerServerAPI, clientAuth *runtime.ClientAuthInfoWriter, service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "delete",
		Short: "Delete one or more policies by their id.",
		Args: func(c *cobra.Command, args []string) error {
			err := cobra.MinimumNArgs(1)(c, args)
			if err != nil {
				return errors.New("at least one policy id must be specified.")
			}
			return nil
		},
		Example: `List available policies:

	$ hamctl --service product policy list

	Policies for service product

	Auto-releases:

	BRANCH     ENV      ID
	master     dev      auto-release-master-dev
	master     prod     auto-release-master-prod

Delete a single policy:

	$ hamctl --service product policy delete auto-release-master-dev

Delete multiple policies:

	$ hamctl --service product policy delete auto-release-master-dev auto-release-master-prod
`,
		RunE: func(c *cobra.Command, args []string) error {
			committerName, committerEmail, err := git.CommitterDetails()
			if err != nil {
				return err
			}

			resp, err := client.Policies.DeletePolicies(policies.NewDeletePoliciesParams().WithBody(&models.DeletePolicyRequest{
				Service:        service,
				PolicyIds:      args,
				CommitterName:  &committerName,
				CommitterEmail: &committerEmail,
			}), *clientAuth)
			if err != nil {
				return err
			}

			fmt.Printf("Deleted %d policies\n", resp.Payload.Count)
			return nil
		},
	}
	// copied from cobra's default usage template with the addition of policy id arguments
	// https://github.com/spf13/cobra/blob/77e4d5aecc4d34e58f72e5a1c4a5a13ef55e6f44/command.go#L464-L487
	command.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}} [policy-id]{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)
	return command
}
