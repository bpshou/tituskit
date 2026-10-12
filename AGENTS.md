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
- Tests are minimal — only `core/crontab/cron_test.go`, `core/deploader/deploader_test.go` and `core/notice/notice_test.go` exist. Most packages have zero tests.
- `notice` tests trigger real desktop notifications (integration-only).
- OpenTelemetry `InitTracer` uses stdout exporter — spans are discarded, not sent to a collector.
- HTTP client defaults: 300s timeout, 3 retries, 3s wait between retries.
- DB pool defaults: 10 idle, 100 max open connections, 1h max lifetime.
- Cron uses second-level precision (6-field expressions), not the standard 5-field.
- `crontab.NextTimeListN` / `NextDateListN` return `ErrNoNextTime` for expressions that
  are syntactically valid but can never fire (e.g. `0 0 0 31 2 *`). Without that check
  `robfig/cron`'s `Schedule.Next` returns the zero `time.Time` and the list silently
  fills with `0001-01-01`. Use `errors.Is(err, crontab.ErrNoNextTime)` to distinguish
  this from a syntax error.
- `crontab.NextRun` returns `ok=false` for `EntryID(0)`: it is the zero value of
  `cron.EntryID` and no registered job ever has it (real ids start at 1). This is a
  documented contract rather than a behavior fix — `Cron.Entry` already returns an
  all-zero `Entry` for unknown ids, so the result would be `ok=false` either way.
- `gorm.io/plugin/dbresolver` is pinned to **v1.6.2** in `go.mod`. `gorm.io/gen@v0.3.29`
  requires v1.5.3, which does not compile against `gorm.io/gorm v1.31.2`
  (`undefined: gorm.Stmt`). Do not let `go mod tidy` drop this override.
- `core/db/postgresql.go` is not gofmt-clean (missing trailing newline) — pre-existing,
  unrelated to current work; `gofmt -l .` will list it.
