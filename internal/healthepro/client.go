package healthepro

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/lukeyeager/school-lunch-schedule/internal/metrics"
)

const baseURL = "https://menus.healthepro.com"

// DisplayItem is one entry in a day's current_display array.
type DisplayItem struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// DayEntry is the resolved menu for a single school day.
type DayEntry struct {
	Day    string        // ISO date "2006-01-02"
	Items  []DisplayItem // from current_display, after fallback logic
	Source string        // "current" or "original"
}

// Client fetches menu data from the Health-e Pro API.
type Client struct {
	baseURL string
	orgID   int
	menuID  int
	http    *retryablehttp.Client
	m       *metrics.Metrics
}

// NewClient creates a Client with exponential-backoff retry.
func NewClient(orgID, menuID int, m *metrics.Metrics) *Client {
	return newClient(baseURL, orgID, menuID, m)
}

func newClient(base string, orgID, menuID int, m *metrics.Metrics) *Client {
	rc := retryablehttp.NewClient()
	rc.RetryMax = 4
	rc.RetryWaitMin = 1 * time.Second
	rc.RetryWaitMax = 30 * time.Second
	rc.Logger = nil
	rc.ResponseLogHook = func(_ retryablehttp.Logger, resp *http.Response) {
		m.HealtheProRequests.WithLabelValues(fmt.Sprintf("%d", resp.StatusCode)).Inc()
	}
	return &Client{baseURL: base, orgID: orgID, menuID: menuID, http: rc, m: m}
}

// rawSetting is the parsed form of the JSON-string "setting" field.
type rawSetting struct {
	CurrentDisplay []struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"current_display"`
	DaysOff []any `json:"days_off"`
}

type apiResponse struct {
	Data []struct {
		Day             string `json:"day"`
		Setting         string `json:"setting"`
		SettingOriginal string `json:"setting_original"`
	} `json:"data"`
}

// FetchMenu fetches the lunch menu for the given date.
// Returns nil if there is no school that day or no menu data is available.
func (c *Client) FetchMenu(date time.Time) (*DayEntry, error) {
	url := fmt.Sprintf("%s/api/organizations/%d/menus/%d/year/%d/month/%d/date_overwrites",
		c.baseURL, c.orgID, c.menuID, date.Year(), int(date.Month()))

	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("healthepro request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Warn("failed to close response body", "err", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("healthepro returned status %d", resp.StatusCode)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	dateStr := date.Format("2006-01-02")
	for _, d := range apiResp.Data {
		if d.Day != dateStr {
			continue
		}

		var setting rawSetting
		if err := json.Unmarshal([]byte(d.Setting), &setting); err != nil {
			return nil, fmt.Errorf("parsing setting for %s: %w", dateStr, err)
		}
		if len(setting.DaysOff) > 0 {
			return nil, nil // explicitly marked no-school
		}

		items, source := resolveItems(d.Setting, d.SettingOriginal)
		if items == nil {
			return nil, nil // no menu data in either field
		}

		return &DayEntry{Day: dateStr, Items: items, Source: source}, nil
	}

	return nil, nil // date not present in month's response
}

// resolveItems returns items from setting.current_display, falling back to
// setting_original.current_display when the primary field is empty (a known
// data-entry bug where overwrites blank the display list).
func resolveItems(settingJSON, settingOrigJSON string) ([]DisplayItem, string) {
	if items := extractItems(settingJSON); len(items) > 0 {
		return items, "current"
	}
	if items := extractItems(settingOrigJSON); len(items) > 0 {
		return items, "original"
	}
	return nil, ""
}

func extractItems(settingJSON string) []DisplayItem {
	var s rawSetting
	if err := json.Unmarshal([]byte(settingJSON), &s); err != nil {
		return nil
	}
	items := make([]DisplayItem, 0, len(s.CurrentDisplay))
	for _, it := range s.CurrentDisplay {
		if it.Type == "category" || it.Type == "recipe" {
			items = append(items, DisplayItem{Type: it.Type, Name: it.Name})
		}
	}
	return items
}
