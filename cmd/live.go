package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var nowFunc = time.Now

type liveOptions struct {
	siteID       int
	once         bool
	jsonOutput   bool
	intervalText string
}

type liveSnapshot struct {
	SiteID          int       `json:"site_id"`
	ObservedAt      time.Time `json:"observed_at"`
	DataAge         string    `json:"data_age"`
	BatterySOC      string    `json:"battery_soc"`
	BatteryVoltage  string    `json:"battery_voltage"`
	BatteryCurrent  string    `json:"battery_current"`
	SolarPower      string    `json:"solar_power"`
	SolarState      string    `json:"solar_state"`
	SolarYieldToday string    `json:"solar_yield_today"`
	InverterState   string    `json:"inverter_state"`
	LoadPower       string    `json:"load_power"`
	GridImportPower string    `json:"grid_import_power"`
	GridExportPower string    `json:"grid_export_power"`
	Stale           bool      `json:"stale"`
	Warnings        []string  `json:"warnings,omitempty"`
}

func newLiveCommand() *cobra.Command {
	opts := liveOptions{intervalText: "10s"}

	cmd := &cobra.Command{
		Use:   "live [--site <id>]",
		Short: "Poll key live metrics for a VRM site",
		RunE: func(cmd *cobra.Command, args []string) error {
			interval, err := time.ParseDuration(opts.intervalText)
			if err != nil {
				return fmt.Errorf("invalid --interval value %q: %w", opts.intervalText, err)
			}
			if interval <= 0 {
				return fmt.Errorf("--interval must be greater than zero")
			}

			resolvedToken, err := resolveToken()
			if err != nil {
				return err
			}

			siteID, err := resolveSiteID(resolvedToken.Value, opts.siteID)
			if err != nil {
				return err
			}

			runOnce := func() error {
				snapshot, err := fetchLiveSnapshot(resolvedToken.Value, siteID)
				if err != nil {
					return err
				}
				return writeSnapshot(cmd, snapshot, opts.jsonOutput, true)
			}

			if opts.once {
				return runOnce()
			}

			if !opts.jsonOutput {
				program := tea.NewProgram(newLiveTUIModel(resolvedToken.Value, siteID, interval), tea.WithAltScreen())
				_, err := program.Run()
				return err
			}

			for {
				snapshot, err := fetchLiveSnapshot(resolvedToken.Value, siteID)
				if err != nil {
					return err
				}
				if err := writeSnapshot(cmd, snapshot, true, false); err != nil {
					return err
				}
				time.Sleep(interval)
			}
		},
	}

	cmd.Flags().IntVar(&opts.siteID, "site", 0, "VRM site ID")
	cmd.Flags().BoolVar(&opts.once, "once", false, "Fetch one snapshot and exit")
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output live snapshot as JSON")
	cmd.Flags().StringVar(&opts.intervalText, "interval", "10s", "Polling interval, for example 5s or 30s")
	return cmd
}

func writeSnapshot(cmd *cobra.Command, snapshot *liveSnapshot, jsonOutput, prettyJSON bool) error {
	if jsonOutput {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		if prettyJSON {
			encoder.SetIndent("", "  ")
		}
		return encoder.Encode(snapshot)
	}

	_, err := fmt.Fprint(cmd.OutOrStdout(), renderLiveSnapshot(snapshot))
	return err
}

