package database

import (
	"database/sql"

	"network-monitor/internal/models"
)

// SaveResult saves a ping result to the database
func (db *DB) SaveResult(result models.PingResult) error {
	query := `
        INSERT INTO ping_results (timestamp, target, success, rtt_ms, error_message)
        VALUES (?, ?, ?, ?, ?)
    `
	_, err := db.Exec(query,
		result.Timestamp,
		result.Target,
		result.Success,
		result.RTT,
		result.ErrorMessage,
	)
	return err
}

// GetRecent retrieves recent ping results
func (db *DB) GetRecent(hours int) ([]models.PingResult, error) {
	query := `
        SELECT timestamp, target, success, rtt_ms, error_message
        FROM ping_results
        WHERE timestamp > datetime('now', '-' || ? || ' hours')
        ORDER BY timestamp DESC
        LIMIT 10000
    `

	rows, err := db.Query(query, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.PingResult
	for rows.Next() {
		var r models.PingResult
		var errMsg sql.NullString
		err := rows.Scan(&r.Timestamp, &r.Target, &r.Success, &r.RTT, &errMsg)
		if err != nil {
			continue
		}
		if errMsg.Valid {
			r.ErrorMessage = errMsg.String
		}
		results = append(results, r)
	}

	return results, nil
}

// GetStats retrieves aggregated statistics
func (db *DB) GetStats(hours int) ([]models.Stats, error) {
	query := `
        SELECT
            target,
            COUNT(*) as total_pings,
            SUM(CASE WHEN success THEN 1 ELSE 0 END) as successful_pings,
            AVG(CASE WHEN success THEN rtt_ms ELSE NULL END) as avg_rtt,
            MAX(CASE WHEN success THEN rtt_ms ELSE NULL END) as max_rtt,
            MIN(CASE WHEN success THEN rtt_ms ELSE NULL END) as min_rtt,
            ROUND((1.0 - (CAST(SUM(CASE WHEN success THEN 1 ELSE 0 END) AS REAL) / COUNT(*))) * 100, 2) as packet_loss
        FROM ping_results
        WHERE timestamp > datetime('now', '-' || ? || ' hours')
        GROUP BY target
    `

	rows, err := db.Query(query, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.Stats
	for rows.Next() {
		var s models.Stats
		err := rows.Scan(&s.Target, &s.TotalPings, &s.Successful,
			&s.AvgRTT, &s.MaxRTT, &s.MinRTT, &s.PacketLoss)
		if err != nil {
			continue
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// GetOutages retrieves detected outages using sliding window approach
func (db *DB) GetOutages(days int) ([]models.Outage, error) {
	query := `
        WITH windowed_pings AS (
            SELECT
                target,
                timestamp,
                success,
                COUNT(*) OVER (
                    PARTITION BY target
                    ORDER BY timestamp
                    ROWS 9 PRECEDING
                ) as window_size,
                SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) OVER (
                    PARTITION BY target
                    ORDER BY timestamp
                    ROWS 9 PRECEDING
                ) as failure_count
            FROM ping_results
            WHERE timestamp > datetime('now', '-' || ? || ' days')
        ),
        outage_periods AS (
            SELECT
                target,
                timestamp,
                success,
                CASE WHEN failure_count >= 5 AND window_size = 10 THEN 1 ELSE 0 END as is_outage,
                ROW_NUMBER() OVER (PARTITION BY target ORDER BY timestamp) -
                ROW_NUMBER() OVER (PARTITION BY target, CASE WHEN failure_count >= 5 AND window_size = 10 THEN 1 ELSE 0 END ORDER BY timestamp) as outage_grp
            FROM windowed_pings
        )
        SELECT
            target,
            MIN(timestamp) as start_time,
            MAX(timestamp) as end_time,
            COUNT(*) as failed_checks
        FROM outage_periods
        WHERE is_outage = 1
        GROUP BY target, outage_grp
        ORDER BY start_time DESC
        LIMIT 100
    `

	rows, err := db.Query(query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outages []models.Outage
	for rows.Next() {
		var o models.Outage
		err := rows.Scan(&o.Target, &o.StartTime, &o.EndTime, &o.FailedChecks)
		if err != nil {
			continue
		}
		o.Duration = o.EndTime.Sub(o.StartTime).String()
		outages = append(outages, o)
	}

	return outages, nil
}

// GetHeatmapData retrieves heatmap data
func (db *DB) GetHeatmapData(days int) ([]models.HeatmapPoint, error) {
	query := `
        SELECT
            hour,
            target,
            AVG(failure_rate) as avg_failure_rate,
            AVG(avg_rtt_ms) as avg_latency,
            MAX(max_rtt_ms) as max_latency,
            SUM(failed_pings) as total_failures,
            SUM(total_pings) as total_pings,
            COUNT(DISTINCT date) as days_with_data
        FROM hourly_patterns
        WHERE date > date('now', '-' || ? || ' days')
        GROUP BY hour, target
        ORDER BY hour, target
    `

	rows, err := db.Query(query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var heatmapData []models.HeatmapPoint
	for rows.Next() {
		var h models.HeatmapPoint
		var avgLatency, maxLatency sql.NullFloat64
		err := rows.Scan(&h.Hour, &h.Target, &h.FailureRate, &avgLatency,
			&maxLatency, &h.TotalFailures, &h.TotalPings, &h.DaysWithData)
		if err != nil {
			continue
		}
		if avgLatency.Valid {
			h.AvgLatency = avgLatency.Float64
		}
		if maxLatency.Valid {
			h.MaxLatency = maxLatency.Float64
		}
		heatmapData = append(heatmapData, h)
	}

	return heatmapData, nil
}

// GetPatterns retrieves pattern data for a specific hour
func (db *DB) GetPatterns(hour string) ([]models.PatternDetail, error) {
	query := `
        SELECT
            date,
            target,
            total_pings,
            failed_pings,
            avg_rtt_ms,
            max_rtt_ms,
            failure_rate
        FROM hourly_patterns
        WHERE hour = ?
        AND date > date('now', '-30 days')
        ORDER BY date DESC, target
    `

	rows, err := db.Query(query, hour)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []models.PatternDetail
	for rows.Next() {
		var p models.PatternDetail
		var avgRTT, maxRTT sql.NullFloat64
		err := rows.Scan(&p.Date, &p.Target, &p.TotalPings, &p.FailedPings,
			&avgRTT, &maxRTT, &p.FailureRate)
		if err != nil {
			continue
		}
		if avgRTT.Valid {
			p.AvgRTT = avgRTT.Float64
		}
		if maxRTT.Valid {
			p.MaxRTT = maxRTT.Float64
		}
		patterns = append(patterns, p)
	}

	return patterns, nil
}
