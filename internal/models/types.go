package models

import (
	"context"
	"time"
)

// Database interface defines operations for data persistence
type Database interface {
	SaveResult(result PingResult) error
	GetRecent(hours int) ([]PingResult, error)
	GetStats(hours int) ([]Stats, error)
	GetOutages(days int) ([]Outage, error)
	GetHeatmapData(days int) ([]HeatmapPoint, error)
	GetPatterns(hour string) ([]PatternDetail, error)
	AggregateHourlyPatterns() error
	ArchiveOldData() error
	Close() error
}

// Pinger interface defines ping execution operations
type Pinger interface {
	Ping(target string, timeout time.Duration) (PingResult, error)
}

// Monitor interface defines the monitoring lifecycle
type Monitor interface {
	Start(ctx context.Context) error
	Stop() error
	Wait()
}

// WebServer interface defines web server operations
type WebServer interface {
	Start(port int) error
	Stop() error
}

// ReportGenerator interface defines report generation operations
type ReportGenerator interface {
	GenerateChart(filename string, data []PingResult) error
	GenerateTextReport(filename string, data []PingResult) error
}
