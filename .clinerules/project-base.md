# Network Monitor Project Summary

## Overview

A focused Go-based network connectivity monitoring solution designed for long-term ISP issue documentation and pattern detection.

## Technology Stack

- **Language**: Go 1.21
- **Database**: SQLite (modernc.org/sqlite driver)
- **Frontend**: HTML/CSS/JavaScript with D3.js for visualizations
- **Chart Generation**: go-chart/v2 library for static PNG reports
- **Build System**: Standard Go build with bash scripts

## Architecture

- **Main Application**: Single Go binary (`main.go`) with embedded static web assets
- **Database Layer**: SQLite with multiple tables for ping results, outages, and aggregated patterns
- **Web Interface**: REST API serving data to D3.js-based dashboard
- **Report Generation**: Separate module (`report.go`) for static PNG chart generation

## Key Components

- **main.go**: Core monitoring logic, ping workers, web server, database management
- **report.go**: Chart generation using go-chart library
- **static/index.html**: Web dashboard with real-time visualizations
- **Database Tables**:
  - `ping_results`: Raw ping data (7-day retention)
  - `hourly_patterns`: Aggregated patterns for heatmap (90-day retention)
  - `outages`: Detected connectivity failures
  - `hourly_stats`: Statistical aggregations

## Features

- **Continuous Monitoring**: Configurable ping intervals to multiple targets
- **Real-time Dashboard**: Web interface at localhost:8080 with live charts
- **Pattern Detection**: 24-hour heatmap overlay showing issue patterns across days
- **Outage Tracking**: Automatic detection of connectivity failures (3+ consecutive pings)
- **Static Reports**: PNG chart generation for ISP evidence documentation
- **Data Management**: Automatic maintenance with configurable retention periods

## Configuration

- **Targets**: Comma-separated IP addresses (default: Google DNS, Cloudflare, OpenDNS)
- **Interval**: Ping frequency (default: 1 second)
- **Timeout**: Ping timeout (default: 5 seconds)
- **Database**: SQLite file path (default: network_monitor.db)
- **Port**: Web server port (default: 8080)

## Deployment

- **Build**: Simple `go build` command
- **Run**: Executable binary with optional flags
- **Service**: Can be configured as macOS launchd service or systemd service
- **Resource Usage**: Low CPU/memory footprint suitable for continuous operation

## Data Flow

1. Ping workers continuously test connectivity to configured targets
2. Results stored in SQLite database with timestamps
3. Hourly maintenance aggregates data for heatmap visualization
4. Web API serves data to frontend dashboard
5. Optional static report generation for documentation

## Use Cases

- ISP connectivity monitoring and issue documentation
- Network troubleshooting and pattern analysis
- Long-term connectivity logging for service agreements
- Real-time network status dashboard

## Development Guidelines

### Git Practices

- **NEVER commit to main/master branch directly** - use feature branches
- Keep commits small and focused with clear, descriptive messages
- Rebase branches before merging to maintain clean history
- Use pull requests for code reviews and discussions

### Build System

- Use Taskfile.yml for build management instead of Makefiles
- Required tasks: `build`, `build-linux`, `build-ci`, `test`, `test-ci`, `lint`
- Build tasks must depend on test and lint tasks
- Build artifacts placed in `build/` directory
- GitHub Actions CI uses build-ci task for automated testing and linting

### Code Quality

- **Formatting**: Use `goimports -w .` (not `gofmt`) for code formatting and import management
- **Linting**: Use golangci-lint with `.golangci.yml` configuration
- **Error Handling**: Use `errors.Is()` and `errors.As()` for robust error checking
- **Testing**: Include basic unit tests for critical functionality
- **Dependencies**: Prefer standard library; justify third-party additions

### Modern Tools

- **Search**: Use `rg` (ripgrep) instead of `grep` for faster, smarter searching
- **File Finding**: Use `fd` instead of `find` for better performance and `.gitignore` respect
- **Code Analysis**: Use `gofuncs` tool for exploring Go function structures

### Project Validation

- Use `validate-docs` tool to ensure standard project structure compliance
- Validates directory structure, required files, and build configuration

## Development Notes

- Cross-platform ping implementation (Windows/Mac/Linux support)
- Embedded static files using Go's `embed` package
- RESTful API design with JSON responses
- D3.js for interactive data visualizations
- SQLite WAL mode for concurrent access
- LLM-shared submodule provides development tools and guidelines
