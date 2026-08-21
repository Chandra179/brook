# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Go modular monolith skeleton. Module `brook`, Go `1.26.5`. One binary, domain modules as Go packages — split to microservices later, not before.

## Commands

```bash
make run              # go run ./cmd/example/
make test             # go test -short -race -count=1 ./...
make lint             # golangci-lint run  (v2 config in .golangci.yml)
make vendor           # go mod tidy && go mod vendor
make swag             # swag init -g cmd/example/main.go -o docs
make mocks            # go tool mockery
make up / down        # docker compose up/down
make modernize        # go fix ./...
make align            # fieldalignment -fix ./...
make re               # scripts/rename-module.sh example
make ci               # act workflow_dispatch  (runs GitHub Actions locally via act)
make migrate-up       # goose -dir store/migrations postgres "$POSTGRES_DSN" up
make migrate-down     # goose -dir store/migrations postgres "$POSTGRES_DSN" down
make migrate-create name=<name>  # goose -dir store/migrations create <name> sql
```

To run a single test: `go test -run TestName ./modules/example/...`.

Real CI is `.github/workflows/ci.yml` (go mod verify → golangci-lint → test → build → docker build).

## Architecture

- **Framework**: Gin. Handlers are `gin.HandlerFunc` methods on a module's unexported `*dependencies` struct.
- **Logger**: `go.uber.org/zap` used directly (no wrapper). Built via `logger.NewLogger(appEnvironment)` in `logger/logger.go` — `zap.NewDevelopment()` for `dev` (console, Debug+, stacktrace from Warn), `zap.NewProduction()` otherwise (JSON, Info+, stacktrace from Error). Not handled in `config/`.
- **gRPC** dependency present (for `middleware.RequestIDUnaryInterceptor`), but no gRPC server is wired in the entrypoint.
- **No global state**; deps injected via constructor, no `init()`.
- **Continuous profiling**: Pyroscope push SDK, gated by `profiling.enabled` in config. This repo only knows Pyroscope's address, not how/where it runs — the collector is owned and operated elsewhere. The root `docker-compose.yml`'s `brook` service reaches it via `http://host.docker.internal:4040` (with `extra_hosts: host-gateway`, since the collector isn't in this compose file's network); plain `make run` reaches it via `http://localhost:4040` (`config/config_dev.yaml`). Don't add a Pyroscope/Grafana service to this repo's compose file or Makefile — that's infra this repo shouldn't own.
- **Tracing**: OpenTelemetry SDK, gated by `tracing.enabled`, exporting spans via OTLP/gRPC (`tracing/tracing.go`'s `NewProvider`) — same address-only rule as Pyroscope: this repo only knows the OTLP endpoint (`tracing.otlp_endpoint` / `OTEL_EXPORTER_OTLP_ENDPOINT`), the collector (Grafana Alloy, forwarding to Tempo) is owned externally. `otelgin.Middleware` starts a span per request and is always registered (no-op when disabled); `middleware.RequestID` reuses that span's trace ID as the request ID when no `X-Request-ID` header is supplied, so logs/traces/responses share one ID.
- **Metrics**: `github.com/prometheus/client_golang` exposition format at `/metrics`, scraped by Alloy — no Prometheus server in this repo. Metric vectors live on a `*prometheus.Registry` constructed once in `server/server.go` and injected into `middleware.NewDependencies` (no global registry, no `init()`), registered by `middleware.Metrics()`.
- **Persistence**: `pgx`/`pgxpool` (not `sqlx`/`lib/pq`). `store/postgres.go` exports only `NewPool(ctx, config.PostgresConfig) (*pgxpool.Pool, error)` — shared infra, like `logger/`, with no knowledge of module domain types. Each module defines the `Store` interface it needs and implements it against the pool in its own `store.go` (see `modules/example/store.go`) — persistence is domain logic per `modules/README.md`, so a future DB swap is contained to one module's store file, not scattered. Migrations are goose SQL files in `store/migrations/`, applied via `make migrate-up` — never auto-run at server startup. `goose` itself is **not** a go.mod dependency (its driver-per-database footprint is large); it's installed on demand via `go install` in the Makefile, same treatment as `golangci-lint`/`swag`/`act`.

## Layout

| Path | Role |
|------|------|
| `cmd/example/main.go` | Entrypoint → `server.RunHttpServer()` |
| `server/server.go` | Assembles Gin router, registers middleware/routes, graceful shutdown |
| `modules/<name>/` | Flat domain module package |
| `middleware/` | Gin middleware + gRPC interceptor. See `middleware/README.md` |
| `config/` | Shared YAML loader + `config_prd.yaml` / `config_dev.yaml` |
| `logger/` | zap logger constructor |
| `tracing/` | OTel `TracerProvider` constructor (OTLP/gRPC exporter) |
| `docs/` | Generated swagger output (do not hand-edit) |
| `store/` | Shared pgx pool constructor (`postgres.go`) + goose migrations (`migrations/`) |

## Config & env

- `APP_ENVIRONMENT` selects the config file: `dev` → `config/config_dev.yaml`, anything else (incl. unset) → `config/config_prd.yaml`. Load with `config.Load(path)` (`config/config.go`).
- Shared config is **flat** (`http`, `logger`, `middleware`, `profiling`) — not nested per-module. Modules receive only what they need via constructor args.
- `PYROSCOPE_SERVER_ADDRESS` env var overrides `profiling.server_address` from the YAML if set.
- `OTEL_EXPORTER_OTLP_ENDPOINT` env var overrides `tracing.otlp_endpoint` from the YAML if set (same override pattern as Pyroscope; also the OTel spec's own conventional env var name).
- `POSTGRES_DSN` env var overrides `postgres.dsn` from the YAML if set — prod leaves the YAML value blank and expects this at deploy time (same pattern as Pyroscope).

## Module pattern (`modules/<name>/`)

Flat package, no `internal/` sub-packages. Wires deps in `dependencies.go` via `NewDependencies(&XConfig{...})` returning an unexported `*dependencies`. Handlers are methods on `*dependencies` registered directly in `server/server.go` (e.g. `exampleDeps.HandleExample` — there is no `mod.Handle` helper). Validation via `c.ShouldBindJSON(&req)` + `binding` struct tags inside handlers (no separate validation middleware).

Reference module: `modules/example/` (files: `dependencies.go`, `types.go`, `service.go`, `store.go`, `handler.go`, `business_error.go`, `constant.go`; no separate `config.go` — no config section needed since shared config is flat). `store.go` declares the module's own `Store` interface and its Postgres-backed implementation together, constructed with `NewPostgresStore(pool)` and wired in from `server/server.go` — copy this shape for any module that needs persistence. `service.go` declares the module's `Service` interface (what it provides to other modules, as opposed to `Store`, which is what it requires) alongside `*dependencies`' implementation of it — see `modules/README.md` for the cross-module communication pattern (`modules/foo/` calling `modules/example/` via `example.Service`). `business_error.go` holds the module's own domain sentinels (plain `errors.New(...)` values, no non-stdlib imports) — the handler checks `errors.Is` against them itself and picks the HTTP status, since HTTP is a transport concern the module has no business knowing about.

Full required/optional file table and the `Store`-vs-`Service` cross-module pattern: `modules/README.md`.

To create a new module: copy `modules/example/`, rename the package, wire deps in `dependencies.go`, and register its handlers in `server/server.go`.

## Middleware order (in `server/server.go`)

```
gin.CustomRecovery → otelgin.Middleware → RequestID → RequestLog → Metrics → handler
```

All in `middleware/` (except `otelgin.Middleware`, from `go.opentelemetry.io/contrib`). Request ID is stored in `context.Context` (shared HTTP/gRPC); retrieve via `middleware.GetRequestID(ctx)`. No custom RealIP middleware — Gin's `engine.SetTrustedProxies()` + `c.ClientIP()` handle it. No custom Recovery middleware beyond `gin.CustomRecovery` (routes panics through zap, not stdout).

**Never log request or response bodies, at any status** — after JSON decoding the body isn't what the client sent anyway, and 4xx bodies are exactly where secrets (failed login passwords, rejected payment details) are most likely to appear. Use the request ID to correlate instead. Full rationale: `docs/logging.md`.

## Errors and logging (see `docs/errors.md`, `docs/logging.md`, `docs/style.md`)

- **Handle errors once**: don't `log.Error(err); return err` in the same function. `RequestLog` (`middleware/request_log.go`) is the single logging boundary for HTTP requests — every layer below wraps and returns, nothing below it logs.
- **Wrap with `%w`** when a caller up the chain might need `errors.Is`/`errors.As` on the root cause; use `%v` to deliberately obfuscate before crossing a package boundary you don't control.
- To surface an error from a handler: call `c.Error(err)` before writing the response — `RequestLog` attaches it to the canonical request log line. Don't call the logger directly from a handler.
- A module's domain errors stay plain sentinels/types with no non-stdlib imports — HTTP knowledge belongs only in the transport layer, so a handler that needs a specific status checks `errors.Is(err, module.ErrFoo)` itself and picks the status inline (see `modules/example/handler.go`). `errors.Is` for recognizing driver/library sentinels belongs at the infra layer (store), translating them into the module's own domain sentinel.
- Log level tracks response status, not developer judgment: Info (2xx/3xx), Warn (4xx), Error (5xx) — derived automatically by `RequestLog`.
- **Exit once, from `main()` only**: `os.Exit`/`log.Fatal` belong in `cmd/*/main.go` or the top-level `Run*` function (`server/server.go`), never in a library/service/store function.
- Verify interface compliance at compile time: `var _ Interface = (*Type)(nil)` next to the type definition.
- Sentinel errors: exported `ErrFoo`, unexported `errFoo`, custom error types `FooError`.
- Import order enforced by `goimports` with `local-prefixes: brook` (`.golangci.yml`): stdlib, blank line, everything else (third-party and `brook/...` separated too).
- Struct literals use field names (`Config{Level: "info"}`), never positional.

`golangci-lint` already enforces `bodyclose`, `nilerr`, `nilnesserr`, `nilnil`, `govet` shadow/fieldalignment, `gofmt`/`goimports` — don't hand-check what the linter catches.

## Renaming the project

This skeleton uses `brook` as the Go module name. When cloning for a new project:

```bash
scripts/rename-module.sh <new-module-name>   # e.g. github.com/myorg/myproject
```

Updates `go.mod`, all import paths (e.g. `"brook/middleware"` → `"github.com/myorg/myproject/middleware"`), and `.golangci.yml`'s `local-prefixes`. Run `go build ./...` after to verify.

## Mocks

`.mockery.yaml` has a single `packages.brook` entry (module root) with `recursive: true` + `all: true`, so it walks every package under the module and generates a testify-style mock for every interface it finds — there's no need to list new packages/modules individually. Output goes to `mocks/{{.InterfaceDirRelative}}`, named `{{.InterfaceName | snakecase}}_mock.go`. Run `make mocks` after adding/changing an interface. Because the generated mock package imports the module package it mocks, unit tests that use it must live in `package <mod>_test` (external test package) to avoid an import cycle — see `modules/example/service_test.go`.
