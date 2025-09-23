package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed static/*
var staticFiles embed.FS

type Config struct {
	Targets      []string
	Interval     time.Duration
	Timeout      time.Duration
	DatabasePath string
	Port         int
}

type PingResult struct {
	Timestamp    time.Time `json:"timestamp"`
	Target       string    `json:"target"`
	Success      bool      `json:"success"`
	RTT          float64   `json:"rtt_ms"`      // milliseconds
	PacketLoss   float64   `json:"packet_loss"` // percentage
	ErrorMessage string    `json:"error_message"`
}

type Monitor struct {
	config  Config
	db      *sql.DB
	results chan PingResult
	wg      sync.WaitGroup
}

func main() {
	var (
		interval = flag.Duration("interval", 1*time.Second, "Ping interval")
		timeout  = flag.Duration("timeout", 5*time.Second, "Ping timeout")
		dbPath   = flag.String("db", "network_monitor.db", "Database path")
		port     = flag.Int("port", 8080, "Web server port")
		targets  = flag.String("targets", "8.8.8.8,1.1.1.1,208.67.222.222", "Comma-separated ping targets")
	)
	flag.Parse()

	config := Config{
		Targets:      strings.Split(*targets, ","),
		Interval:     *interval,
		Timeout:      *timeout,
		DatabasePath: *dbPath,
		Port:         *port,
	}

	monitor, err := NewMonitor(config)
	if err != nil {
		log.Fatal("Failed to create monitor:", err)
	}
	defer monitor.Close()

	// Start monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go monitor.Start(ctx)
	go monitor.StartWebServer()

	log.Printf("Monitor started. Pinging %v every %v", config.Targets, config.Interval)
	log.Printf("Web interface available at http://localhost:%d", config.Port)

	<-sigChan
	log.Println("Shutting down...")
	cancel()
	monitor.Wait()
}

func NewMonitor(config Config) (*Monitor, error) {
	db, err := initDatabase(config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	return &Monitor{
		config:  config,
		db:      db,
		results: make(chan PingResult, 100),
	}, nil
}

func (m *Monitor) Start(ctx context.Context) {
	// Start result processor
	m.wg.Add(1)
	go m.processResults(ctx)

	// Start pingers for each target
	for _, target := range m.config.Targets {
		m.wg.Add(1)
		go m.pingWorker(ctx, target)
	}

	// Start maintenance routines for long-term operation
	m.wg.Add(1)
	go m.maintenanceWorker(ctx)
}

func (m *Monitor) maintenanceWorker(ctx context.Context) {
	defer m.wg.Done()

	// Run maintenance every hour
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	// Run immediately on start
	m.performMaintenance()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.performMaintenance()
		}
	}
}

func (m *Monitor) performMaintenance() {
	log.Println("Running maintenance tasks...")

	// Aggregate hourly patterns for heatmap
	if err := m.aggregateHourlyPatterns(); err != nil {
		log.Printf("Failed to aggregate hourly patterns: %v", err)
	}

	// Archive old detailed data (keep raw data for 7 days, aggregated for 90 days)
	if err := m.archiveOldData(); err != nil {
		log.Printf("Failed to archive old data: %v", err)
	}

	log.Println("Maintenance complete")
}

func (m *Monitor) aggregateHourlyPatterns() error {
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
	_, err := m.db.Exec(query)
	return err
}

func (m *Monitor) archiveOldData() error {
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

	if _, err := m.db.Exec(archiveQuery); err != nil {
		return err
	}

	// Delete raw ping results older than 7 days (we keep aggregated data)
	deleteQuery := `DELETE FROM ping_results WHERE timestamp < datetime('now', '-7 days')`
	if _, err := m.db.Exec(deleteQuery); err != nil {
		return err
	}

	// Delete hourly patterns older than 90 days
	deletePatternQuery := `DELETE FROM hourly_patterns WHERE date < date('now', '-90 days')`
	if _, err := m.db.Exec(deletePatternQuery); err != nil {
		return err
	}

	// Vacuum to reclaim space (run occasionally)
	if time.Now().Day() == 1 { // Run on first day of month
		_, err := m.db.Exec("VACUUM")
		return err
	}

	return nil
}

func (m *Monitor) pingWorker(ctx context.Context, target string) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	// Immediate first ping
	m.performPing(target)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.performPing(target)
		}
	}
}

func (m *Monitor) performPing(target string) {
	result := PingResult{
		Timestamp: time.Now(),
		Target:    target,
	}

	// Platform-specific ping command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "1", "-w", fmt.Sprintf("%d", m.config.Timeout.Milliseconds()), target)
	} else {
		cmd = exec.Command("ping", "-c", "1", "-W", fmt.Sprintf("%d", int(m.config.Timeout.Seconds())), target)
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
	} else {
		result.Success = true
		result.RTT = parsePingOutput(string(output))
	}

	select {
	case m.results <- result:
	default:
		log.Printf("Result channel full, dropping result for %s", target)
	}
}

func parsePingOutput(output string) float64 {
	// Parse RTT from ping output
	// Linux/Mac: "time=XX.X ms"
	// Windows: "time=XXms" or "time<1ms"

	var patterns = []string{
		`time[=<]([0-9.]+)\s*ms`,
		`time[=<]([0-9.]+)ms`,
		`round-trip min/avg/max = [0-9.]+/([0-9.]+)/`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(output)
		if len(matches) > 1 {
			if rtt, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return rtt
			}
		}
	}

	return 0
}

func (m *Monitor) processResults(ctx context.Context) {
	defer m.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case result := <-m.results:
			if err := m.saveResult(result); err != nil {
				log.Printf("Failed to save result: %v", err)
			}
		}
	}
}

func (m *Monitor) saveResult(result PingResult) error {
	query := `
        INSERT INTO ping_results (timestamp, target, success, rtt_ms, error_message)
        VALUES (?, ?, ?, ?, ?)
    `
	_, err := m.db.Exec(query,
		result.Timestamp,
		result.Target,
		result.Success,
		result.RTT,
		result.ErrorMessage,
	)
	return err
}

func (m *Monitor) Close() {
	close(m.results)
	m.db.Close()
}

func (m *Monitor) Wait() {
	m.wg.Wait()
}

func initDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrent access
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")

	schema := `
    CREATE TABLE IF NOT EXISTS ping_results (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME NOT NULL,
        target TEXT NOT NULL,
        success BOOLEAN NOT NULL,
        rtt_ms REAL,
        error_message TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    CREATE INDEX IF NOT EXISTS idx_timestamp ON ping_results(timestamp);
    CREATE INDEX IF NOT EXISTS idx_target_timestamp ON ping_results(target, timestamp);

    CREATE TABLE IF NOT EXISTS outages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        target TEXT NOT NULL,
        start_time DATETIME NOT NULL,
        end_time DATETIME,
        duration_seconds INTEGER,
        checks_failed INTEGER
    );

    CREATE TABLE IF NOT EXISTS hourly_stats (
        hour DATETIME NOT NULL,
        target TEXT NOT NULL,
        total_pings INTEGER,
        successful_pings INTEGER,
        avg_rtt_ms REAL,
        max_rtt_ms REAL,
        min_rtt_ms REAL,
        p95_rtt_ms REAL,
        p99_rtt_ms REAL,
        packet_loss_percent REAL,
        PRIMARY KEY (hour, target)
    );

    -- New table for heatmap data (aggregated by hour of day)
    CREATE TABLE IF NOT EXISTS hourly_patterns (
        date DATE NOT NULL,
        hour INTEGER NOT NULL, -- 0-23
        target TEXT NOT NULL,
        total_pings INTEGER,
        failed_pings INTEGER,
        avg_rtt_ms REAL,
        max_rtt_ms REAL,
        failure_rate REAL,
        PRIMARY KEY (date, hour, target)
    );

    CREATE INDEX IF NOT EXISTS idx_hourly_patterns ON hourly_patterns(hour, target);
    `

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return db, nil
}

// Web server
func (m *Monitor) StartWebServer() {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/recent", m.handleRecent)
	mux.HandleFunc("/api/stats", m.handleStats)
	mux.HandleFunc("/api/outages", m.handleOutages)
	mux.HandleFunc("/api/heatmap", m.handleHeatmap)
	mux.HandleFunc("/api/patterns", m.handlePatterns)

	// Static files - serve static/ directory as webroot
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", m.config.Port), mux))
}

func (m *Monitor) handleRecent(w http.ResponseWriter, r *http.Request) {
	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil {
			hours = parsed
		}
	}

	query := `
        SELECT timestamp, target, success, rtt_ms, error_message
        FROM ping_results
        WHERE timestamp > datetime('now', '-' || ? || ' hours')
        ORDER BY timestamp DESC
        LIMIT 10000
    `

	rows, err := m.db.Query(query, hours)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []PingResult
	for rows.Next() {
		var r PingResult
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (m *Monitor) handleStats(w http.ResponseWriter, r *http.Request) {
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
        WHERE timestamp > datetime('now', '-24 hours')
        GROUP BY target
    `

	rows, err := m.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Stats struct {
		Target     string  `json:"target"`
		TotalPings int     `json:"total_pings"`
		Successful int     `json:"successful_pings"`
		AvgRTT     float64 `json:"avg_rtt"`
		MaxRTT     float64 `json:"max_rtt"`
		MinRTT     float64 `json:"min_rtt"`
		PacketLoss float64 `json:"packet_loss"`
	}

	var stats []Stats
	for rows.Next() {
		var s Stats
		err := rows.Scan(&s.Target, &s.TotalPings, &s.Successful,
			&s.AvgRTT, &s.MaxRTT, &s.MinRTT, &s.PacketLoss)
		if err != nil {
			continue
		}
		stats = append(stats, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (m *Monitor) handleOutages(w http.ResponseWriter, r *http.Request) {
	// Detect outages (consecutive failures)
	query := `
        WITH outage_groups AS (
            SELECT 
                target,
                timestamp,
                success,
                ROW_NUMBER() OVER (PARTITION BY target ORDER BY timestamp) -
                ROW_NUMBER() OVER (PARTITION BY target, success ORDER BY timestamp) as grp
            FROM ping_results
            WHERE timestamp > datetime('now', '-7 days')
        )
        SELECT 
            target,
            MIN(timestamp) as start_time,
            MAX(timestamp) as end_time,
            COUNT(*) as failed_checks
        FROM outage_groups
        WHERE success = 0
        GROUP BY target, grp
        HAVING COUNT(*) >= 3  -- At least 3 consecutive failures
        ORDER BY start_time DESC
        LIMIT 100
    `

	rows, err := m.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Outage struct {
		Target       string    `json:"target"`
		StartTime    time.Time `json:"start_time"`
		EndTime      time.Time `json:"end_time"`
		FailedChecks int       `json:"failed_checks"`
		Duration     string    `json:"duration"`
	}

	var outages []Outage
	for rows.Next() {
		var o Outage
		err := rows.Scan(&o.Target, &o.StartTime, &o.EndTime, &o.FailedChecks)
		if err != nil {
			continue
		}
		o.Duration = o.EndTime.Sub(o.StartTime).String()
		outages = append(outages, o)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(outages)
}

func (m *Monitor) handleHeatmap(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}

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

	rows, err := m.db.Query(query, days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

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

	var heatmapData []HeatmapPoint
	for rows.Next() {
		var h HeatmapPoint
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(heatmapData)
}

func (m *Monitor) handlePatterns(w http.ResponseWriter, r *http.Request) {
	// Get daily patterns for specific hour
	hour := r.URL.Query().Get("hour")
	if hour == "" {
		http.Error(w, "hour parameter required", http.StatusBadRequest)
		return
	}

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

	rows, err := m.db.Query(query, hour)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type PatternDetail struct {
		Date        string  `json:"date"`
		Target      string  `json:"target"`
		TotalPings  int     `json:"total_pings"`
		FailedPings int     `json:"failed_pings"`
		AvgRTT      float64 `json:"avg_rtt"`
		MaxRTT      float64 `json:"max_rtt"`
		FailureRate float64 `json:"failure_rate"`
	}

	var patterns []PatternDetail
	for rows.Next() {
		var p PatternDetail
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patterns)
}
