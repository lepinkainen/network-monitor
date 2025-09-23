package report

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (g *Generator) generateTextReport(outputDir string, hours int) error {
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

	rows, err := g.db.Query(query, hours)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Fprintln(file, "\nOVERALL STATISTICS")

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

	outageRows, outageErr := g.db.Query(outageQuery, hours)
	if outageErr != nil {
		return outageErr
	}
	defer outageRows.Close()

	fmt.Fprintln(file, "\nOUTAGE PERIODS (3+ consecutive failures)")

	outageCount := 0
	for outageRows.Next() {
		var target string
		var startTime, endTime time.Time
		var failedChecks int

		if scanErr := outageRows.Scan(&target, &startTime, &endTime, &failedChecks); scanErr != nil {
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
