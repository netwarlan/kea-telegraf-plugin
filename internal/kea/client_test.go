package kea

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Minimal real-shaped response with 2 subnets and global stats.
const testResponse = `[
  {
    "result": 0,
    "arguments": {
      "pkt4-received": [[5, "2026-02-23 17:36:02.458178"]],
      "pkt4-sent": [[3, "2026-02-23 17:36:02.458179"]],
      "pkt4-ack-sent": [[2, "2026-02-24 01:10:46.845834"], [0, "2026-02-23 17:36:02.458171"]],
      "v4-allocation-fail": [[0, "2026-02-23 17:36:02.458180"]],
      "subnet[1].total-addresses": [[231, "2026-02-23 17:36:02.494325"]],
      "subnet[1].assigned-addresses": [[10, "2026-02-23 17:36:02.495390"]],
      "subnet[1].pool[0].total-addresses": [[231, "2026-02-23 17:36:02.494349"]],
      "subnet[2].total-addresses": [[11, "2026-02-23 17:36:02.494384"]],
      "subnet[2].assigned-addresses": [[3, "2026-02-23 17:36:02.495433"]]
    }
  }
]`

func TestGetStats_ArrayWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(testResponse))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, 5*time.Second)
	stats, err := client.GetStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check global stats
	if stats.Global["pkt4-received"] != 5 {
		t.Errorf("pkt4-received = %d, want 5", stats.Global["pkt4-received"])
	}
	if stats.Global["pkt4-ack-sent"] != 2 {
		t.Errorf("pkt4-ack-sent = %d, want 2 (should use first data point)", stats.Global["pkt4-ack-sent"])
	}

	// Check subnet stats
	if stats.Subnets["1"]["total-addresses"] != 231 {
		t.Errorf("subnet[1].total-addresses = %d, want 231", stats.Subnets["1"]["total-addresses"])
	}
	if stats.Subnets["1"]["assigned-addresses"] != 10 {
		t.Errorf("subnet[1].assigned-addresses = %d, want 10", stats.Subnets["1"]["assigned-addresses"])
	}
	if stats.Subnets["1"]["pool[0].total-addresses"] != 231 {
		t.Errorf("subnet[1].pool[0].total-addresses = %d, want 231", stats.Subnets["1"]["pool[0].total-addresses"])
	}
	if stats.Subnets["2"]["total-addresses"] != 11 {
		t.Errorf("subnet[2].total-addresses = %d, want 11", stats.Subnets["2"]["total-addresses"])
	}
}

func TestGetStats_DirectResponse(t *testing.T) {
	directResponse := `{
		"result": 0,
		"arguments": {
			"pkt4-received": [[42, "2026-02-23 17:36:02.458178"]]
		}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(directResponse))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, 5*time.Second)
	stats, err := client.GetStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.Global["pkt4-received"] != 42 {
		t.Errorf("pkt4-received = %d, want 42", stats.Global["pkt4-received"])
	}
}

func TestGetStats_ErrorResult(t *testing.T) {
	errorResponse := `[{"result": 1, "text": "something went wrong"}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(errorResponse))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, 5*time.Second)
	_, err := client.GetStats()
	if err == nil {
		t.Fatal("expected error for non-zero result")
	}
}

func TestGetStats_SendsCorrectRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", ct)
		}

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decoding request: %v", err)
		}
		if req.Command != "statistic-get-all" {
			t.Errorf("command = %s, want statistic-get-all", req.Command)
		}

		w.Write([]byte(`[{"result": 0, "arguments": {}}]`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, 5*time.Second)
	client.GetStats()
}

func TestExtractValue_NonNumericSkipped(t *testing.T) {
	raw := json.RawMessage(`[["not-a-number", "2026-01-01 00:00:00.000"]]`)
	_, ok := extractValue(raw)
	if ok {
		t.Error("expected non-numeric value to be skipped")
	}
}

func TestExtractValue_EmptyArray(t *testing.T) {
	raw := json.RawMessage(`[]`)
	_, ok := extractValue(raw)
	if ok {
		t.Error("expected empty array to be skipped")
	}
}
