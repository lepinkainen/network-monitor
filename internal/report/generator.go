package report

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Generator creates static images and reports for ISP evidence
type Generator struct {
	db *sql.DB
}

// NewGenerator creates a new report generator
func NewGenerator(db *sql.DB) *Generator {
	return &Generator{db: db}
}

// GenerateReport creates a comprehensive report with charts
func (g *Generator) GenerateReport(outputDir string, hours int) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	reportDir := filepath.Join(outputDir, fmt.Sprintf("network_report_%s", timestamp))
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Generate various charts
	if err := g.generateLatencyChart(reportDir, hours); err != nil {
		log.Printf("Failed to generate latency chart: %v", err)
	}

	if err := g.generateAvailabilityChart(reportDir, hours); err != nil {
		log.Printf("Failed to generate availability chart: %v", err)
	}

	if err := g.generateOutageSummary(reportDir, hours); err != nil {
		log.Printf("Failed to generate outage summary: %v", err)
	}

	if err := g.generateTextReport(reportDir, hours); err != nil {
		log.Printf("Failed to generate text report: %v", err)
	}

	log.Printf("Report generated in: %s", reportDir)
	return nil
}
