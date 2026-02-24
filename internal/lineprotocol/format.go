package lineprotocol

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"kea-telegraf-plugin/internal/kea"
)

var poolRe = regexp.MustCompile(`^pool\[(\d+)\]\.(.+)$`)

// Format converts parsed Kea stats into InfluxDB line protocol output.
// Returns one line for global stats and one line per subnet.
func Format(stats *kea.Stats, server string) string {
	var lines []string

	// Global stats line
	if line := formatLine("kea_dhcp4", server, "", stats.Global); line != "" {
		lines = append(lines, line)
	}

	// Per-subnet lines, sorted by subnet ID numerically
	subnetIDs := sortedSubnetIDs(stats.Subnets)
	for _, id := range subnetIDs {
		fields := stats.Subnets[id]
		if line := formatLine("kea_dhcp4", server, id, fields); line != "" {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

func formatLine(measurement, server, subnetID string, fields map[string]int64) string {
	if len(fields) == 0 {
		return ""
	}

	// Build tags
	tags := fmt.Sprintf("%s,server=%s", measurement, escapeTagValue(server))
	if subnetID != "" {
		tags += fmt.Sprintf(",subnet_id=%s", subnetID)
	}

	// Build fields with cleaned names, sorted for deterministic output
	fieldParts := make([]string, 0, len(fields))
	for name, value := range fields {
		cleaned := cleanFieldName(name)
		fieldParts = append(fieldParts, fmt.Sprintf("%s=%di", cleaned, value))
	}
	sort.Strings(fieldParts)

	return tags + " " + strings.Join(fieldParts, ",")
}

// cleanFieldName transforms Kea stat names into InfluxDB-safe field names.
//
//	"pkt4-received"                    → "pkt4_received"
//	"pool[0].total-addresses"          → "pool0_total_addresses"
//	"pool[0].cumulative-assigned-addr" → "pool0_cumulative_assigned_addresses"
func cleanFieldName(name string) string {
	// Handle pool[N].field → poolN_field
	if matches := poolRe.FindStringSubmatch(name); matches != nil {
		poolNum := matches[1]
		field := matches[2]
		name = fmt.Sprintf("pool%s_%s", poolNum, field)
	}

	// Replace dashes with underscores
	return strings.ReplaceAll(name, "-", "_")
}

func sortedSubnetIDs(subnets map[string]map[string]int64) []string {
	ids := make([]string, 0, len(subnets))
	for id := range subnets {
		ids = append(ids, id)
	}
	// Numeric sort via zero-padded comparison
	sort.Slice(ids, func(i, j int) bool {
		return padLeft(ids[i]) < padLeft(ids[j])
	})
	return ids
}

func padLeft(s string) string {
	return fmt.Sprintf("%010s", s)
}

// escapeTagValue escapes special characters in InfluxDB line protocol tag values.
func escapeTagValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, " ", `\ `)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "=", `\=`)
	return s
}
