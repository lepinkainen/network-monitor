package config

import (
	"flag"
	"strings"
	"time"
)

// ParseFlags parses command-line flags and returns a Config
func ParseFlags() Config {
	var (
		interval = flag.Duration("interval", 1*time.Second, "Ping interval")
		timeout  = flag.Duration("timeout", 5*time.Second, "Ping timeout")
		dbPath   = flag.String("db", "network_monitor.db", "Database path")
		port     = flag.Int("port", 8080, "Web server port")
		targets  = flag.String("targets", "8.8.8.8,1.1.1.1,208.67.222.222", "Comma-separated ping targets")
	)
	flag.Parse()

	return Config{
		Targets:      strings.Split(*targets, ","),
		Interval:     *interval,
		Timeout:      *timeout,
		DatabasePath: *dbPath,
		Port:         *port,
	}
}
