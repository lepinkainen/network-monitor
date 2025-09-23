package models

import "time"

// PingResult represents a single ping measurement
type PingResult struct {
	Timestamp    time.Time `json:"timestamp"`
	Target       string    `json:"target"`
	Success      bool      `json:"success"`
	RTT          float64   `json:"rtt_ms"`      // milliseconds
	PacketLoss   float64   `json:"packet_loss"` // percentage
	ErrorMessage string    `json:"error_message"`
}
