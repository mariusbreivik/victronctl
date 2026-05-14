package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchLiveSnapshotBuildsSnapshotFromWidgetData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/installations/42/system-overview":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"records": map[string]any{
					"devices": []map[string]any{
						{"idDeviceType": 2, "instance": 3, "name": "Battery"},
						{"idDeviceType": 4, "instance": 4, "name": "Solar Charger"},
						{"idDeviceType": 1, "instance": 1, "name": "VE.Bus"},
					},
				},
			})
		case r.URL.Path == "/installations/42/widgets/BatterySummary":
			if got, want := r.URL.Query().Get("instance"), "3"; got != want {
				t.Fatalf("battery instance = %q, want %q", got, want)
			}
			writeWidgetResponse(w, map[string]widgetAttributeSpec{
				"SOC": {FormattedWithUnit: "78%", Value: "78"},
				"V":   {FormattedWithUnit: "52.4 V", Value: "52.4"},
				"I":   {FormattedWithUnit: "-12 A", Value: "-12"},
			}, "35s", false)
		case r.URL.Path == "/installations/42/widgets/SolarChargerSummary":
			if got, want := r.URL.Query().Get("instance"), "4"; got != want {
				t.Fatalf("solar instance = %q, want %q", got, want)
			}
			writeWidgetResponse(w, map[string]widgetAttributeSpec{
				"ScW": {FormattedWithUnit: "640 W", Value: "640", ValueFloat: floatPtr(640)},
				"ScS": {NameEnum: "Bulk"},
				"YT":  {FormattedWithUnit: "3.8 kWh", Value: "3.8"},
			}, "35s", true)
		case r.URL.Path == "/installations/42/widgets/Status":
			if got, want := r.URL.Query().Get("instance"), "1"; got != want {
				t.Fatalf("status instance = %q, want %q", got, want)
			}
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
	}))
	defer server.Close()

	previousBaseURL := vrmAPIBaseURLValue
	previousClient := vrmHTTPClient
	previousNow := nowFunc
	vrmAPIBaseURLValue = server.URL
	vrmHTTPClient = server.Client()
	nowFunc = func() time.Time {
		return time.Date(2026, time.May, 14, 12, 30, 0, 0, time.FixedZone("CEST", 2*60*60))
	}
	t.Cleanup(func() {
		vrmAPIBaseURLValue = previousBaseURL
		vrmHTTPClient = previousClient
		nowFunc = previousNow
	})

	snapshot, err := fetchLiveSnapshot("test-token", 42)
	if err != nil {
		t.Fatalf("fetchLiveSnapshot returned error: %v", err)
	}

	if snapshot.SiteID != 42 {
		t.Fatalf("snapshot.SiteID = %d, want 42", snapshot.SiteID)
	}
	if snapshot.ObservedAt.Format(time.RFC3339) != "2026-05-14T12:30:00+02:00" {
		t.Fatalf("snapshot.ObservedAt = %s", snapshot.ObservedAt.Format(time.RFC3339))
	}
	if snapshot.DataAge != "35s" {
		t.Fatalf("snapshot.DataAge = %q, want 35s", snapshot.DataAge)
	}
	if snapshot.BatterySOC != "78%" || snapshot.BatteryVoltage != "52.4 V" || snapshot.BatteryCurrent != "-12 A" {
		t.Fatalf("unexpected battery fields: %+v", snapshot)
	}
	if snapshot.SolarPower != "640 W" || snapshot.SolarState != "Bulk" || snapshot.SolarYieldToday != "3.8 kWh" {
		t.Fatalf("unexpected solar fields: %+v", snapshot)
	}
	if snapshot.InverterState != "Inverting" {
		t.Fatalf("snapshot.InverterState = %q, want Inverting", snapshot.InverterState)
	}
	if snapshot.LoadPower != "200 W" {
		t.Fatalf("snapshot.LoadPower = %q, want 200 W", snapshot.LoadPower)
	}
	if snapshot.GridImportPower != "400 W" || snapshot.GridExportPower != "0 W" {
		t.Fatalf("unexpected grid values: in=%q out=%q", snapshot.GridImportPower, snapshot.GridExportPower)
	}
	if !snapshot.Stale {
		t.Fatal("snapshot.Stale = false, want true")
	}
	if len(snapshot.Warnings) != 1 || snapshot.Warnings[0] != "Solar data is stale (35s old)" {
		t.Fatalf("snapshot.Warnings = %#v, want one stale warning", snapshot.Warnings)
	}
}

