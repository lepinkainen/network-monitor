package config

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

// ParseFlags parses command-line flags and returns a Config
func ParseFlags() (Config, error) {
	var (
		interval = flag.Duration("interval", 1*time.Second, "Ping interval")
		timeout  = flag.Duration("timeout", 5*time.Second, "Ping timeout")
		dbPath   = flag.String("db", "network_monitor.db", "Database path")
		port     = flag.Int("port", 8080, "Web server port")
		targets  = flag.String("targets", "8.8.8.8,1.1.1.1,208.67.222.222,192.168.1.1", "Comma-separated ping targets")
		devMode  = flag.Bool("dev", false, "Enable development mode (live static file editing)")
		cfgPath  = flag.String("config", "", "Path to YAML configuration file (optional)")
	)
	flag.Parse()

	baseConfig := Config{
		Targets:      splitTargets(*targets),
		Interval:     *interval,
		Timeout:      *timeout,
		DatabasePath: *dbPath,
		Port:         *port,
		DevMode:      *devMode,
	}

	mergedConfig, err := mergeConfigFile(baseConfig, *cfgPath)
	if err != nil {
		return Config{}, fmt.Errorf("load configuration: %w", err)
	}

	return mergedConfig, nil
}

func splitTargets(raw string) []string {
	parts := strings.Split(raw, ",")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}
