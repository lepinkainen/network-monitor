package report

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

func (g *Generator) generateLatencyChart(outputDir string, hours int) error {
	query := `
        SELECT timestamp, target, rtt_ms
        FROM ping_results
        WHERE success = 1
        AND timestamp > datetime('now', '-' || ? || ' hours')
        ORDER BY timestamp
    `

	rows, err := g.db.Query(query, hours)
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

		if err := graph.Render(chart.PNG, file); err != nil {
			file.Close()
			return err
		}
		file.Close()
	}

	return nil
}

func (g *Generator) generateAvailabilityChart(outputDir string, hours int) error {
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

	rows, err := g.db.Query(query, hours)
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

		if scanErr := rows.Scan(&hourStr, &target, &uptime); scanErr != nil {
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

func (g *Generator) generateOutageSummary(outputDir string, hours int) error {
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

	rows, err := g.db.Query(query, hours)
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
		var values []chart.Value

		for hour, count := range hourlyOutages {
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
