package slack

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/lukeyeager/school-lunch-schedule/internal/healthepro"
	"github.com/lukeyeager/school-lunch-schedule/internal/metrics"
)

// Client posts messages to a Slack incoming webhook.
type Client struct {
	webhookURL string
	http       *retryablehttp.Client
	m          *metrics.Metrics
}

// NewClient creates a Client with exponential-backoff retry.
func NewClient(webhookURL string, m *metrics.Metrics) *Client {
	return newClient(webhookURL, m)
}

func newClient(webhookURL string, m *metrics.Metrics) *Client {
	rc := retryablehttp.NewClient()
	rc.RetryMax = 4
	rc.RetryWaitMin = 1 * time.Second
	rc.RetryWaitMax = 30 * time.Second
	rc.Logger = nil
	rc.ResponseLogHook = func(_ retryablehttp.Logger, resp *http.Response) {
		m.SlackRequests.WithLabelValues(fmt.Sprintf("%d", resp.StatusCode)).Inc()
	}
	return &Client{webhookURL: webhookURL, http: rc, m: m}
}

type slackMessage struct {
	Text string `json:"text"`
}

// PostEveningPreview posts tomorrow's menu as a night-before preview.
func (c *Client) PostEveningPreview(date time.Time, entry *healthepro.DayEntry) error {
	header := fmt.Sprintf(":fork_and_knife: *Tomorrow's Lunch* | %s", date.Format("Monday, January 2"))
	return c.post(header, entry)
}

// PostMorningUpdate posts today's menu when it changed since the evening preview.
func (c *Client) PostMorningUpdate(date time.Time, entry *healthepro.DayEntry) error {
	header := fmt.Sprintf(":warning: *Lunch Menu Updated* | %s", date.Format("Monday, January 2"))
	return c.post(header, entry)
}

func (c *Client) post(header string, entry *healthepro.DayEntry) error {
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")

	if entry.Source == "original" {
		sb.WriteString("_⚠️ Recovered from backup — app may show blank_\n\n")
	}

	for _, item := range entry.Items {
		switch item.Type {
		case "category":
			sb.WriteString(fmt.Sprintf("*%s*\n", item.Name))
		case "recipe":
			sb.WriteString(fmt.Sprintf("  • %s\n", item.Name))
		}
	}

	msg := slackMessage{Text: sb.String()}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling slack message: %w", err)
	}

	resp, err := c.http.Post(c.webhookURL, "application/json", body)
	if err != nil {
		return fmt.Errorf("posting to slack: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Warn("failed to close slack response body", "err", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}
	return nil
}
