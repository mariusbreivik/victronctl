package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type authOptions struct {
	jsonOutput bool
}

func newAuthCommand() *cobra.Command {
	opts := authOptions{}

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Validate auth configuration for the VRM API",
		Long:  fmt.Sprintf("Validate auth configuration for the Victron VRM API at %s.", vrmAPIBaseURL),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedToken, err := resolveToken()
			if err != nil {
				return err
			}

			response, err := fetchCurrentUser(resolvedToken.Value)
			if err != nil {
				return err
			}

			if opts.jsonOutput {
				payload := struct {
					BaseURL string `json:"base_url"`
					User    struct {
						ID    int    `json:"id"`
						Name  string `json:"name"`
						Email string `json:"email"`
					} `json:"user"`
					Token struct {
						Source string `json:"source"`
						Masked string `json:"masked"`
					} `json:"token"`
				}{
					BaseURL: vrmAPIBaseURL,
				}

				payload.User.ID = response.User.ID
				payload.User.Name = response.User.Name
				payload.User.Email = response.User.Email
				payload.Token.Source = resolvedToken.Source
				payload.Token.Masked = maskToken(resolvedToken.Value)

				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(payload)
			}

			fmt.Fprintf(
				cmd.OutOrStdout(),
				"Authentication successful\n\nBase URL  %s\nUser      %s <%s>\nUser ID   %d\nToken     %s=%s\n",
				vrmAPIBaseURL,
				response.User.Name,
				response.User.Email,
				response.User.ID,
				resolvedToken.Source,
				maskToken(resolvedToken.Value),
			)

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output authentication result as JSON")

	return cmd
}

func init() {
	rootCmd.AddCommand(newAuthCommand())
}
