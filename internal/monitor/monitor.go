package monitor

import (
	"context"
	"log"
	"sync"

	"network-monitor/internal/config"
	"network-monitor/internal/database"
	"network-monitor/internal/models"
	"network-monitor/internal/ping"
)

// Monitor coordinates ping monitoring operations
type Monitor struct {
	config  config.Config
	db      *database.DB
	pinger  *ping.Pinger
	results chan models.PingResult
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// New creates a new Monitor
func New(cfg config.Config, db *database.DB, pinger *ping.Pinger) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Monitor{
		config:  cfg,
		db:      db,
		pinger:  pinger,
		results: make(chan models.PingResult, 100),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start begins the monitoring process
func (m *Monitor) Start() error {
	log.Printf("Starting monitor with %d targets", len(m.config.Targets))

	// Start result processor
	m.wg.Add(1)
	go m.processResults()

	// Start pingers for each target
	for _, target := range m.config.Targets {
		m.wg.Add(1)
		go m.pingWorker(target)
	}

	// Start maintenance routines
	m.wg.Add(1)
	go m.maintenanceWorker()

	log.Printf("Monitor started. Pinging %v every %v", m.config.Targets, m.config.Interval)
	return nil
}

// Stop gracefully stops the monitor
func (m *Monitor) Stop() {
	log.Println("Stopping monitor...")
	m.cancel()
	close(m.results)
}

// Wait blocks until all goroutines finish
func (m *Monitor) Wait() {
	m.wg.Wait()
	log.Println("Monitor stopped")
}
