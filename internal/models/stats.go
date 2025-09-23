package models

import "time"

// Stats represents aggregated statistics for a target
type Stats struct {
	Target     string  `json:"target"`
	TotalPings int     `json:"total_pings"`
	Successful int     `json:"successful_pings"`
	AvgRTT     float64 `json:"avg_rtt"`
	MaxRTT     float64 `json:"max_rtt"`
	MinRTT     float64 `json:"min_rtt"`
	PacketLoss float64 `json:"packet_loss"`
}

// Outage represents a connectivity outage period
type Outage struct {
	Target       string    `json:"target"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	FailedChecks int       `json:"failed_checks"`
	Duration     string    `json:"duration"`
}

// HeatmapPoint represents a data point for the heatmap visualization
type HeatmapPoint struct {
	Hour          int     `json:"hour"`
	Target        string  `json:"target"`
	FailureRate   float64 `json:"failure_rate"`
	AvgLatency    float64 `json:"avg_latency"`
	MaxLatency    float64 `json:"max_latency"`
	TotalFailures int     `json:"total_failures"`
	TotalPings    int     `json:"total_pings"`
	DaysWithData  int     `json:"days_with_data"`
}

// PatternDetail represents detailed pattern data for a specific hour
type PatternDetail struct {
	Date        string  `json:"date"`
	Target      string  `json:"target"`
	TotalPings  int     `json:"total_pings"`
	FailedPings int     `json:"failed_pings"`
	AvgRTT      float64 `json:"avg_rtt"`
	MaxRTT      float64 `json:"max_rtt"`
	FailureRate float64 `json:"failure_rate"`
}
