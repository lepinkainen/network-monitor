package ping

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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
		Timestamp:  time.Now(),
		Target:     target,
		PacketLoss: 100,
	}

	normalizedTimeout := normalizeTimeout(timeout)
	contextTimeout := normalizedTimeout + 500*time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", buildPingArgs(target, normalizedTimeout)...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if ctx.Err() == context.DeadlineExceeded {
		result.ErrorMessage = fmt.Sprintf("ping timed out after %s", normalizedTimeout)
		return result, ctx.Err()
	}

	if err != nil {
		result.ErrorMessage = strings.TrimSpace(outputStr)
		if result.ErrorMessage == "" {
			result.ErrorMessage = err.Error()
		}
		return result, err
	}

	rtt := parsePingOutput(outputStr)
	if rtt <= 0 {
		result.ErrorMessage = "unable to parse round-trip time"
		return result, fmt.Errorf("unable to parse ping output: %s", strings.TrimSpace(outputStr))
	}

	result.Success = true
	result.PacketLoss = 0
	result.RTT = rtt
	return result, nil
}

func normalizeTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return time.Second
	}
	return timeout
}

func buildPingArgs(target string, timeout time.Duration) []string {
	switch runtime.GOOS {
	case "windows":
		ms := int(timeout / time.Millisecond)
		if ms < 1 {
			ms = 1
		}
		return []string{"-n", "1", "-w", strconv.Itoa(ms), target}
	case "darwin":
		ms := int(timeout / time.Millisecond)
		if ms < 1 {
			ms = 1
		}
		return []string{"-n", "-c", "1", "-W", strconv.Itoa(ms), target}
	default:
		secs := int((timeout + time.Second - 1) / time.Second)
		if secs < 1 {
			secs = 1
		}
		return []string{"-n", "-c", "1", "-W", strconv.Itoa(secs), target}
	}
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
