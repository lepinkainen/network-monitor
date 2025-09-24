# CLAUDE.md

Network Connectivity Monitor - AI Agent Guidelines

## Project Overview

A Go-based network monitoring tool that performs continuous ping tests to detect ISP issues and connectivity patterns. Features intelligent data retention, pattern detection via heatmaps, and web-based visualization.

**Key Purpose**: Long-term ISP issue documentation with compelling visual evidence.

## Architecture & Entry Points

- **Main Entry**: `cmd/monitor/main.go` - Orchestrates all components with graceful shutdown
- **Internal Structure**: Clean separation via `internal/` packages
- **Static Assets**: Embedded via `//go:embed static/*` in main.go
- **Database**: SQLite with WAL mode for concurrent access

### Critical Components

```plain
internal/
├── config/     - CLI flags and validation (config.go, flags.go)
├── database/   - SQLite operations, schema, maintenance (db.go, queries.go)
├── models/     - Data structures (ping.go, stats.go, types.go)
├── monitor/    - Worker orchestration and lifecycle (monitor.go, worker.go)
├── ping/       - Cross-platform ping implementation
├── report/     - PNG chart generation using go-chart/v2
└── web/        - HTTP server and REST API (handlers.go, server.go)
```

## Database Schema Strategy

**Smart Retention Pattern**:

- `ping_results`: Raw data (7-day retention)
- `hourly_patterns`: Aggregated for heatmap (90-day retention)
- `outages`: Detected failures (permanent)
- `hourly_stats`: Statistical summaries

**Key Insight**: Maintenance runs hourly via `internal/database/maintenance.go` - automatic data aggregation and cleanup.

## Build & Development Workflow

### Essential Commands

```bash
task build          # Build after tests+lint (required for deployment)
task dev           # Development server (go run ./cmd/monitor)
task test          # Run all tests
task lint          # goimports + vet + golangci-lint
task build-linux   # Cross-compile for Linux deployment
```

### Pre-commit Requirements

- **Always run**: `task build` before considering changes complete
- **Format**: Uses `goimports -w .` (NOT gofmt) for imports management
- **Linting**: golangci-lint with `.golangci.yml` config

## Project-Specific Conventions

### Error Handling Pattern

```go
// Preferred throughout codebase
if errors.Is(err, database.ErrOutageExists) {
    // handle specifically
}
```

### Configuration Philosophy

- CLI flags via `internal/config/flags.go`
- Validation in separate `config.Validate()` method
- Defaults optimized for home ISP monitoring

### Ping Implementation Detail

- Cross-platform: Windows/Mac/Linux support in `internal/ping/ping.go`
- **Outage Detection**: 5+ failures in any 10 consecutive pings
- Uses OS-native ping (not raw sockets) for reliability

## Web Interface Integration

### API Patterns

- RESTful JSON endpoints in `internal/web/handlers.go`
- Real-time data serving for D3.js frontend
- **Key Route**: `/api/data` powers the heatmap visualization

### Static Assets

- Single `static/index.html` with embedded D3.js
- **Pattern**: All static files embedded at compile time
- No build step for frontend - vanilla HTML/JS/CSS

## Testing & CI/CD

### Test Strategy

- `*_test.go` files for critical functionality only
- **CI Pattern**: Separate test/lint/build jobs in GitHub Actions
- Uses `task test-ci` for coverage reporting

### Docker Deployment

- `Dockerfile` + `docker-compose.yml` for containerization
- **Volume Pattern**: `./data:/app/data` for database persistence
- Health checks via web interface availability

## Integration Points

### llm-shared Submodule

- Development tools in `llm-shared/utils/`
- **Key Tool**: `gofuncs.go` for function analysis
- Project validation via `validate-docs.go`

### External Dependencies

- **Database**: `modernc.org/sqlite` (pure Go SQLite)
- **Charts**: `github.com/wcharczuk/go-chart/v2` for PNG report generation
- **Minimal Dependencies**: Prefers standard library

## ISP Documentation Workflow

### Data Collection Strategy

1. **Hour 1**: Basic connectivity data
2. **Day 1**: Initial pattern recognition
3. **Week 1**: Clear time-of-day patterns
4. **Month 1**: Compelling evidence for ISP discussions

### Report Generation

- PNG charts via `internal/report/` package
- **Visual Evidence**: Heatmap screenshots after 1-2 weeks most effective
- Export capability for CSV data analysis

## Development Notes

- **Deployment Target**: Single-user, private monitoring (not SaaS)
- **Service Integration**: Includes launchd/systemd service examples in README
- **Resource Efficient**: <1% CPU, 20-50MB RAM with default settings
- **Cross-Platform**: Full macOS/Linux/Windows support with OS-specific deployment guides
