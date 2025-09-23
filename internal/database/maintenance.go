package database

import (
	"time"
)

// AggregateHourlyPatterns aggregates hourly patterns for heatmap
func (db *DB) AggregateHourlyPatterns() error {
	query := `
        INSERT OR REPLACE INTO hourly_patterns (date, hour, target, total_pings, failed_pings, avg_rtt_ms, max_rtt_ms, failure_rate)
        SELECT
            strftime('%Y-%m-%d', timestamp) as date,
            CAST(strftime('%H', timestamp) AS INTEGER) as hour,
            target,
            COUNT(*) as total_pings,
            SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failed_pings,
            AVG(CASE WHEN success THEN rtt_ms ELSE NULL END) as avg_rtt_ms,
            MAX(CASE WHEN success THEN rtt_ms ELSE NULL END) as max_rtt_ms,
            ROUND((SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) * 100.0 / COUNT(*)), 2) as failure_rate
        FROM ping_results
        WHERE timestamp > datetime('now', '-2 days')
        AND strftime('%Y-%m-%d', timestamp) IS NOT NULL
        GROUP BY date, hour, target
    `
	_, err := db.Exec(query)
	return err
}

// ArchiveOldData archives old data and cleans up
func (db *DB) ArchiveOldData() error {
	// First, ensure hourly stats are captured for old data
	archiveQuery := `
        INSERT OR IGNORE INTO hourly_stats (hour, target, total_pings, successful_pings, avg_rtt_ms, max_rtt_ms, min_rtt_ms, packet_loss_percent)
        SELECT
            strftime('%Y-%m-%d %H:00:00', timestamp) as hour,
            target,
            COUNT(*) as total_pings,
            SUM(CASE WHEN success THEN 1 ELSE 0 END) as successful_pings,
            AVG(CASE WHEN success THEN rtt_ms ELSE NULL END) as avg_rtt_ms,
            MAX(CASE WHEN success THEN rtt_ms ELSE NULL END) as max_rtt_ms,
            MIN(CASE WHEN success THEN rtt_ms ELSE NULL END) as min_rtt_ms,
            ROUND((1.0 - (CAST(SUM(CASE WHEN success THEN 1 ELSE 0 END) AS REAL) / COUNT(*))) * 100, 2) as packet_loss_percent
        FROM ping_results
        WHERE timestamp < datetime('now', '-7 days')
        AND timestamp > datetime('now', '-90 days')
        GROUP BY hour, target
    `

	if _, err := db.Exec(archiveQuery); err != nil {
		return err
	}

	// Delete raw ping results older than 7 days (we keep aggregated data)
	deleteQuery := `DELETE FROM ping_results WHERE timestamp < datetime('now', '-7 days')`
	if _, err := db.Exec(deleteQuery); err != nil {
		return err
	}

	// Delete hourly patterns older than 90 days
	deletePatternQuery := `DELETE FROM hourly_patterns WHERE date < date('now', '-90 days')`
	if _, err := db.Exec(deletePatternQuery); err != nil {
		return err
	}

	// Vacuum to reclaim space (run occasionally)
	if time.Now().Day() == 1 { // Run on first day of month
		_, err := db.Exec("VACUUM")
		return err
	}

	return nil
}
