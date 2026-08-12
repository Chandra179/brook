# Brook — Agent Guide

Go modular monolith skeleton. Module `brook`, Go `1.26.5`. One binary, domain modules as Go packages.

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

Real CI is `.github/workflows/ci.yml` (go mod verify → golangci-lint → test → build → docker build).

## Architecture

- **Framework**: Gin. Handlers are `gin.HandlerFunc` methods on a module's unexported `*dependencies` struct.
- **Logger**: `go.uber.org/zap` used directly (no wrapper). Built via `logger.NewLogger(appEnvironment)` in `logger/logger.go` — Development for `dev`, else Production. **Not** handled in `config/`.
- **gRPC** dependency present (for `middleware.RequestIDUnaryInterceptor`), but no gRPC server is wired in the entrypoint.
- **No global state**; deps injected via constructor.
- **Persistence**: `pgx`/`pgxpool`. `store/postgres.go` exports only `NewPool(ctx, config.PostgresConfig)` — shared infra, no domain knowledge. Each module owns its own `Store` interface + Postgres implementation (see `modules/example/interface.go` + `store.go`). Migrations are goose SQL files in `store/migrations/`, applied explicitly via `make migrate-up` (never at server startup). `goose` is intentionally **not** a go.mod dependency — installed on demand via `go install`, same as `golangci-lint`/`swag`.
- **Tracing**: OpenTelemetry SDK (`tracing/tracing.go`'s `NewProvider`), gated by `tracing.enabled`, exporting via OTLP/gRPC. Same address-only rule as Pyroscope — only `tracing.otlp_endpoint` is known here, the collector (Grafana Alloy → Tempo) is owned externally. `otelgin.Middleware` always runs (no-op when disabled); `middleware.RequestID` reuses its span's trace ID as the request ID when no `X-Request-ID` header is present.
- **Metrics**: `github.com/prometheus/client_golang` at `/metrics` (exposition format, scraped by Alloy — no Prometheus server here). Metric vectors live on a `*prometheus.Registry` built once in `server/http_server.go` and injected into `middleware.NewDependencies` — no global registry.

## Layout

| Path | Role |
|------|------|
| `cmd/example/main.go` | Entrypoint → `server.RunHttpServer()` |
| `server/http_server.go` | Assembles Gin router, registers middleware/routes, graceful shutdown |
| `modules/<name>/` | Flat domain module package |
| `middleware/` | Gin middleware + gRPC interceptor. See `middleware/README.md` |
| `config/` | Shared YAML loader + `config_prd.yaml` / `config_dev.yaml` |
| `logger/` | zap logger constructor |
| `tracing/` | OTel `TracerProvider` constructor (OTLP/gRPC exporter) |
| `docs/` | Generated swagger output (do not hand-edit) |
| `store/` | Shared pgx pool constructor + goose migrations (`store/migrations/`) |

## Config & env

- `APP_ENVIRONMENT` selects the config file: `dev` → `config/config_dev.yaml`, anything else (incl. unset) → `config/config_prd.yaml`. Load with `config.Load(path)` (`config/config.go`).
- Shared config is **flat** (`http`, `logger`, `middleware`, `profiling`) — not nested per-module. Modules receive only what they need via constructor args.
- `profiling` (Pyroscope push SDK) is gated by `enabled`. This repo only knows Pyroscope's address, not how/where it runs — the collector is owned and operated elsewhere. Reached via `http://host.docker.internal:4040` from the root `docker-compose.yml`'s `brook` service (needs `extra_hosts: host-gateway`, since the collector isn't in this compose file's network), or `http://localhost:4040` for plain `make run`. Don't add a Pyroscope/Grafana service to this repo.
- `tracing` (OTel SDK) is gated by `enabled`, same address-only rule: only `otlp_endpoint` is known, Alloy is owned externally. `OTEL_EXPORTER_OTLP_ENDPOINT` env var overrides `tracing.otlp_endpoint` if set.
- `POSTGRES_DSN` env var overrides `postgres.dsn` from the YAML if set (same override pattern as `PYROSCOPE_SERVER_ADDRESS`).

## Module pattern (`modules/<name>/`)

Flat package. Wires deps in `dependencies.go` via `NewDependencies(&XConfig{...})` returning an unexported `*dependencies`. Handlers are methods on `*dependencies` registered directly in `server/http_server.go` (e.g. `exampleDeps.HandleExample` — there is no `mod.Handle` helper). Validation via `c.ShouldBindJSON(&req)` + `binding` struct tags inside handlers.

Reference module: `modules/example/` (files: `dependencies.go`, `types.go`, `interface.go`, `service.go`, `store.go`, `http_handler.go`, `errors.go`, `constant.go`; no separate `config.go`). `store.go` implements the module's own `Store` interface against a `*pgxpool.Pool`, constructed via `NewPostgresStore(pool)` and wired in from `server/http_server.go` — copy this shape for any module needing persistence.

## Mocks

`.mockery.yaml` has one `packages.brook` entry (module root) with `recursive: true` + `all: true` — it walks the whole module and mocks every interface it finds, so new packages don't need a config entry. Since the generated mock imports the module package it mocks, tests using it must live in `package <mod>_test` (external test package) to avoid an import cycle — see `modules/example/service_test.go`.

## Middleware order (in `server/http_server.go`)

```
gin.CustomRecovery → otelgin.Middleware → RequestID → RequestLog → Metrics → handler
```

All in `middleware/` (except `otelgin.Middleware`, from `go.opentelemetry.io/contrib`). Request ID stored in `context.Context` (shared HTTP/gRPC); retrieve via `middleware.GetRequestID(ctx)`.

## Creating a new module

Copy `modules/example/`, rename the package, wire deps in `dependencies.go`, and register its handlers in `server/http_server.go`. No config section needed (shared config is flat).

## Renaming project

```bash
scripts/rename-module.sh <new-module-name>
```

Updates `go.mod`, all import paths, and `.golangci.yml` `local-prefixes`. Verify with `go build ./...`.