func fetchLiveSnapshot(token string, siteID int) (*liveSnapshot, error) {
	overview, err := fetchSystemOverview(token, siteID)
	if err != nil {
		return nil, err
	}

	batteryInstance := firstDeviceInstance(overview.Records.Devices, 2)
	solarInstance := firstDeviceInstance(overview.Records.Devices, 4)
	vebusInstance := firstDeviceInstance(overview.Records.Devices, 1)

	snapshot := &liveSnapshot{
		SiteID:          siteID,
		ObservedAt:      nowFunc().Local(),
		DataAge:         "unknown",
		BatterySOC:      "n/a",
		BatteryVoltage:  "n/a",
		BatteryCurrent:  "n/a",
		SolarPower:      "n/a",
		SolarState:      "n/a",
		SolarYieldToday: "n/a",
		InverterState:   "n/a",
		LoadPower:       "n/a",
		GridImportPower: "n/a",
		GridExportPower: "n/a",
	}

	var staleFlags []bool

	if batteryInstance == 0 {
		snapshot.Warnings = append(snapshot.Warnings, "battery instance not found")
	} else {
		battery, err := fetchSummaryWidget(token, siteID, "BatterySummary", batteryInstance)
		if err != nil {
			snapshot.Warnings = append(snapshot.Warnings, err.Error())
		} else {
			if attr, ok := summaryAttributeByCode(battery, "SOC"); ok {
				snapshot.BatterySOC = chooseFormattedValue(attr)
			}
			if attr, ok := summaryAttributeByCode(battery, "V"); ok {
				snapshot.BatteryVoltage = chooseFormattedValue(attr)
			}
			if attr, ok := summaryAttributeByCode(battery, "I"); ok {
				snapshot.BatteryCurrent = chooseFormattedValue(attr)
			}
			if age, ok := summaryWidgetAgeValue(battery); ok {
				snapshot.DataAge = age
			}
			if summaryWidgetStale(battery) {
				staleFlags = append(staleFlags, true)
				if age, ok := summaryWidgetAgeValue(battery); ok {
					snapshot.Warnings = append(snapshot.Warnings, fmt.Sprintf("Battery data is stale (%s old)", age))
				}
			} else {
				staleFlags = append(staleFlags, false)
			}
		}
	}

	if solarInstance == 0 {
		snapshot.Warnings = append(snapshot.Warnings, "solar charger instance not found")
	} else {
		solar, err := fetchSummaryWidget(token, siteID, "SolarChargerSummary", solarInstance)
		if err != nil {
			snapshot.Warnings = append(snapshot.Warnings, err.Error())
		} else {
			if attr, ok := summaryAttributeByCode(solar, "ScW"); ok {
				snapshot.SolarPower = chooseFormattedValue(attr)
			}
			if attr, ok := summaryAttributeByCode(solar, "ScS"); ok {
				snapshot.SolarState = chooseFormattedValue(attr)
			}
			if attr, ok := summaryAttributeByCode(solar, "YT"); ok {
				snapshot.SolarYieldToday = chooseFormattedValue(attr)
			}
			if age, ok := summaryWidgetAgeValue(solar); ok && snapshot.DataAge == "unknown" {
				snapshot.DataAge = age
			}
			if summaryWidgetStale(solar) {
				staleFlags = append(staleFlags, true)
				if age, ok := summaryWidgetAgeValue(solar); ok {
					snapshot.Warnings = append(snapshot.Warnings, fmt.Sprintf("Solar data is stale (%s old)", age))
				}
			} else {
				staleFlags = append(staleFlags, false)
			}
		}
	}

	if vebusInstance == 0 {
		snapshot.Warnings = append(snapshot.Warnings, "VE.Bus instance not found")
	} else {
		status, err := fetchSummaryWidget(token, siteID, "Status", vebusInstance)
		if err != nil {
			snapshot.Warnings = append(snapshot.Warnings, err.Error())
		} else {
			if attr, ok := summaryAttributeByCode(status, "S"); ok {
				snapshot.InverterState = chooseFormattedValue(attr)
			}
			loadPower := sumSummaryFloatByCodes(status, "OP1", "OP2", "OP3")
			snapshot.LoadPower = formatPower(maxFloat(loadPower, 0))

			gridPower := sumSummaryFloatByCodes(status, "IP1", "IP2", "IP3")
			gridVoltage := sumSummaryFloatByCodes(status, "IV1", "IV2", "IV3")
			if gridVoltage > 0 {
				snapshot.GridImportPower = formatPower(maxFloat(gridPower, 0))
				snapshot.GridExportPower = formatPower(maxFloat(-gridPower, 0))
			} else {
				snapshot.GridImportPower = "n/a"
				snapshot.GridExportPower = "n/a"
				snapshot.Warnings = append(snapshot.Warnings, fmt.Sprintf("Grid input not detected (input voltage %s)", firstFormattedValue(status, "IV1", "IV2", "IV3")))
			}
			if age, ok := summaryWidgetAgeValue(status); ok && snapshot.DataAge == "unknown" {
				snapshot.DataAge = age
			}
			if summaryWidgetStale(status) {
				staleFlags = append(staleFlags, true)
				if age, ok := summaryWidgetAgeValue(status); ok {
					snapshot.Warnings = append(snapshot.Warnings, fmt.Sprintf("Status data is stale (%s old)", age))
				}
			} else {
				staleFlags = append(staleFlags, false)
			}
		}
	}

	for _, stale := range staleFlags {
		if stale {
			snapshot.Stale = true
			break
		}
	}

	if len(snapshot.Warnings) > 0 {
		snapshot.Warnings = uniqueStrings(snapshot.Warnings)
	}

	return snapshot, nil
}

