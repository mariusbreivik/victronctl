package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	vrmAPIBaseURL   = "https://vrmapi.victronenergy.com/v2"
	defaultTokenEnv = "VICTRON_VRM_TOKEN"
)

var (
	vrmAPIBaseURLValue = vrmAPIBaseURL
	vrmHTTPClient      = &http.Client{Timeout: 10 * time.Second}
)

type apiToken struct {
	Value  string
	Source string
}

type userMeResponse struct {
	Success   bool   `json:"success"`
	Errors    any    `json:"errors"`
	ErrorCode string `json:"error_code"`
	User      struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
}

type installationsResponse struct {
	Success bool               `json:"success"`
	Errors  any                `json:"errors"`
	Records []installationSite `json:"records"`
}

type installationSite struct {
	IDSite      int    `json:"idSite"`
	AccessLevel int    `json:"accessLevel"`
	Owner       bool   `json:"owner"`
	IsAdmin     bool   `json:"is_admin"`
	Name        string `json:"name"`
	Identifier  string `json:"identifier"`
	IDUser      int    `json:"idUser"`
	Timezone    string `json:"timezone"`
	DeviceIcon  string `json:"device_icon"`
}

type apiErrorResponse struct {
	Success   bool   `json:"success"`
	Errors    any    `json:"errors"`
	ErrorCode string `json:"error_code"`
}

type systemOverviewResponse struct {
	Success bool `json:"success"`
	Records struct {
		Devices []overviewDevice `json:"devices"`
	} `json:"records"`
	Errors any `json:"errors"`
}

type overviewDevice struct {
	Name            string `json:"name"`
	CustomName      any    `json:"customName"`
	ProductCode     string `json:"productCode"`
	ProductName     string `json:"productName"`
	IDSite          int    `json:"idSite"`
	FirmwareVersion string `json:"firmwareVersion"`
	LastConnection  any    `json:"lastConnection"`
	Connection      string `json:"connection"`
	Instance        int    `json:"instance"`
	IDDeviceType    int    `json:"idDeviceType"`
	Identifier      string `json:"identifier"`
}

type summaryWidgetResponse struct {
	Success bool `json:"success"`
	Records struct {
		Data map[string]json.RawMessage `json:"data"`
		Meta map[string]struct {
			Code string `json:"code"`
		} `json:"meta"`
		AttributeOrder []int `json:"attributeOrder"`
	} `json:"records"`
	Errors any `json:"errors"`
}

type summaryWidgetAttribute struct {
	Code                   string   `json:"code"`
	IDDataAttribute        int      `json:"idDataAttribute"`
	SecondsAgo             any      `json:"secondsAgo"`
	SecondsToNextLog       any      `json:"secondsToNextLog"`
	Value                  string   `json:"value"`
	ValueFloat             *float64 `json:"valueFloat"`
	ValueString            *string  `json:"valueString"`
	ValueEnum              any      `json:"valueEnum"`
	NameEnum               *string  `json:"nameEnum"`
	DataType               string   `json:"dataType"`
	DBusServiceType        any      `json:"dbusServiceType"`
	DBusPath               any      `json:"dbusPath"`
	Instance               int      `json:"instance"`
	FormattedValue         string   `json:"formattedValue"`
	ValueFormattedWithUnit string   `json:"valueFormattedWithUnit"`
	HasOldData             bool     `json:"hasOldData"`
}

type summaryWidgetAge struct {
	Value                  any    `json:"value"`
	ValueFormattedWithUnit string `json:"valueFormattedWithUnit"`
}

func resolveToken() (*apiToken, error) {
	token := strings.TrimSpace(os.Getenv(defaultTokenEnv))

	if token == "" {
		return nil, fmt.Errorf("missing API token: set %s", defaultTokenEnv)
	}

	return &apiToken{Value: token, Source: defaultTokenEnv}, nil
}

func newAuthenticatedRequest(method, path, token string) (*http.Request, error) {
	request, err := http.NewRequest(method, vrmAPIBaseURLValue+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request %s %s: %w", method, path, err)
	}

	request.Header.Set("x-authorization", "Token "+token)
	return request, nil
}

func fetchCurrentUser(token string) (*userMeResponse, error) {
	var payload userMeResponse
	if err := fetchJSON(token, http.MethodGet, "/users/me", &payload); err != nil {
		return nil, err
	}

	if !payload.Success {
		if payload.Errors != nil {
			return nil, fmt.Errorf("auth failed: %v", payload.Errors)
		}

		return nil, fmt.Errorf("auth failed")
	}

	return &payload, nil
}

