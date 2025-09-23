package ping

import (
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"network-monitor/internal/models"
)

// Pinger implements the Pinger interface
type Pinger struct{}

// New creates a new Pinger
func New() *Pinger {
	return &Pinger{}
}

// Ping executes a ping to the target and returns the result
func (p *Pinger) Ping(target string, timeout time.Duration) (models.PingResult, error) {
	result := models.PingResult{
		Timestamp: time.Now(),
		Target:    target,
	}

	// Platform-specific ping command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "1", "-w", strconv.Itoa(int(timeout.Milliseconds())), target)
	} else {
		cmd = exec.Command("ping", "-c", "1", "-W", strconv.Itoa(int(timeout.Seconds())), target)
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		result.RTT = parsePingOutput(string(output)) // Try to parse even on error
	} else {
		result.Success = true
		result.RTT = parsePingOutput(string(output))
	}

	// Debug: log output if parsing failed
	if result.Success && result.RTT == 0 {
		// This is for debugging - in production this would be removed
		_ = output // avoid unused variable warning
	}

	return result, nil
}

// parsePingOutput parses RTT from ping output
func parsePingOutput(output string) float64 {
	// Parse RTT from ping output
	// macOS: "time=XX.X ms" or "round-trip min/avg/max/stddev = X.X/X.X/X.X/X.X ms"
	// Linux: "time=XX.X ms" or "round-trip min/avg/max = X.X/X.X/X.X ms"
	// Windows: "time=XXms" or "time<1ms"

	var patterns = []string{
		`time=([0-9.]+)ms`,    // Windows: time=44ms or time=44.5ms (not time<1ms)
		`time=([0-9.]+)\s*ms`, // macOS/Linux individual: time=44.347 ms
		`round-trip min/avg/max/stddev = [0-9.]+/([0-9.]+)/[0-9.]+/[0-9.]+\s*ms`, // macOS summary: round-trip min/avg/max/stddev = 44.347/44.347/44.347/0.000 ms
		`round-trip min/avg/max = [0-9.]+/([0-9.]+)/[0-9.]+\s*ms`,                // Linux summary: round-trip min/avg/max = 44.347/44.347/44.347 ms
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
