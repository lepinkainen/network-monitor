package monitor

import (
	"log"
	"time"
)

// pingWorker continuously pings a target at the configured interval
func (m *Monitor) pingWorker(target string) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	// Immediate first ping
	m.performPing(target)

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performPing(target)
		}
	}
}

// performPing executes a single ping and sends the result to the results channel
func (m *Monitor) performPing(target string) {
	result, err := m.pinger.Ping(target, m.config.Timeout)
	if err != nil {
		log.Printf("Failed to ping %s: %v", target, err)
		return
	}

	select {
	case m.results <- result:
	default:
		log.Printf("Result channel full, dropping result for %s", target)
	}
}

// processResults processes ping results from the results channel
func (m *Monitor) processResults() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case result := <-m.results:
			if err := m.db.SaveResult(result); err != nil {
				log.Printf("Failed to save result: %v", err)
			}
		}
	}
}
