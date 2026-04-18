package slack

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/lukeyeager/school-lunch-schedule/internal/healthepro"
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

func sampleEntry(source string) *healthepro.DayEntry {
	return &healthepro.DayEntry{
		Day:    "2026-04-20",
		Source: source,
		Items: []healthepro.DisplayItem{
			{Type: "category", Name: "Lunch Entree"},
			{Type: "recipe", Name: "Pizza"},
		},
	}
}

func capturePost(t *testing.T) (*httptest.Server, func() string) {
	t.Helper()
	var captured string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	return ts, func() string { return captured }
}

func TestPostEveningPreview_ContainsTomorrow(t *testing.T) {
	ts, getText := capturePost(t)
	defer ts.Close()

	client := newClient(ts.URL, testMetrics(t))
	date, _ := time.Parse("2006-01-02", "2026-04-20")
	if err := client.PostEveningPreview(date, sampleEntry("current")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var msg slackMessage
	if err := json.Unmarshal([]byte(getText()), &msg); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if !strings.Contains(msg.Text, "Tomorrow") {
		t.Errorf("expected 'Tomorrow' in message, got: %s", msg.Text)
	}
	if !strings.Contains(msg.Text, "Pizza") {
		t.Errorf("expected 'Pizza' in message, got: %s", msg.Text)
	}
}

func TestPostMorningUpdate_ContainsUpdated(t *testing.T) {
	ts, getText := capturePost(t)
	defer ts.Close()

	client := newClient(ts.URL, testMetrics(t))
	date, _ := time.Parse("2006-01-02", "2026-04-20")
	if err := client.PostMorningUpdate(date, sampleEntry("current")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var msg slackMessage
	if err := json.Unmarshal([]byte(getText()), &msg); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if !strings.Contains(msg.Text, "Updated") {
		t.Errorf("expected 'Updated' in message, got: %s", msg.Text)
	}
}

func TestPost_FallbackWarning(t *testing.T) {
	ts, getText := capturePost(t)
	defer ts.Close()

	client := newClient(ts.URL, testMetrics(t))
	date, _ := time.Parse("2006-01-02", "2026-04-20")
	if err := client.PostEveningPreview(date, sampleEntry("original")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(getText(), "Recovered from backup") {
		t.Errorf("expected fallback warning, got: %s", getText())
	}
}

func TestPost_SlackError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid_payload"))
	}))
	defer ts.Close()

	client := newClient(ts.URL, testMetrics(t))
	client.http.RetryMax = 0
	date, _ := time.Parse("2006-01-02", "2026-04-20")
	err := client.PostEveningPreview(date, sampleEntry("current"))
	if err == nil {
		t.Fatal("expected error for 400, got nil")
	}
}
