package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewPromote(options *Options) *cobra.Command {
	var serviceName, environment, configRepo, artifactFileName string
	var command = &cobra.Command{
		Use:   "promote",
		Short: "Promote a service to a specific environment following promoting conventions.",
		RunE: func(c *cobra.Command, args []string) error {
			url := options.httpBaseURL + "/promote"

			client := &http.Client{
				Timeout: options.httpTimeout,
			}

			promReq := httpinternal.PromoteRequest{
				Service:     serviceName,
				Environment: environment,
			}

			b := new(bytes.Buffer)
			err := json.NewEncoder(b).Encode(promReq)
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
			var r httpinternal.PromoteResponse

			err = decoder.Decode(&r)
			if err != nil {
				return err
			}

			if r.Status != "" {
				fmt.Printf("%s\n", r.Status)
			} else {
				fmt.Printf("[✓] Promotion of %s from %s to %s initialized\n", r.Tag, r.FromEnvironment, r.ToEnvironment)
			}

			return nil
		},
	}
	command.Flags().StringVar(&serviceName, "service", "", "Service to promote to specified environment (required)")
	command.MarkFlagRequired("service")
	command.Flags().StringVar(&environment, "env", "", "Environment to promote to (required)")
	command.MarkFlagRequired("env")
	command.Flags().StringVar(&configRepo, "config-repo", "git@github.com:lunarway/k8s-cluster-config.git", "Kubernetes cluster configuration repository.")
	command.Flags().StringVar(&artifactFileName, "file", "artifact.json", "")
	return command
}
