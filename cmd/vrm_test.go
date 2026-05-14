package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResolveToken(t *testing.T) {
	t.Setenv(defaultTokenEnv, "  secret-token  ")

	token, err := resolveToken()
	if err != nil {
		t.Fatalf("resolveToken returned error: %v", err)
	}

	if token.Value != "secret-token" {
		t.Fatalf("resolveToken value = %q, want %q", token.Value, "secret-token")
	}

	if token.Source != defaultTokenEnv {
		t.Fatalf("resolveToken source = %q, want %q", token.Source, defaultTokenEnv)
	}
}

func TestResolveTokenMissing(t *testing.T) {
	t.Setenv(defaultTokenEnv, "")

	_, err := resolveToken()
	if err == nil {
		t.Fatal("resolveToken returned nil error, want missing token error")
	}

	if !strings.Contains(err.Error(), defaultTokenEnv) {
		t.Fatalf("resolveToken error = %q, want env name %q", err.Error(), defaultTokenEnv)
	}
}

func TestNewAuthenticatedRequest(t *testing.T) {
	previousBaseURL := vrmAPIBaseURLValue
	vrmAPIBaseURLValue = "https://example.test/api"
	t.Cleanup(func() {
		vrmAPIBaseURLValue = previousBaseURL
	})

	request, err := newAuthenticatedRequest(http.MethodGet, "/users/me", "token-123")
	if err != nil {
		t.Fatalf("newAuthenticatedRequest returned error: %v", err)
	}

	if got, want := request.URL.String(), "https://example.test/api/users/me"; got != want {
		t.Fatalf("request URL = %q, want %q", got, want)
	}

	if got, want := request.Header.Get("x-authorization"), "Token token-123"; got != want {
		t.Fatalf("auth header = %q, want %q", got, want)
	}
}

func TestFetchJSONSuccessAndErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			if got, want := r.Header.Get("x-authorization"), "Token test-token"; got != want {
				t.Fatalf("auth header = %q, want %q", got, want)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "value": "ready"})
		case "/bad-status":
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"errors": []string{"invalid token"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	previousBaseURL := vrmAPIBaseURLValue
	previousClient := vrmHTTPClient
	vrmAPIBaseURLValue = server.URL
	vrmHTTPClient = server.Client()
	t.Cleanup(func() {
		vrmAPIBaseURLValue = previousBaseURL
		vrmHTTPClient = previousClient
	})

	var payload struct {
		Success bool   `json:"success"`
		Value   string `json:"value"`
	}

	if err := fetchJSON("test-token", http.MethodGet, "/ok", &payload); err != nil {
		t.Fatalf("fetchJSON success returned error: %v", err)
	}

	if !payload.Success || payload.Value != "ready" {
		t.Fatalf("fetchJSON payload = %+v, want success payload", payload)
	}

	err := fetchJSON("test-token", http.MethodGet, "/bad-status", &payload)
	if err == nil {
		t.Fatal("fetchJSON error path returned nil error")
	}

	if !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("fetchJSON error = %q, want API error details", err.Error())
	}
}

func TestResolveSiteIDAutoSelectsSingleInstallation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"user":    map[string]any{"id": 7, "name": "Marius", "email": "marius@example.com"},
			})
		case "/users/7/installations":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"records": []map[string]any{{"idSite": 42, "name": "Cabin", "identifier": "ABC", "timezone": "Europe/Oslo", "accessLevel": 3}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	previousBaseURL := vrmAPIBaseURLValue
	previousClient := vrmHTTPClient
	vrmAPIBaseURLValue = server.URL
	vrmHTTPClient = server.Client()
	t.Cleanup(func() {
		vrmAPIBaseURLValue = previousBaseURL
		vrmHTTPClient = previousClient
	})

	siteID, err := resolveSiteID("test-token", 0)
	if err != nil {
		t.Fatalf("resolveSiteID returned error: %v", err)
	}

	if siteID != 42 {
		t.Fatalf("resolveSiteID = %d, want 42", siteID)
	}
}

func TestResolveSiteIDRequiresExplicitSiteForMultipleInstallations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"user":    map[string]any{"id": 7, "name": "Marius", "email": "marius@example.com"},
			})
		case "/users/7/installations":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"records": []map[string]any{{"idSite": 42}, {"idSite": 43}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	previousBaseURL := vrmAPIBaseURLValue
	previousClient := vrmHTTPClient
	vrmAPIBaseURLValue = server.URL
	vrmHTTPClient = server.Client()
	t.Cleanup(func() {
		vrmAPIBaseURLValue = previousBaseURL
		vrmHTTPClient = previousClient
	})

	_, err := resolveSiteID("test-token", 0)
	if err == nil {
		t.Fatal("resolveSiteID returned nil error for multiple sites")
	}

	if !strings.Contains(err.Error(), "multiple sites found") {
		t.Fatalf("resolveSiteID error = %q, want multiple sites message", err.Error())
	}
}

func TestFormattingHelpers(t *testing.T) {
	timeValue := time.Date(2026, time.May, 14, 13, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	formattedTime := formatDisplayTime(timeValue)
	expectedTime := timeValue.Local().Format("2006-01-02 15:04:05 MST")
	if formattedTime != expectedTime {
		t.Fatalf("formatDisplayTime = %q, want %q", formattedTime, expectedTime)
	}

	if got, want := maskToken("abcdefghij"), "ab******ij"; got != want {
		t.Fatalf("maskToken = %q, want %q", got, want)
	}

	if got, want := formatDurationSeconds(3661), "1h1m"; got != want {
		t.Fatalf("formatDurationSeconds = %q, want %q", got, want)
	}

	if got, want := formatSecondsAge("125"), "2m5s"; got != want {
		t.Fatalf("formatSecondsAge = %q, want %q", got, want)
	}

	if got, want := fmt.Sprint(uniqueStrings([]string{" one ", "", "one", "two ", "two"})), "[one two]"; got != want {
		t.Fatalf("uniqueStrings = %s, want %s", got, want)
	}

	if got := formatUnixTimestamp(false); got != "unknown" {
		t.Fatalf("formatUnixTimestamp(false) = %q, want unknown", got)
	}
}
