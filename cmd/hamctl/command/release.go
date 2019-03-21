package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewRelease(options *Options) *cobra.Command {
	var serviceName, environment, branch, artifact string
	var command = &cobra.Command{
		Use:   "release",
		Short: `Release a specific artifact or latest artifact from a branch into a specific environment.`,
		Example: `Release artifact 'master-482c9d808e-3bf40478e5' from service 'product' into environemnt 'dev':

  hamctl release --service product --env dev --artifact master-482c9d808e-3bf40478e5

Release latest artifact from branch 'master' of service 'product' into environemnt 'dev':

  hamctl release --service product --env dev --branch master`,
		RunE: func(c *cobra.Command, args []string) error {
			if branch != "" && artifact != "" {
				return errors.New("--branch and --artifact cannot both be specificed")
			}
			if branch == "" && artifact == "" {
				return errors.New("--branch or --artifact is required")
			}
			url := options.httpBaseURL + "/release"

			client := &http.Client{
				Timeout: options.httpTimeout,
			}

			committerName, committerEmail, err := git.CommitterDetails()
			if err != nil {
				return err
			}

			releaseRequest := httpinternal.ReleaseRequest{
				Service:        serviceName,
				Environment:    environment,
				Branch:         branch,
				ArtifactID:     artifact,
				CommitterName:  committerName,
				CommitterEmail: committerEmail,
			}

			b := new(bytes.Buffer)
			err = json.NewEncoder(b).Encode(releaseRequest)
			if err != nil {
				return err
			}

			req, err := http.NewRequest(http.MethodPost, url, b)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+options.authToken)
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			decoder := json.NewDecoder(resp.Body)
			if resp.StatusCode != http.StatusOK {
				var r httpinternal.ErrorResponse
				err = decoder.Decode(&r)
				if err != nil {
					return err
				}
				return errors.New(r.Message)
			}

			var r httpinternal.ReleaseResponse

			err = decoder.Decode(&r)
			if err != nil {
				return errors.WithMessage(err, "decode HTTP response")
			}

			if r.Status != "" {
				fmt.Printf("%s\n", r.Status)
			} else {
				fmt.Printf("[✓] Release of %s to %s initialized\n", r.Tag, r.ToEnvironment)
			}

			return nil
		},
	}
	command.Flags().StringVar(&serviceName, "service", "", "service from with to release into specified environment (required)")
	command.MarkFlagRequired("service")
	command.Flags().StringVar(&environment, "env", "", "environment to release to (required)")
	command.MarkFlagRequired("env")
	command.Flags().StringVar(&branch, "branch", "", "release latest artifact from this branch (mutually exclusive with --artifact)")
	command.Flags().StringVar(&artifact, "artifact", "", "release this artifact id (mutually exclusive with --branch)")
	return command
}
