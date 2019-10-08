package command

import (
	"fmt"
	"os"

	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/spf13/cobra"
)

func NewCompletion(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion <shell>",
		Short: `Output shell completion code`,
		Long: `Output shell completion code for the specified shell (bash or zsh).
The shell code must be evaluated to provide interactive
completion of hamctl commands.  This can be done by sourcing it from
the .bash_profile.

Note for zsh users: zsh completions are only supported in versions of zsh >= 5.2

Installing bash completion on macOS using homebrew

    If running Bash 3.2 included with macOS

    	brew install bash-completion

    If running Bash 4.1+

    	brew install bash-completion@2

    You may need add the completion to your completion directory

    	hamctl completion bash > $(brew --prefix)/etc/bash_completion.d/hamctl

Installing bash completion on Linux

    If bash-completion is not installed on Linux, please install the 'bash-completion' package
    via your distribution's package manager.

    Load the hamctl completion code for bash into the current shell

    	source <(hamctl completion bash)

    Write bash completion code to a file and source if from .bash_profile

     	hamctl completion bash > ~/.hamctl/completion.bash.inc
     	printf "
     	            # hamctl shell completion
     	source '$HOME/.hamctl/completion.bash.inc'
     	            " >> $HOME/.bash_profile
    	source $HOME/.bash_profile

    Load the hamctl completion code for zsh[1] into the current shell

    	source <(hamctl completion zsh)

    Set the hamctl completion code for zsh[1] to autoload on startup

    	hamctl completion zsh > "${fpath[1]}/_hamctl"`,
		ValidArgs: []string{"bash", "zsh"},
		Args: func(cmd *cobra.Command, args []string) error {
			if cobra.ExactArgs(1)(cmd, args) != nil || cobra.OnlyValidArgs(cmd, args) != nil {
				return fmt.Errorf("only %v arguments are allowed", cmd.ValidArgs)
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "zsh":
				completion.Zsh(os.Stdout, rootCmd)
			case "bash":
				rootCmd.GenBashCompletion(os.Stdout)
			default:
			}
		},
	}

	return cmd
}