func summaryAttributeByCode(widget *summaryWidgetResponse, code string) (*summaryWidgetAttribute, bool) {
	for key, meta := range widget.Records.Meta {
		if meta.Code != code {
			continue
		}

		payload, ok := widget.Records.Data[key]
		if !ok {
			return nil, false
		}

		var attr summaryWidgetAttribute
		if err := json.Unmarshal(payload, &attr); err != nil {
			return nil, false
		}

		return &attr, true
	}

	return nil, false
}

func summaryWidgetAgeValue(widget *summaryWidgetResponse) (string, bool) {
	payload, ok := widget.Records.Data["secondsAgo"]
	if !ok {
		return "", false
	}

	var age summaryWidgetAge
	if err := json.Unmarshal(payload, &age); err != nil {
		return "", false
	}

	return formatSecondsAge(age.Value), true
}

func summaryWidgetStale(widget *summaryWidgetResponse) bool {
	payload, ok := widget.Records.Data["hasOldData"]
	if !ok {
		return false
	}

	var stale bool
	if err := json.Unmarshal(payload, &stale); err != nil {
		return false
	}

	return stale
}

func chooseFormattedValue(attr *summaryWidgetAttribute) string {
	if attr == nil {
		return "n/a"
	}
	if attr.ValueFormattedWithUnit != "" {
		return attr.ValueFormattedWithUnit
	}
	if attr.FormattedValue != "" {
		return attr.FormattedValue
	}
	if attr.NameEnum != nil && *attr.NameEnum != "" {
		return *attr.NameEnum
	}
	if attr.Value != "" {
		return attr.Value
	}
	return "n/a"
}

func sumSummaryFloatByCodes(widget *summaryWidgetResponse, codes ...string) float64 {
	var total float64
	for _, code := range codes {
		attr, ok := summaryAttributeByCode(widget, code)
		if !ok || attr == nil || attr.ValueFloat == nil {
			continue
		}
		total += *attr.ValueFloat
	}
	return total
}

func formatPower(value float64) string {
	return fmt.Sprintf("%.0f W", value)
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func firstFormattedValue(widget *summaryWidgetResponse, codes ...string) string {
	for _, code := range codes {
		attr, ok := summaryAttributeByCode(widget, code)
		if ok && attr != nil {
			return chooseFormattedValue(attr)
		}
	}
	return "n/a"
}

func hasDetailedStaleWarning(warnings []string) bool {
	for _, warning := range warnings {
		if strings.Contains(strings.ToLower(warning), "data is stale") {
			return true
		}
	}
	return false
}

func renderLiveSnapshot(snapshot *liveSnapshot) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Live status for site %d\n\n", snapshot.SiteID))
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

func init() {
	rootCmd.AddCommand(newLiveCommand())
}
