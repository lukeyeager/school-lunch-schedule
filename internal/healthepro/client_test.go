package healthepro

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/lukeyeager/school-lunch-schedule/internal/metrics"
)

func testMetrics(t *testing.T) *metrics.Metrics {
	t.Helper()
	reg := prometheus.NewRegistry()
	return &metrics.Metrics{
		HealtheProRequests: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "healthepro_requests_total",
		}, []string{"status_code"}),
		SlackRequests: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "slack_requests_total",
		}, []string{"status_code"}),
	}
}

func makeResponse(t *testing.T, date, settingDisplay, origDisplay string, daysOff []any) string {
	t.Helper()

	setting := map[string]any{
		"current_display": json.RawMessage(settingDisplay),
		"days_off":        daysOff,
	}
	settingJSON, err := json.Marshal(setting)
	if err != nil {
		t.Fatalf("marshal setting: %v", err)
	}

	orig := map[string]any{
		"current_display": json.RawMessage(origDisplay),
		"days_off":        []any{},
	}
	origJSON, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal orig: %v", err)
	}

	resp := map[string]any{
		"data": []any{
			map[string]any{
				"day":              date,
				"setting":          string(settingJSON),
				"setting_original": string(origJSON),
			},
		},
	}
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	return string(out)
}

func TestFetchMenu_CurrentDisplay(t *testing.T) {
	date := "2026-04-20"
	display := `[{"type":"category","name":"Lunch Entree"},{"type":"recipe","name":"Pizza"}]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(makeResponse(t, date, display, `[]`, nil)))
	}))
	defer ts.Close()

	client := newClient(ts.URL, 1, 1, testMetrics(t))
	d, _ := time.Parse("2006-01-02", date)
	entry, err := client.FetchMenu(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.Source != "current" {
		t.Errorf("expected source=current, got %q", entry.Source)
	}
	if len(entry.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(entry.Items))
	}
	if entry.Items[1].Name != "Pizza" {
		t.Errorf("expected Pizza, got %q", entry.Items[1].Name)
	}
}

func TestFetchMenu_FallbackToOriginal(t *testing.T) {
	date := "2026-04-17"
	origDisplay := `[{"type":"category","name":"Lunch Entree"},{"type":"recipe","name":"Chicken Leg"}]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(makeResponse(t, date, `[]`, origDisplay, nil)))
	}))
	defer ts.Close()

	client := newClient(ts.URL, 1, 1, testMetrics(t))
	d, _ := time.Parse("2006-01-02", date)
	entry, err := client.FetchMenu(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.Source != "original" {
		t.Errorf("expected source=original, got %q", entry.Source)
	}
	if entry.Items[1].Name != "Chicken Leg" {
		t.Errorf("expected Chicken Leg, got %q", entry.Items[1].Name)
	}
}

func TestFetchMenu_DaysOff(t *testing.T) {
	date := "2026-04-18"
	display := `[{"type":"category","name":"Lunch Entree"},{"type":"recipe","name":"Pizza"}]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(makeResponse(t, date, display, `[]`, []any{"holiday"})))
	}))
	defer ts.Close()

	client := newClient(ts.URL, 1, 1, testMetrics(t))
	d, _ := time.Parse("2006-01-02", date)
	entry, err := client.FetchMenu(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry != nil {
		t.Errorf("expected nil entry for day-off, got %+v", entry)
	}
}

func TestFetchMenu_DateNotInResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer ts.Close()

	client := newClient(ts.URL, 1, 1, testMetrics(t))
	d, _ := time.Parse("2006-01-02", "2026-04-19")
	entry, err := client.FetchMenu(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry != nil {
		t.Errorf("expected nil for missing date, got %+v", entry)
	}
}

func TestFetchMenu_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := newClient(ts.URL, 1, 1, testMetrics(t))
	client.http.RetryMax = 0 // no retries in test
	d, _ := time.Parse("2006-01-02", "2026-04-20")
	_, err := client.FetchMenu(d)
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}
