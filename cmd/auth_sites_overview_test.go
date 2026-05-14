package cmd

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAuthCommandJSONOutput(t *testing.T) {
	t.Setenv(defaultTokenEnv, "secret-token")
	withMockVRMServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got, want := r.Header.Get("x-authorization"), "Token secret-token"; got != want {
			t.Fatalf("auth header = %q, want %q", got, want)
		}
		_, _ = fmt.Fprint(w, `{"success":true,"user":{"id":7,"name":"Marius","email":"marius@example.com"}}`)
	})

	output, err := runCobraCommand(t, []string{"--json"}, newAuthCommand)
	if err != nil {
		t.Fatalf("auth command returned error: %v", err)
	}

	payload := decodeJSONOutput[struct {
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
	}](t, output)

	if payload.User.ID != 7 || payload.User.Name != "Marius" || payload.User.Email != "marius@example.com" {
		t.Fatalf("unexpected auth payload user: %+v", payload.User)
	}
	if payload.Token.Source != defaultTokenEnv || payload.Token.Masked != "se********en" {
		t.Fatalf("unexpected auth payload token: %+v", payload.Token)
	}
}

func TestAuthCommandHumanOutput(t *testing.T) {
	t.Setenv(defaultTokenEnv, "abcd1234")
	withMockVRMServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"success":true,"user":{"id":9,"name":"Alex","email":"alex@example.com"}}`)
	})

	output, err := runCobraCommand(t, nil, newAuthCommand)
	if err != nil {
		t.Fatalf("auth command returned error: %v", err)
	}

	assertContainsAll(t, output,
		"Authentication successful",
		"User      Alex <alex@example.com>",
		"User ID   9",
		"Token     VICTRON_VRM_TOKEN=ab****34",
	)
	if strings.Contains(output, "abcd1234") {
		t.Fatalf("output exposed full token: %q", output)
	}
}

func TestSitesCommandJSONOutput(t *testing.T) {
	t.Setenv(defaultTokenEnv, "secret-token")
	withMockVRMServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me":
			_, _ = fmt.Fprint(w, `{"success":true,"user":{"id":7,"name":"Marius","email":"marius@example.com"}}`)
		case "/users/7/installations":
			_, _ = fmt.Fprint(w, `{"success":true,"records":[{"idSite":42,"name":"Cabin","identifier":"abc123","timezone":"Europe/Oslo","accessLevel":3},{"idSite":43,"name":"Home","identifier":"def456","timezone":"Europe/Oslo","accessLevel":1}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	output, err := runCobraCommand(t, []string{"--json"}, newSitesCommand)
	if err != nil {
		t.Fatalf("sites command returned error: %v", err)
	}

	payload := decodeJSONOutput[struct {
		BaseURL string             `json:"base_url"`
		User    map[string]any     `json:"user"`
		Sites   []installationSite `json:"sites"`
	}](t, output)

	if payload.User["name"] != "Marius" {
		t.Fatalf("unexpected user payload: %+v", payload.User)
	}
	if len(payload.Sites) != 2 || payload.Sites[0].IDSite != 42 || payload.Sites[1].Name != "Home" {
		t.Fatalf("unexpected sites payload: %+v", payload.Sites)
	}
}

func TestSitesCommandHumanOutputWhenNoSites(t *testing.T) {
	t.Setenv(defaultTokenEnv, "secret-token")
	withMockVRMServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me":
			_, _ = fmt.Fprint(w, `{"success":true,"user":{"id":7,"name":"Marius","email":"marius@example.com"}}`)
		case "/users/7/installations":
			_, _ = fmt.Fprint(w, `{"success":true,"records":[]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	output, err := runCobraCommand(t, nil, newSitesCommand)
	if err != nil {
		t.Fatalf("sites command returned error: %v", err)
	}

	assertContainsAll(t, output,
		"Sites for Marius <marius@example.com>",
		"No sites found.",
	)
}

func TestOverviewCommandJSONOutput(t *testing.T) {
	t.Setenv(defaultTokenEnv, "secret-token")
	nowFunc = fixedNow
	withMockVRMServer(t, overviewFixtureHandler(t))

	output, err := runCobraCommand(t, []string{"--site", "42", "--json"}, newOverviewCommand)
	if err != nil {
		t.Fatalf("overview command returned error: %v", err)
	}

	payload := decodeJSONOutput[struct {
		BaseURL string           `json:"base_url"`
		SiteID  int              `json:"site_id"`
		Summary *liveSnapshot    `json:"summary"`
		Devices []overviewDevice `json:"devices"`
	}](t, output)

	if payload.SiteID != 42 {
		t.Fatalf("payload.SiteID = %d, want 42", payload.SiteID)
	}
	if payload.Summary == nil || payload.Summary.BatterySOC != "78%" || payload.Summary.GridImportPower != "400 W" {
		t.Fatalf("unexpected summary payload: %+v", payload.Summary)
	}
	if len(payload.Devices) != 3 || payload.Devices[0].Name != "Battery monitor" {
		t.Fatalf("unexpected devices payload: %+v", payload.Devices)
	}
}

func TestOverviewCommandHumanOutputUsesCustomNameAndSummary(t *testing.T) {
	t.Setenv(defaultTokenEnv, "secret-token")
	nowFunc = fixedNow
	withMockVRMServer(t, overviewFixtureHandler(t))

	output, err := runCobraCommand(t, []string{"--site", "42"}, newOverviewCommand)
	if err != nil {
		t.Fatalf("overview command returned error: %v", err)
	}

	assertContainsAll(t, output,
		"System overview for site 42",
		"Summary",
		"Updated    "+formatDisplayTime(fixedNow()),
		"Battery    78%",
		"Grid in    400 W",
		"Devices",
		"[1] House Battery",
		"Product    SmartShunt",
		"Last seen  "+formatUnixTimestamp(1747216800),
	)
	if strings.Contains(output, "[1] Battery monitor") {
		t.Fatalf("overview output ignored custom name: %q", output)
	}
}

func runCobraCommand(t *testing.T, args []string, factory func() *cobra.Command) (string, error) {
	t.Helper()

	return runCommand(t, args, factory)
}

func assertContainsAll(t *testing.T, output string, parts ...string) {
	t.Helper()

	for _, part := range parts {
		if !strings.Contains(output, part) {
			t.Fatalf("output missing %q\nfull output:\n%s", part, output)
		}
	}
}

func overviewFixtureHandler(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/installations/42/system-overview":
			_, _ = fmt.Fprint(w, `{"success":true,"records":{"devices":[{"name":"Battery monitor","customName":"House Battery","productName":"SmartShunt","firmwareVersion":"1.0.0","lastConnection":1747216800,"instance":3,"idDeviceType":2},{"name":"Solar charger","customName":null,"productName":"SmartSolar","firmwareVersion":"2.0.0","lastConnection":1747216800,"instance":4,"idDeviceType":4},{"name":"Inverter","customName":null,"productName":"MultiPlus","firmwareVersion":"3.0.0","lastConnection":1747216800,"instance":1,"idDeviceType":1}]}}`)
		case r.URL.Path == "/installations/42/widgets/BatterySummary":
			writeWidgetResponse(w, map[string]widgetAttributeSpec{
				"SOC": {FormattedWithUnit: "78%", Value: "78"},
				"V":   {FormattedWithUnit: "52.4 V", Value: "52.4"},
				"I":   {FormattedWithUnit: "-12 A", Value: "-12"},
			}, "35s", false)
		case r.URL.Path == "/installations/42/widgets/SolarChargerSummary":
			writeWidgetResponse(w, map[string]widgetAttributeSpec{
				"ScW": {FormattedWithUnit: "640 W", Value: "640", ValueFloat: floatPtr(640)},
				"ScS": {NameEnum: "Bulk"},
				"YT":  {FormattedWithUnit: "3.8 kWh", Value: "3.8"},
			}, "35s", false)
		case r.URL.Path == "/installations/42/widgets/Status":
			writeWidgetResponse(w, map[string]widgetAttributeSpec{
				"S":   {NameEnum: "Inverting"},
				"OP1": {FormattedWithUnit: "150 W", Value: "150", ValueFloat: floatPtr(150)},
				"OP2": {FormattedWithUnit: "50 W", Value: "50", ValueFloat: floatPtr(50)},
				"IP1": {FormattedWithUnit: "400 W", Value: "400", ValueFloat: floatPtr(400)},
				"IV1": {FormattedWithUnit: "230 V", Value: "230", ValueFloat: floatPtr(230)},
			}, "40s", false)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}
