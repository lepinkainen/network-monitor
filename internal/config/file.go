package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "config/config.yml"

// fileConfig represents the YAML configuration structure.
type fileConfig struct {
	Targets      []string `yaml:"targets"`
	Interval     string   `yaml:"interval"`
	Timeout      string   `yaml:"timeout"`
	DatabasePath string   `yaml:"database_path"`
	Port         *int     `yaml:"port"`
	DevMode      *bool    `yaml:"dev_mode"`
}

func mergeConfigFile(base Config, path string) (Config, error) {
	if path == "" {
		path = defaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return base, nil
		}
		return Config{}, fmt.Errorf("read config file %q: %w", path, err)
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file %q: %w", path, err)
	}

	if len(cfg.Targets) > 0 {
		cleanedTargets := make([]string, 0, len(cfg.Targets))
		for _, target := range cfg.Targets {
			trimmed := strings.TrimSpace(target)
			if trimmed != "" {
				cleanedTargets = append(cleanedTargets, trimmed)
			}
		}
		if len(cleanedTargets) > 0 {
			base.Targets = cleanedTargets
		}
	}

	if cfg.Interval != "" {
		duration, err := time.ParseDuration(cfg.Interval)
		if err != nil {
			return Config{}, fmt.Errorf("invalid interval duration %q: %w", cfg.Interval, err)
		}
		base.Interval = duration
	}

	if cfg.Timeout != "" {
		duration, err := time.ParseDuration(cfg.Timeout)
		if err != nil {
			return Config{}, fmt.Errorf("invalid timeout duration %q: %w", cfg.Timeout, err)
		}
		base.Timeout = duration
	}

	if cfg.DatabasePath != "" {
		base.DatabasePath = cfg.DatabasePath
	}

	if cfg.Port != nil {
		base.Port = *cfg.Port
	}

	if cfg.DevMode != nil {
		base.DevMode = *cfg.DevMode
	}

	return base, nil
}