func fetchInstallations(token string, userID int) (*installationsResponse, error) {
	var payload installationsResponse
	path := "/users/" + strconv.Itoa(userID) + "/installations"
	if err := fetchJSON(token, http.MethodGet, path, &payload); err != nil {
		return nil, err
	}

	if !payload.Success {
		if payload.Errors != nil {
			return nil, fmt.Errorf("list sites failed: %v", payload.Errors)
		}

		return nil, fmt.Errorf("list sites failed")
	}

	return &payload, nil
}

func resolveSiteID(token string, siteID int) (int, error) {
	if siteID > 0 {
		return siteID, nil
	}

	user, err := fetchCurrentUser(token)
	if err != nil {
		return 0, err
	}

	installations, err := fetchInstallations(token, user.User.ID)
	if err != nil {
		return 0, err
	}

	if len(installations.Records) == 1 {
		return installations.Records[0].IDSite, nil
	}

	if len(installations.Records) == 0 {
		return 0, fmt.Errorf("no sites found for authenticated user")
	}

	return 0, fmt.Errorf("multiple sites found; pass --site")
}

func fetchSystemOverview(token string, siteID int) (*systemOverviewResponse, error) {
	var payload systemOverviewResponse
	path := "/installations/" + strconv.Itoa(siteID) + "/system-overview"
	if err := fetchJSON(token, http.MethodGet, path, &payload); err != nil {
		return nil, err
	}

	if !payload.Success {
		if payload.Errors != nil {
			return nil, fmt.Errorf("overview failed: %v", payload.Errors)
		}

		return nil, fmt.Errorf("overview failed")
	}

	return &payload, nil
}

func fetchSummaryWidget(token string, siteID int, widget string, instance int) (*summaryWidgetResponse, error) {
	var payload summaryWidgetResponse
	path := "/installations/" + strconv.Itoa(siteID) + "/widgets/" + widget
	if instance > 0 {
		path += "?" + url.Values{"instance": []string{strconv.Itoa(instance)}}.Encode()
	}

	if err := fetchJSON(token, http.MethodGet, path, &payload); err != nil {
		return nil, err
	}

	if !payload.Success {
		if payload.Errors != nil {
			return nil, fmt.Errorf("widget %s failed: %v", widget, payload.Errors)
		}

		return nil, fmt.Errorf("widget %s failed", widget)
	}

	return &payload, nil
}

func fetchJSON(token, method, path string, target any) error {
	request, err := newAuthenticatedRequest(method, path, token)
	if err != nil {
		return err
	}

	response, err := vrmHTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("request %s%s: %w", vrmAPIBaseURLValue, path, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read response for %s: %w", path, err)
	}

	if response.StatusCode != http.StatusOK {
		var apiError apiErrorResponse
		if err := json.Unmarshal(body, &apiError); err == nil && apiError.Errors != nil {
			return fmt.Errorf("request %s failed with status %d: %v", path, response.StatusCode, apiError.Errors)
		}

		return fmt.Errorf("request %s failed with status %d", path, response.StatusCode)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode response for %s: %w", path, err)
	}

	return nil
}

func maskToken(token string) string {
	if len(token) <= 4 {
		return "****"
	}

	return token[:2] + strings.Repeat("*", len(token)-4) + token[len(token)-2:]
}

func formatUnixTimestamp(value any) string {
	switch v := value.(type) {
	case float64:
		return formatDisplayTime(time.Unix(int64(v), 0))
	case int64:
		return formatDisplayTime(time.Unix(v, 0))
	case int:
		return formatDisplayTime(time.Unix(int64(v), 0))
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return v
		}
		return formatDisplayTime(time.Unix(parsed, 0))
	case bool:
		if !v {
			return "unknown"
		}
	}

	return "unknown"
}

func formatDisplayTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05 MST")
}

func formatSecondsAge(value any) string {
	switch v := value.(type) {
	case float64:
		return formatDurationSeconds(int(v))
	case int:
		return formatDurationSeconds(v)
	case int64:
		return formatDurationSeconds(int(v))
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return v
		}
		return formatDurationSeconds(parsed)
	}

	return "unknown"
}

func formatDurationSeconds(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}

	d := time.Duration(seconds) * time.Second
	if d < time.Minute {
		return d.String()
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dd%dh", int(d.Hours())/24, int(d.Hours())%24)
}

func firstDeviceInstance(devices []overviewDevice, deviceType int) int {
	for _, device := range devices {
		if device.IDDeviceType == deviceType && device.Instance > 0 {
			return device.Instance
		}
	}

	return 0
}
