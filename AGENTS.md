# AGENTS.md

## Project overview

Go utility library (`github.com/bpshou/tituskit`). Single `core/` directory with 8 independent packages. No application entrypoint — this is a library consumed by other projects.

## Build & verify

```bash
go build ./...
go vet ./...
go test ./...
```

No Makefile, CI, or linter config exists. Standard Go toolchain only.

## Structure

```
core/
  auth/       JWT generation & parsing (generic Jwt[T])
  config/     Viper config file loading
  crontab/    Cron scheduling (second-level precision) & expression parsing
  db/         GORM MySQL/PostgreSQL setup & gen code generation
  log/        slog structured logging + OpenTelemetry trace integration
  notice/     Desktop notifications (beeep)
  rester/     HTTP client wrapper (resty v3)
  tmpl/       Go template engine with Sprig functions
```

Each package is self-contained. No cross-package imports within `core/`.

## Conventions

- **Constructor pattern:** Every package uses `New*()` (e.g., `NewCron()`, `New()`, `NewGenEngine()`).
- **Chinese comments:** All code comments are in Simplified Chinese.
- **Logging:** All packages use `log/slog` with context variants (`slog.ErrorContext`, etc.).
- **Defaults with override:** Sensible defaults provided, configurable via parameters.
- **Package-level singleton:** `rester` exposes `defaultClient` with convenience functions.

## Gotchas

- `core/db/generator.go` function is named `GromGen` (typo, not `GormGen`).
- Tests are minimal — only `core/notice/notice_test.go` exists. Most packages have zero tests.
- `notice` tests trigger real desktop notifications (integration-only).
- OpenTelemetry `InitTracer` uses stdout exporter — spans are discarded, not sent to a collector.
- HTTP client defaults: 300s timeout, 3 retries, 3s wait between retries.
- DB pool defaults: 10 idle, 100 max open connections, 1h max lifetime.
- Cron uses second-level precision (6-field expressions), not the standard 5-field.
