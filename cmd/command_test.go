package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func runCommand(t *testing.T, cmdArgs []string, commandFactory func() *cobra.Command) (string, error) {
	t.Helper()

	command := commandFactory()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(cmdArgs)

	err := command.Execute()
	return output.String(), err
}

func withMockVRMServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	previousBaseURL := vrmAPIBaseURLValue
	previousClient := vrmHTTPClient
	previousNow := nowFunc

	vrmAPIBaseURLValue = server.URL
	vrmHTTPClient = server.Client()

	t.Cleanup(func() {
		server.Close()
		vrmAPIBaseURLValue = previousBaseURL
		vrmHTTPClient = previousClient
		nowFunc = previousNow
	})
}

func decodeJSONOutput[T any](t *testing.T, output string) T {
	t.Helper()

	var payload T
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to decode JSON output: %v\noutput:\n%s", err, output)
	}
	return payload
}

func fixedNow() time.Time {
	return time.Date(2026, time.May, 14, 10, 30, 0, 0, time.UTC)
}
