package monitor

import (
	"log"
	"time"
)

// maintenanceWorker runs periodic maintenance tasks
func (m *Monitor) maintenanceWorker() {
	defer m.wg.Done()

	// Run maintenance every hour
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	// Wait 60 seconds before first maintenance to avoid startup race conditions
	startupDelay := time.NewTimer(60 * time.Second)
	defer startupDelay.Stop()

	// Wait for startup delay before running first maintenance
	select {
	case <-m.ctx.Done():
		return
	case <-startupDelay.C:
		m.performMaintenance()
	}

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performMaintenance()
		}
	}
}

// performMaintenance runs maintenance tasks
func (m *Monitor) performMaintenance() {
	log.Println("Running maintenance tasks...")

	// Aggregate hourly patterns for heatmap
	if err := m.db.AggregateHourlyPatterns(); err != nil {
		log.Printf("Failed to aggregate hourly patterns: %v", err)
	} else {
		log.Println("Successfully aggregated hourly patterns")
	}

	// Archive old detailed data (keep raw data for 7 days, aggregated for 90 days)
	if err := m.db.ArchiveOldData(); err != nil {
		log.Printf("Failed to archive old data: %v", err)
	} else {
		log.Println("Successfully archived old data")
	}

	log.Println("Maintenance complete")
}
