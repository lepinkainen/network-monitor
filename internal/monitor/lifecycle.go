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

	// Run immediately on start
	m.performMaintenance()

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
	}

	// Archive old detailed data (keep raw data for 7 days, aggregated for 90 days)
	if err := m.db.ArchiveOldData(); err != nil {
		log.Printf("Failed to archive old data: %v", err)
	}

	log.Println("Maintenance complete")
}
