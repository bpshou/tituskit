# AGENTS.md

## Project overview

Go utility library (`github.com/bpshou/tituskit`). Two top-level trees: `core/` holds 9
independent infrastructure packages, `gin/resp` holds Gin response helpers. No application
entrypoint — this is a library consumed by other projects.

## Build & verify

```bash
go build ./...
go vet ./...
go test ./...
go test -race ./...     # rester / deploader / log / db 涉及并发，改这些包时带上
go test -cover ./...    # 检查改动是否被测试覆盖
gofmt -l .              # 期望输出为空
```

No Makefile, CI, or linter config exists. Standard Go toolchain only.

## Testing

**Rule: new functionality must be covered by unit tests.**
Every new exported function, method, branch of behavior, or bug fix must ship with unit
tests in the same package. Before calling any change done:

1. Run `go test -cover ./...` and inspect the package you touched.
2. If the new code is not covered, add tests — do not skip this step and do not rely on
   "it is covered indirectly".
3. A change that lowers a package's coverage is a regression: restore it, or state
   explicitly why the remaining lines are unreachable.

Test conventions:

- Table-driven tests (`tests := []struct{ name string; ... }`) with Chinese comments and
  failure messages, matching the existing files.
- Tests must be deterministic and hermetic: no database, network, desktop session, or
  wall-clock dependency. Use `httptest`, `t.TempDir()`, `t.Setenv()`, and injectable
  seams (`newDB`, `findCycle`, hooks) instead of the real thing.
- Anything that genuinely cannot be unit-tested (see the unreachable list in Gotchas)
  must be documented there rather than silently left untested.
- Never delete a defensive guard just to raise a coverage number.

## Structure

```
core/
  auth/       JWT generation & parsing (generic Jwt[T], HS256)
  config/     Viper config file loading
  crontab/    Cron scheduling (second-level precision) & expression parsing
  db/         GORM MySQL/PostgreSQL setup & gen code generation
  deploader/  Dependency-ordered module loading (concurrent topological load)
  log/        slog structured logging + OpenTelemetry trace integration
  notice/     Desktop notifications (beeep)
  rester/     HTTP client wrapper (resty v3)
  tmpl/       Go template engine with Sprig functions
gin/
  resp/       Gin response envelopes (Format A web / Format B OpenAI-compatible)
```

Packages are self-contained: no package in this repo imports another.
Comments are Simplified Chinese everywhere except `gin/resp`, which is English.

## Conventions

- **Constructor pattern:** most packages expose `New*()` (`NewCron()`, `New()`,
  `NewGenEngine()`, `NewJwt()`). Exceptions: `deploader.Load()` and `notice.Send*()` are
  the entrypoints, and `gin/resp` is a set of package-level helper functions.
- **Logging:** `log/slog` with context variants (`slog.ErrorContext`, ...). `core/log`
  wraps handlers to inject `trace`/`span` ids from the context.
- **Defaults with override:** sensible defaults provided, configurable via parameters.
- **Errors over exits:** a library must not call `log.Fatal`/`os.Exit` — it would kill the
  host process. Return `error` instead (this is why `tmpl` returns errors now).
- **Package-level singleton:** `rester` exposes `defaultClient` with convenience functions.

## Gotchas

- `core/db/generator.go` function is named `GromGen` (typo, not `GormGen`), kept for
  backward compatibility. It **panics** on failure: `gorm.io/gen`'s `Generator.Execute()`
  has no error return and panics internally, so there is no error to propagate.
  Its `path` argument is an output root — `"."` yields `./model` and `./query`.
- Test files: `core/auth/jwt_test.go`, `core/config/viper_test.go`,
  `core/crontab/cron_test.go`, `core/db/common_test.go`, `core/db/common_newdb_test.go`,
  `core/db/connection_test.go`, `core/db/generator_test.go`, `core/db/logger_test.go`,
  `core/deploader/deploader_test.go`, `core/log/slog_test.go`, `core/log/handler_test.go`,
  `core/log/tracer_test.go`, `core/notice/notice_test.go`,
  `core/rester/infra_resty_test.go`, `core/tmpl/gen_engine_test.go`,
  `gin/resp/response_test.go`. Every package has tests now.
- Coverage is 100% for config / crontab / deploader / notice / rester / tmpl / gin-resp.
  The lines still uncovered are structurally unreachable — keep them, do not delete a
  guard to raise the number:
  - `core/auth/jwt.go`: the `jwtToken == nil` and `!Valid` guards. golang-jwt returns a
    non-nil token only together with a nil error, and `Valid` is true in that case.
  - `core/log/tracer.go`: the `resource.New` / `stdouttrace.New` / `Shutdown` error
    branches — none of them fail with the options used here.
  - `core/db/generator.go`: `g.Execute()` needs a live database with tables; only the
    panic path (`GenerateAllTable` fails) is covered. The MySQL/PostgreSQL happy paths
    that require a real server are covered through the `newDB` seam instead.
- `notice` tests hit the real desktop notifier (integration-only); the error path is
  covered deterministically by clearing `PATH` so beeep cannot find its helper command.
- OpenTelemetry `InitTracer` uses the stdout exporter with `io.Discard` — spans are
  discarded, only `trace_id`/`span_id` show up in logs. It returns a single shutdown
  function, not `(func(), error)`.
- HTTP client defaults: 300s timeout, 3 retries, 3s wait between retries.
- `rester.Client` applies its configuration exactly once, behind a `sync.Once` (a
  zero-value `Client` also lazily creates its resty client there; `New()` creates it
  eagerly): configure `Timeout`/`RetryCount`/`RetryWaitTime`/`Debug` **before the first
  request**, they are ignored afterwards. resty's `OnError`/`OnInvalid`/`OnPanic`
  **append**, so re-configuring per request (the old behavior) leaked hooks and
  duplicated every error log line. `Client` contains a `sync.Once` and must not be copied.
- `core/log.New` falls back to `os.Stdout` when `Config.IoWriter` is nil. Passing nil
  straight to `slog` panics on the first write.
- `tmpl.AddTemplate` returns `error` (never exits). After a failed `Parse`, that template
  name stays unusable because `text/template`'s `New(name)` already registered it.
- DB pool defaults: 10 idle, 100 max open connections, 1h max lifetime. Both `NewMySQL`
  and `NewPostgreSQL` accept a nil `*gorm.Config` and inject the slog-based GORM logger
  when `config.Logger` is nil.
- `core/auth` pins HS256 via `jwt.WithValidMethods` and validates `Issuer` when it is
  non-empty. Without that, HS384/HS512 tokens and tokens from a foreign issuer were
  accepted. `Jwt[T]` no longer has a `Claims` field (it was never read).
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
- `deploader.Load` panics on duplicate module names, unknown dependencies, and dependency
  cycles (including self-dependency). Cycles used to deadlock forever, because every
  goroutine waited on a `done` channel that could never close.
- `gorm.io/plugin/dbresolver` is pinned to **v1.6.2** in `go.mod`. `gorm.io/gen@v0.3.29`
  requires v1.5.3, which does not compile against `gorm.io/gorm v1.31.2`
  (`undefined: gorm.Stmt`). Do not let `go mod tidy` drop this override.
