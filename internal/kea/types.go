package kea

import "encoding/json"

// Request is the Kea Control Agent command format.
type Request struct {
	Command string `json:"command"`
}

// Response is the Kea Control Agent response format.
// Some Kea versions wrap the response in an array.
type Response struct {
	Result    int                        `json:"result"`
	Text      string                     `json:"text,omitempty"`
	Arguments map[string]json.RawMessage `json:"arguments"`
}

// Stats holds parsed Kea statistics separated into global and per-subnet maps.
type Stats struct {
	Global  map[string]int64
	Subnets map[string]map[string]int64 // keyed by subnet ID
}
