# Repository Guidelines

## Project Structure & Module Organization

Source code centers on `main.go` and the `internal/` tree. Key packages include `internal/monitor` for orchestrating workers, `internal/ping` for ICMP sampling, `internal/database` for SQLite access, and `internal/web` for the HTTP dashboard. Static web assets live in `static/`, build artifacts are written to `build/`, and generated reports land in `reports/` when created. Shared helper utilities reside under `llm-shared/`.

## Build, Test, and Development Commands

Use `task build` for a full build with linting and unit tests, or `task build-linux` when cross-compiling for Linux. Run `task dev` (or `go run . --dev`) to launch the dashboard with live assets. `task lint` enforces format and vetting, while `task clean` resets the `build/` directory. For containerized runs, prefer `docker-compose up --build` and stop with `docker-compose down`.

## Coding Style & Naming Conventions

Target Go 1.21. Run `goimports -w .` before commits; it formats code and fixes imports. Keep packages lower_snake_case and exported symbols in PascalCase with concise doc comments where behavior is non-obvious. Limit files to focused responsibilities; prefer splitting helpers into the existing `internal/*` domains rather than adding new top-level folders.

## Testing Guidelines

Unit tests live alongside code (e.g., `internal/ping/ping_test.go`). Execute `task test` or `go test ./...` before pushing; CI expects clean runs. For coverage-sensitive work, run `task test-ci` to mirror pipeline settings. Name tests with the behavior under test (`TestWorkerHandlesTimeout`) and favor table-driven cases to clarify target/interval permutations.

## Commit & Pull Request Guidelines

Follow Conventional Commits (`feat:`, `fix:`, `refactor:`) as shown in `git log`. Keep PRs focused, include a brief summary of monitoring or UI impacts, and link tracking issues. Add screenshots for UI-facing changes or mention CLI output for report generation tweaks. Confirm docs stay current when touch points like `Taskfile.yml` or `static/` change.

## Operations & Configuration Notes

Runtime configuration is pulled from flags and files under `config/`. SQLite data defaults to `network_monitor.db`; keep it out of commits. When generating reports via `task report`, ensure `build/network-monitor` exists. Coordinate schema or retention tweaks with the reporting charts so PNG exports remain accurate.
