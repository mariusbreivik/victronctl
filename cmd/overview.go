package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type overviewOptions struct {
	siteID     int
	jsonOutput bool
}

func newOverviewCommand() *cobra.Command {
	opts := overviewOptions{}

	cmd := &cobra.Command{
		Use:   "overview [--site <id>]",
		Short: "Show the VRM system overview for a site",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedToken, err := resolveToken()
			if err != nil {
				return err
			}

			siteID, err := resolveSiteID(resolvedToken.Value, opts.siteID)
			if err != nil {
				return err
			}

			overview, err := fetchSystemOverview(resolvedToken.Value, siteID)
			if err != nil {
				return err
			}

			snapshot, err := fetchLiveSnapshot(resolvedToken.Value, siteID)
			if err != nil {
				return err
			}

			if opts.jsonOutput {
				payload := struct {
					BaseURL string           `json:"base_url"`
					SiteID  int              `json:"site_id"`
					Summary *liveSnapshot    `json:"summary"`
					Devices []overviewDevice `json:"devices"`
				}{
					BaseURL: vrmAPIBaseURL,
					SiteID:  siteID,
					Summary: snapshot,
					Devices: overview.Records.Devices,
				}

				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(payload)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "System overview for site %d\n\n", siteID)
			fmt.Fprint(cmd.OutOrStdout(), renderOverviewSummary(snapshot))
			fmt.Fprintln(cmd.OutOrStdout())
			if len(overview.Records.Devices) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No devices found.")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Devices")
			fmt.Fprintln(cmd.OutOrStdout())

			for i, device := range overview.Records.Devices {
				displayName := device.Name
				if customName, ok := device.CustomName.(string); ok && customName != "" {
					displayName = customName
				}

				fmt.Fprintf(
					cmd.OutOrStdout(),
					"[%d] %s\n  Product    %s\n  Firmware   %s\n  Last seen  %s\n\n",
					i+1,
					displayName,
					device.ProductName,
					device.FirmwareVersion,
					formatUnixTimestamp(device.LastConnection),
				)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&opts.siteID, "site", 0, "VRM site ID")
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output overview as JSON")
	return cmd
}

func init() {
	rootCmd.AddCommand(newOverviewCommand())
}

func renderOverviewSummary(snapshot *liveSnapshot) string {
	var builder strings.Builder
	builder.WriteString("Summary\n\n")
	builder.WriteString(fmt.Sprintf("Updated    %s\n", formatDisplayTime(snapshot.ObservedAt)))
	builder.WriteString(fmt.Sprintf("Data age   %s\n", snapshot.DataAge))
	if snapshot.Stale && !hasDetailedStaleWarning(snapshot.Warnings) {
		builder.WriteString("Warning    Data is stale\n")
	}
	for _, warning := range snapshot.Warnings {
		builder.WriteString(fmt.Sprintf("Warning    %s\n", warning))
	}
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("Battery    %s\n", snapshot.BatterySOC))
	builder.WriteString(fmt.Sprintf("Voltage    %s\n", snapshot.BatteryVoltage))
	builder.WriteString(fmt.Sprintf("Current    %s\n", snapshot.BatteryCurrent))
	builder.WriteString(fmt.Sprintf("Solar      %s\n", snapshot.SolarPower))
	builder.WriteString(fmt.Sprintf("Solar st   %s\n", snapshot.SolarState))
	builder.WriteString(fmt.Sprintf("Yield td   %s\n", snapshot.SolarYieldToday))
	builder.WriteString(fmt.Sprintf("Inverter   %s\n", snapshot.InverterState))
	builder.WriteString(fmt.Sprintf("Load       %s\n", snapshot.LoadPower))
	builder.WriteString(fmt.Sprintf("Grid in    %s\n", snapshot.GridImportPower))
	builder.WriteString(fmt.Sprintf("Grid out   %s\n", snapshot.GridExportPower))
	return builder.String()
}