func TestFetchLiveSnapshotWarnsWhenGridVoltageIsMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/installations/99/system-overview":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"records": map[string]any{
					"devices": []map[string]any{
						{"idDeviceType": 1, "instance": 1, "name": "VE.Bus"},
					},
				},
			})
		case r.URL.Path == "/installations/99/widgets/Status":
			writeWidgetResponse(w, map[string]widgetAttributeSpec{
				"S":   {NameEnum: "Passthru"},
				"IP1": {FormattedWithUnit: "250 W", Value: "250", ValueFloat: floatPtr(250)},
				"IV1": {FormattedWithUnit: "0 V", Value: "0", ValueFloat: floatPtr(0)},
			}, "10s", false)
		default:
			writeWidgetResponse(w, map[string]widgetAttributeSpec{}, "10s", false)
		}
	}))
	defer server.Close()

	previousBaseURL := vrmAPIBaseURLValue
	previousClient := vrmHTTPClient
	previousNow := nowFunc
	vrmAPIBaseURLValue = server.URL
	vrmHTTPClient = server.Client()
	nowFunc = time.Now
	t.Cleanup(func() {
		vrmAPIBaseURLValue = previousBaseURL
		vrmHTTPClient = previousClient
		nowFunc = previousNow
	})

	snapshot, err := fetchLiveSnapshot("test-token", 99)
	if err != nil {
		t.Fatalf("fetchLiveSnapshot returned error: %v", err)
	}

	if snapshot.GridImportPower != "n/a" || snapshot.GridExportPower != "n/a" {
		t.Fatalf("grid values = %q/%q, want n/a", snapshot.GridImportPower, snapshot.GridExportPower)
	}

	warnings := strings.Join(snapshot.Warnings, " | ")
	if !strings.Contains(warnings, "battery instance not found") {
		t.Fatalf("warnings = %q, want battery missing warning", warnings)
	}
	if !strings.Contains(warnings, "solar charger instance not found") {
		t.Fatalf("warnings = %q, want solar missing warning", warnings)
	}
	if !strings.Contains(warnings, "Grid input not detected (input voltage 0 V)") {
		t.Fatalf("warnings = %q, want grid voltage warning", warnings)
	}
}

func TestRenderLiveSnapshotOmitsGenericStaleWarningWhenDetailedWarningExists(t *testing.T) {
	snapshot := &liveSnapshot{
		SiteID:          42,
		ObservedAt:      time.Date(2026, time.May, 14, 12, 30, 0, 0, time.FixedZone("CEST", 2*60*60)),
		DataAge:         "35s",
		BatterySOC:      "78%",
		BatteryVoltage:  "52.4 V",
		BatteryCurrent:  "-12 A",
		SolarPower:      "640 W",
		SolarState:      "Bulk",
		SolarYieldToday: "3.8 kWh",
		InverterState:   "Inverting",
		LoadPower:       "200 W",
		GridImportPower: "400 W",
		GridExportPower: "0 W",
		Stale:           true,
		Warnings:        []string{"Solar data is stale (35s old)"},
	}

	rendered := renderLiveSnapshot(snapshot)
	if strings.Count(rendered, "Warning") != 1 {
		t.Fatalf("rendered snapshot warnings = %q", rendered)
	}
	if strings.Contains(rendered, "Warning    Data is stale") {
		t.Fatalf("rendered snapshot included duplicate stale warning: %q", rendered)
	}
	if !strings.Contains(rendered, "Warning    Solar data is stale (35s old)") {
		t.Fatalf("rendered snapshot missing detailed stale warning: %q", rendered)
	}
}

type widgetAttributeSpec struct {
	FormattedWithUnit string
	FormattedValue    string
	Value             string
	ValueFloat        *float64
	NameEnum          string
}

func writeWidgetResponse(w http.ResponseWriter, attrs map[string]widgetAttributeSpec, age string, stale bool) {
	data := map[string]any{
		"secondsAgo": map[string]any{"value": age, "valueFormattedWithUnit": age},
		"hasOldData": stale,
	}
	meta := map[string]any{}
	attributeOrder := make([]int, 0, len(attrs))

	index := 1
	for code, attr := range attrs {
		key := code
		payload := map[string]any{
			"code":                   code,
			"idDataAttribute":        index,
			"instance":               1,
			"formattedValue":         attr.FormattedValue,
			"value":                  attr.Value,
			"valueFormattedWithUnit": attr.FormattedWithUnit,
			"hasOldData":             false,
		}
		if attr.ValueFloat != nil {
			payload["valueFloat"] = *attr.ValueFloat
		}
		if attr.NameEnum != "" {
			payload["nameEnum"] = attr.NameEnum
		}
		data[key] = payload
		meta[key] = map[string]any{"code": code}
		attributeOrder = append(attributeOrder, index)
		index++
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"records": map[string]any{
			"data":           data,
			"meta":           meta,
			"attributeOrder": attributeOrder,
		},
	})
}

func floatPtr(value float64) *float64 {
	return &value
}
