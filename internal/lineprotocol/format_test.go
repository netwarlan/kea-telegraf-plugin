package lineprotocol

import (
	"strings"
	"testing"

	"kea-telegraf-plugin/internal/kea"
)

func TestCleanFieldName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"pkt4-received", "pkt4_received"},
		{"pkt4-ack-sent", "pkt4_ack_sent"},
		{"total-addresses", "total_addresses"},
		{"pool[0].total-addresses", "pool0_total_addresses"},
		{"pool[0].cumulative-assigned-addresses", "pool0_cumulative_assigned_addresses"},
		{"pool[1].declined-addresses", "pool1_declined_addresses"},
		{"v4-allocation-fail", "v4_allocation_fail"},
		{"v4-lease-reuses", "v4_lease_reuses"},
		{"assigned-addresses", "assigned_addresses"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanFieldName(tt.input)
			if got != tt.want {
				t.Errorf("cleanFieldName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormat_GlobalAndSubnets(t *testing.T) {
	stats := &kea.Stats{
		Global: map[string]int64{
			"pkt4-received": 100,
			"pkt4-sent":     50,
		},
		Subnets: map[string]map[string]int64{
			"1": {
				"total-addresses":          231,
				"assigned-addresses":       10,
				"pool[0].total-addresses":  231,
			},
			"2": {
				"total-addresses":    11,
				"assigned-addresses": 3,
			},
		},
	}

	output := Format(stats, "dhcp-server-01")
	lines := strings.Split(output, "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}

	// Global line
	if !strings.HasPrefix(lines[0], "kea_dhcp4,server=dhcp-server-01 ") {
		t.Errorf("global line prefix wrong: %s", lines[0])
	}
	if !strings.Contains(lines[0], "pkt4_received=100i") {
		t.Errorf("global line missing pkt4_received: %s", lines[0])
	}
	if !strings.Contains(lines[0], "pkt4_sent=50i") {
		t.Errorf("global line missing pkt4_sent: %s", lines[0])
	}

	// Subnet 1 line (should come before subnet 2)
	if !strings.Contains(lines[1], "subnet_id=1") {
		t.Errorf("expected subnet_id=1 on line 2: %s", lines[1])
	}
	if !strings.Contains(lines[1], "total_addresses=231i") {
		t.Errorf("subnet 1 missing total_addresses: %s", lines[1])
	}
	if !strings.Contains(lines[1], "pool0_total_addresses=231i") {
		t.Errorf("subnet 1 missing pool0_total_addresses: %s", lines[1])
	}

	// Subnet 2 line
	if !strings.Contains(lines[2], "subnet_id=2") {
		t.Errorf("expected subnet_id=2 on line 3: %s", lines[2])
	}
	if !strings.Contains(lines[2], "total_addresses=11i") {
		t.Errorf("subnet 2 missing total_addresses: %s", lines[2])
	}
}

func TestFormat_EmptyStats(t *testing.T) {
	stats := &kea.Stats{
		Global:  map[string]int64{},
		Subnets: map[string]map[string]int64{},
	}

	output := Format(stats, "test")
	if output != "" {
		t.Errorf("expected empty output, got: %s", output)
	}
}

func TestFormat_SubnetOrdering(t *testing.T) {
	stats := &kea.Stats{
		Global: map[string]int64{"pkt4-received": 1},
		Subnets: map[string]map[string]int64{
			"10": {"total-addresses": 231},
			"2":  {"total-addresses": 11},
			"1":  {"total-addresses": 231},
			"41": {"total-addresses": 231},
		},
	}

	output := Format(stats, "test")
	lines := strings.Split(output, "\n")

	// line 0 = global, lines 1-4 = subnets
	expected := []string{"subnet_id=1", "subnet_id=2", "subnet_id=10", "subnet_id=41"}
	for i, exp := range expected {
		if !strings.Contains(lines[i+1], exp) {
			t.Errorf("line %d: expected %s, got: %s", i+1, exp, lines[i+1])
		}
	}
}
