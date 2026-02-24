package kea

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

const maxResponseSize = 10 * 1024 * 1024 // 10 MB

var subnetRe = regexp.MustCompile(`^subnet\[(\d+)\]\.(.+)$`)

// Client queries the Kea Control Agent API.
type Client struct {
	URL        string
	HTTPClient *http.Client
}

// NewClient creates a Kea API client with the given URL and timeout.
func NewClient(url string, timeout time.Duration) *Client {
	return &Client{
		URL: url,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetStats queries statistic-get-all and returns parsed stats.
func (c *Client) GetStats() (*Stats, error) {
	resp, err := c.query()
	if err != nil {
		return nil, err
	}

	if resp.Result != 0 {
		return nil, fmt.Errorf("kea returned error result %d: %s", resp.Result, resp.Text)
	}

	return parseArguments(resp.Arguments)
}

// GetRawJSON queries statistic-get-all and returns the raw JSON response.
func (c *Client) GetRawJSON() ([]byte, error) {
	body, err := json.Marshal(Request{Command: "statistic-get-all"})
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("connecting to kea: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kea returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return data, nil
}

func (c *Client) query() (*Response, error) {
	body, err := json.Marshal(Request{Command: "statistic-get-all"})
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("connecting to kea: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kea returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Handle array-wrapped response: [{"result": 0, ...}]
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '[' {
		var responses []Response
		if err := json.Unmarshal(data, &responses); err != nil {
			return nil, fmt.Errorf("parsing array response: %w", err)
		}
		if len(responses) == 0 {
			return nil, fmt.Errorf("empty response array")
		}
		return &responses[0], nil
	}

	var keaResp Response
	if err := json.Unmarshal(data, &keaResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &keaResp, nil
}

// parseArguments extracts numeric values from Kea's stat format.
// Each stat is: "name": [[value, "timestamp"], ...] — we take [0][0].
func parseArguments(args map[string]json.RawMessage) (*Stats, error) {
	stats := &Stats{
		Global:  make(map[string]int64),
		Subnets: make(map[string]map[string]int64),
	}

	for key, raw := range args {
		value, ok := extractValue(raw)
		if !ok {
			continue
		}

		if matches := subnetRe.FindStringSubmatch(key); matches != nil {
			subnetID := matches[1]
			fieldName := matches[2]
			if stats.Subnets[subnetID] == nil {
				stats.Subnets[subnetID] = make(map[string]int64)
			}
			stats.Subnets[subnetID][fieldName] = value
		} else {
			stats.Global[key] = value
		}
	}

	return stats, nil
}

// extractValue parses [[value, "timestamp"], ...] and returns the first value as int64.
func extractValue(raw json.RawMessage) (int64, bool) {
	var dataPoints [][]json.RawMessage
	if err := json.Unmarshal(raw, &dataPoints); err != nil {
		return 0, false
	}
	if len(dataPoints) == 0 || len(dataPoints[0]) == 0 {
		return 0, false
	}

	// Try integer first
	var intVal int64
	if err := json.Unmarshal(dataPoints[0][0], &intVal); err == nil {
		return intVal, true
	}

	// Try float and truncate
	var floatVal float64
	if err := json.Unmarshal(dataPoints[0][0], &floatVal); err == nil {
		return int64(floatVal), true
	}

	// Non-numeric (e.g. string) — skip
	return 0, false
}
