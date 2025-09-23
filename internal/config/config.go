package config

import (
	"fmt"
	"time"
)

// Config holds all configuration for the network monitor
type Config struct {
	Targets      []string
	Interval     time.Duration
	Timeout      time.Duration
	DatabasePath string
	Port         int
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Targets) == 0 {
		return fmt.Errorf("at least one target must be specified")
	}
	if c.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.DatabasePath == "" {
		return fmt.Errorf("database path cannot be empty")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}
