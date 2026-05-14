package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type sitesOptions struct {
	jsonOutput bool
}

func newSitesCommand() *cobra.Command {
	opts := sitesOptions{}

	cmd := &cobra.Command{
		Use:   "sites",
		Short: "List VRM sites available to the authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedToken, err := resolveToken()
			if err != nil {
				return err
			}

			user, err := fetchCurrentUser(resolvedToken.Value)
			if err != nil {
				return err
			}

			sites, err := fetchInstallations(resolvedToken.Value, user.User.ID)
			if err != nil {
				return err
			}

			if opts.jsonOutput {
				payload := struct {
					BaseURL string             `json:"base_url"`
					User    map[string]any     `json:"user"`
					Sites   []installationSite `json:"sites"`
				}{
					BaseURL: vrmAPIBaseURL,
					User: map[string]any{
						"id":    user.User.ID,
						"name":  user.User.Name,
						"email": user.User.Email,
					},
					Sites: sites.Records,
				}

				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(payload)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Sites for %s <%s>\n\n", user.User.Name, user.User.Email)
			if len(sites.Records) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No sites found.")
				return nil
			}

			for i, site := range sites.Records {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"[%d] %s\n  Site ID     %d\n  Identifier  %s\n  Timezone    %s\n  Access      %d\n\n",
					i+1,
					site.Name,
					site.IDSite,
					site.Identifier,
					site.Timezone,
					site.AccessLevel,
				)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output sites as JSON")
	return cmd
}

func init() {
	rootCmd.AddCommand(newSitesCommand())
}
