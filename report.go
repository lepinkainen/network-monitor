package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

// ReportGenerator creates static images and reports for ISP evidence
type ReportGenerator struct {
	db *sql.DB
}

func NewReportGenerator(db *sql.DB) *ReportGenerator {
	return &ReportGenerator{db: db}
}

// GenerateReport creates a comprehensive report with charts
func (r *ReportGenerator) GenerateReport(outputDir string, hours int) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	reportDir := filepath.Join(outputDir, fmt.Sprintf("network_report_%s", timestamp))
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Generate various charts
	if err := r.generateLatencyChart(reportDir, hours); err != nil {
		log.Printf("Failed to generate latency chart: %v", err)
	}

	if err := r.generateAvailabilityChart(reportDir, hours); err != nil {
		log.Printf("Failed to generate availability chart: %v", err)
	}

	if err := r.generateOutageSummary(reportDir, hours); err != nil {
		log.Printf("Failed to generate outage summary: %v", err)
	}

	if err := r.generateTextReport(reportDir, hours); err != nil {
		log.Printf("Failed to generate text report: %v", err)
	}

	log.Printf("Report generated in: %s", reportDir)
	return nil
}

func (r *ReportGenerator) generateLatencyChart(outputDir string, hours int) error {
	query := `
        SELECT timestamp, target, rtt_ms
        FROM ping_results
        WHERE success = 1 
        AND timestamp > datetime('now', '-' || ? || ' hours')
        ORDER BY timestamp
    `

	rows, err := r.db.Query(query, hours)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Group data by target
	targetData := make(map[string]struct {
		timestamps []time.Time
		values     []float64
	})

	for rows.Next() {
		var timestamp time.Time
		var target string
		var rtt float64

		if err := rows.Scan(&timestamp, &target, &rtt); err != nil {
			continue
		}

		data := targetData[target]
		data.timestamps = append(data.timestamps, timestamp)
		data.values = append(data.values, rtt)
		targetData[target] = data
	}

	// Create chart for each target
	for target, data := range targetData {
		graph := chart.Chart{
			Title: fmt.Sprintf("Network Latency - %s", target),
			TitleStyle: chart.Style{
				FontSize: 16,
			},
			Background: chart.Style{
				Padding: chart.Box{
					Top:    20,
					Left:   20,
					Right:  20,
					Bottom: 20,
				},
			},
			Width:  1200,
			Height: 400,
			XAxis: chart.XAxis{
				Name: "Time",
				NameStyle: chart.Style{
					FontSize: 12,
				},
				Style: chart.Style{
					StrokeColor: drawing.ColorBlack,
					FontSize:    10,
				},
				ValueFormatter: chart.TimeMinuteValueFormatter,
			},
			YAxis: chart.YAxis{
				Name: "Latency (ms)",
				NameStyle: chart.Style{
					FontSize: 12,
				},
				Style: chart.Style{
					StrokeColor: drawing.ColorBlack,
					FontSize:    10,
				},
				GridMajorStyle: chart.Style{
					StrokeColor: drawing.Color{R: 200, G: 200, B: 200, A: 255},
					StrokeWidth: 1.0,
				},
			},
			Series: []chart.Series{
				chart.TimeSeries{
					Name: target,
					Style: chart.Style{
						StrokeColor: chart.GetDefaultColor(0),
						StrokeWidth: 2,
					},
					XValues: data.timestamps,
					YValues: data.values,
				},
			},
		}

		// Add moving average
		if len(data.values) > 10 {
			ts := graph.Series[0].(chart.TimeSeries)
			graph.Series = append(graph.Series, chart.SMASeries{
				Name: "Moving Avg",
				Style: chart.Style{
					StrokeColor:     chart.GetDefaultColor(1),
					StrokeWidth:     2,
					StrokeDashArray: []float64{5, 5},
				},
				InnerSeries: ts,
				Period:      10,
			})
		}

		filename := filepath.Join(outputDir, fmt.Sprintf("latency_%s.png", sanitizeFilename(target)))
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		if err := graph.Render(chart.PNG, file); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReportGenerator) generateAvailabilityChart(outputDir string, hours int) error {
	query := `
        WITH hourly_stats AS (
            SELECT 
                strftime('%Y-%m-%d %H:00:00', timestamp) as hour,
                target,
                COUNT(*) as total,
                SUM(CASE WHEN success THEN 1 ELSE 0 END) as successful
            FROM ping_results
            WHERE timestamp > datetime('now', '-' || ? || ' hours')
            GROUP BY hour, target
            ORDER BY hour
        )
        SELECT 
            hour, 
            target,
            (CAST(successful AS REAL) / total) * 100 as uptime_percent
        FROM hourly_stats
    `

	rows, err := r.db.Query(query, hours)
	if err != nil {
		return err
	}
	defer rows.Close()

	targetData := make(map[string]struct {
		timestamps []time.Time
		values     []float64
	})

	for rows.Next() {
		var hourStr string
		var target string
		var uptime float64

		if err := rows.Scan(&hourStr, &target, &uptime); err != nil {
			continue
		}

		hour, _ := time.Parse("2006-01-02 15:04:05", hourStr)

		data := targetData[target]
		data.timestamps = append(data.timestamps, hour)
		data.values = append(data.values, uptime)
		targetData[target] = data
	}

	// Combined availability chart
	var allSeries []chart.Series
	colorIndex := 0

	for target, data := range targetData {
		allSeries = append(allSeries, chart.TimeSeries{
			Name: target,
			Style: chart.Style{
				StrokeColor: chart.GetDefaultColor(colorIndex),
				StrokeWidth: 2,
			},
			XValues: data.timestamps,
			YValues: data.values,
		})
		colorIndex++
	}

	graph := chart.Chart{
		Title: "Network Availability (Hourly)",
		TitleStyle: chart.Style{
			FontSize: 16,
		},
		Background: chart.Style{
			Padding: chart.Box{
				Top:    20,
				Left:   20,
				Right:  20,
				Bottom: 20,
			},
		},
		Width:  1200,
		Height: 400,
		XAxis: chart.XAxis{
			Name: "Time",
			Style: chart.Style{
				StrokeColor: drawing.ColorBlack,
				FontSize:    10,
			},
			ValueFormatter: chart.TimeHourValueFormatter,
		},
		YAxis: chart.YAxis{
			Name: "Uptime %",
			Style: chart.Style{
				StrokeColor: drawing.ColorBlack,
				FontSize:    10,
			},
			Range: &chart.ContinuousRange{
				Min: 0,
				Max: 100,
			},
			GridMajorStyle: chart.Style{
				StrokeColor: drawing.Color{R: 200, G: 200, B: 200, A: 255},
				StrokeWidth: 1.0,
			},
		},
		Series: allSeries,
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	filename := filepath.Join(outputDir, "availability.png")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return graph.Render(chart.PNG, file)
}

func (r *ReportGenerator) generateOutageSummary(outputDir string, hours int) error {
	// Query for outage periods
	query := `
        WITH outage_detection AS (
            SELECT 
                target,
                timestamp,
                success,
                LAG(success, 1, 1) OVER (PARTITION BY target ORDER BY timestamp) as prev_success,
                LEAD(success, 1, 1) OVER (PARTITION BY target ORDER BY timestamp) as next_success
            FROM ping_results
            WHERE timestamp > datetime('now', '-' || ? || ' hours')
        ),
        outage_events AS (
            SELECT 
                target,
                timestamp,
                CASE 
                    WHEN success = 0 AND prev_success = 1 THEN 'start'
                    WHEN success = 0 AND next_success = 1 THEN 'end'
                    WHEN success = 0 THEN 'ongoing'
                END as event_type
            FROM outage_detection
            WHERE success = 0
        )
        SELECT * FROM outage_events
        ORDER BY timestamp
    `

	rows, err := r.db.Query(query, hours)
	if err != nil {
		return err
	}
	defer rows.Close()

	type OutageEvent struct {
		Target    string
		Timestamp time.Time
		EventType string
	}

	var events []OutageEvent
	for rows.Next() {
		var e OutageEvent
		if err := rows.Scan(&e.Target, &e.Timestamp, &e.EventType); err != nil {
			continue
		}
		events = append(events, e)
	}

	// Create bar chart showing outage count by hour
	hourlyOutages := make(map[string]int)
	for _, e := range events {
		hour := e.Timestamp.Format("2006-01-02 15:00")
		hourlyOutages[hour]++
	}

	if len(hourlyOutages) > 0 {
		var categories []string
		var values []chart.Value

		for hour, count := range hourlyOutages {
			categories = append(categories, hour)
			values = append(values, chart.Value{
				Label: hour,
				Value: float64(count),
			})
		}

		graph := chart.BarChart{
			Title: "Outage Events by Hour",
			TitleStyle: chart.Style{
				FontSize: 16,
			},
			Background: chart.Style{
				Padding: chart.Box{
					Top:    20,
					Left:   20,
					Right:  20,
					Bottom: 20,
				},
			},
			Width:    1200,
			Height:   400,
			Bars:     values,
			BarWidth: 40,
		}

		filename := filepath.Join(outputDir, "outage_frequency.png")
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		if err := graph.Render(chart.PNG, file); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReportGenerator) generateTextReport(outputDir string, hours int) error {
	filename := filepath.Join(outputDir, "summary.txt")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "Network Connectivity Report\n")
	fmt.Fprintf(file, "Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "Period: Last %d hours\n\n", hours)
	fmt.Fprintln(file, strings.Repeat("=", 60))

	// Overall statistics
	query := `
        SELECT 
            target,
            COUNT(*) as total_pings,
            SUM(CASE WHEN success THEN 1 ELSE 0 END) as successful_pings,
            AVG(CASE WHEN success THEN rtt_ms ELSE NULL END) as avg_rtt,
            MAX(CASE WHEN success THEN rtt_ms ELSE NULL END) as max_rtt,
            MIN(CASE WHEN success THEN rtt_ms ELSE NULL END) as min_rtt
        FROM ping_results
        WHERE timestamp > datetime('now', '-' || ? || ' hours')
        GROUP BY target
    `

	rows, err := r.db.Query(query, hours)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Fprintln(file, "\nOVERALL STATISTICS\n")

	for rows.Next() {
		var target string
		var total, successful int
		var avgRTT, maxRTT, minRTT sql.NullFloat64

		if err := rows.Scan(&target, &total, &successful, &avgRTT, &maxRTT, &minRTT); err != nil {
			continue
		}

		uptime := float64(successful) / float64(total) * 100
		packetLoss := 100 - uptime

		fmt.Fprintf(file, "Target: %s\n", target)
		fmt.Fprintf(file, "  Total Pings: %d\n", total)
		fmt.Fprintf(file, "  Successful: %d (%.2f%%)\n", successful, uptime)
		fmt.Fprintf(file, "  Packet Loss: %.2f%%\n", packetLoss)

		if avgRTT.Valid {
			fmt.Fprintf(file, "  Average RTT: %.2f ms\n", avgRTT.Float64)
			fmt.Fprintf(file, "  Min RTT: %.2f ms\n", minRTT.Float64)
			fmt.Fprintf(file, "  Max RTT: %.2f ms\n", maxRTT.Float64)
		}
		fmt.Fprintln(file)
	}

	fmt.Fprintln(file, strings.Repeat("=", 60))

	// Outage periods
	outageQuery := `
        WITH grouped_failures AS (
            SELECT 
                target,
                timestamp,
                success,
                ROW_NUMBER() OVER (PARTITION BY target ORDER BY timestamp) -
                ROW_NUMBER() OVER (PARTITION BY target, success ORDER BY timestamp) as grp
            FROM ping_results
            WHERE timestamp > datetime('now', '-' || ? || ' hours')
        )
        SELECT 
            target,
            MIN(timestamp) as start_time,
            MAX(timestamp) as end_time,
            COUNT(*) as failed_checks
        FROM grouped_failures
        WHERE success = 0
        GROUP BY target, grp
        HAVING COUNT(*) >= 3
        ORDER BY start_time DESC
    `

	rows, err = r.db.Query(outageQuery, hours)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Fprintln(file, "\nOUTAGE PERIODS (3+ consecutive failures)\n")

	outageCount := 0
	for rows.Next() {
		var target string
		var startTime, endTime time.Time
		var failedChecks int

		if err := rows.Scan(&target, &startTime, &endTime, &failedChecks); err != nil {
			continue
		}

		duration := endTime.Sub(startTime)
		fmt.Fprintf(file, "Outage #%d\n", outageCount+1)
		fmt.Fprintf(file, "  Target: %s\n", target)
		fmt.Fprintf(file, "  Start: %s\n", startTime.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(file, "  End: %s\n", endTime.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(file, "  Duration: %s\n", duration)
		fmt.Fprintf(file, "  Failed Checks: %d\n", failedChecks)
		fmt.Fprintln(file)

		outageCount++
	}

	if outageCount == 0 {
		fmt.Fprintln(file, "No significant outages detected.")
	} else {
		fmt.Fprintf(file, "\nTotal Outages: %d\n", outageCount)
	}

	fmt.Fprintln(file, strings.Repeat("=", 60))
	fmt.Fprintln(file, "\nThis report documents network connectivity issues.")
	fmt.Fprintln(file, "Charts and detailed data are available in the accompanying files.")

	return nil
}

func sanitizeFilename(s string) string {
	// Replace dots and special characters for safe filenames
	replacer := strings.NewReplacer(
		".", "_",
		":", "_",
		"/", "_",
		"\\", "_",
		" ", "_",
	)
	return replacer.Replace(s)
}
