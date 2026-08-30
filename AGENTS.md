# Brook — Agent Guide

Go modular monolith skeleton. Module `brook`, Go `1.27.0`. One binary, domain modules as Go packages.

## Commands

```bash
make run              # kills :8080, starts ONLY postgres+arangodb via docker compose, then APP_ENVIRONMENT=dev go run ./cmd/example/
make test             # go test -short -race -count=1 ./...
make lint             # golangci-lint run  (v2 config in .golangci.yml)
make vendor           # go mod tidy && go mod vendor
make swag             # swag init -g cmd/example/main.go -o docs
make mocks            # go tool mockery
make up / down        # docker compose up/down (up starts the brook app container too)
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
- **Persistence**: two stores, both in `store/` (shared infra, no domain knowledge):
  - Postgres via `pgx`/`pgxpool` — `store/postgres.go` exports only `NewPool(ctx, config.PostgresConfig)`. Each module owns its own `Store` interface + Postgres implementation together in its own `store.go` (see `modules/example/store.go`). Migrations are goose SQL files in `store/migrations/`, applied explicitly via `make migrate-up` (never at server startup). `goose` is intentionally **not** a go.mod dependency — installed on demand via `go install`, same as `golangci-lint`/`swag`.
  - ArangoDB via `github.com/arangodb/go-driver/v2` (v2 API — `arangodb` + `connection` packages, HTTP/2, `NewClient(conn)`; not the v1 `http.NewConnection`/`driver.NewClient` API). `store/arango.go` exports `NewArangoClient(ctx, config.ArangoDBConfig)`, which pings the server and auto-creates the configured database if missing (no migrations for ArangoDB).
- **No observability stack**: metrics (Prometheus), tracing (OTel), and profiling (Pyroscope) have all been removed. Do not re-add `/metrics`, `otelgin`, `tracing/`, or Pyroscope.

## Layout

| Path | Role |
|------|------|
| `cmd/example/main.go` | Entrypoint → `server.RunHttpServer()` |
| `server/server.go` | Assembles Gin router, registers middleware/routes, graceful shutdown |
| `modules/<name>/` | Flat domain module package |
| `middleware/` | Gin middleware + gRPC interceptor. See `middleware/README.md` |
| `config/` | Shared YAML loader + `config_prd.yaml` / `config_dev.yaml` |
| `logger/` | zap logger constructor |
| `docs/` | Generated swagger output (do not hand-edit) |
| `store/` | Shared `postgres.go` (pgx pool) + `arango.go` (ArangoDB client) + goose migrations (`store/migrations/`) |

## Config & env

- `APP_ENVIRONMENT` selects the config file: `dev` → `config/config_dev.yaml`, anything else (incl. unset) → `config/config_prd.yaml`. Load with `config.Load(path)` (`config/config.go`).
- Shared config is **flat** (`http`, `logger`, `middleware`, `postgres`, `arangodb`) — not nested per-module. Modules receive only what they need via constructor args.
- Env overrides applied in `config.Load` (same pattern, only when non-empty): `POSTGRES_DSN` → `postgres.dsn`; `ARANGODB_DATABASE`/`ARANGODB_USERNAME`/`ARANGODB_PASSWORD` → their `arangodb` fields.
- ArangoDB root credentials come from the `arangodb` config section (dev: user `root`, password `brook`). In compose the ArangoDB container runs with `ARANGO_ROOT_PASSWORD`; it is mapped to host port `8530` (not `8529` — that port is reserved by an unrelated external ArangoDB on this machine), so dev config points at `http://localhost:8530`.

## Module pattern (`modules/<name>/`)

Flat package, no `internal/` sub-packages. Wires deps in `dependencies.go` via `NewDependencies(&XConfig{...})` returning an unexported `*dependencies`. Handlers are methods on `*dependencies` registered directly in `server/server.go` (e.g. `exampleDeps.HandleExample` — there is no `mod.Handle` helper). Validation via `c.ShouldBindJSON(&req)` + `binding` struct tags inside handlers.

Reference module: `modules/example/` (files: `dependencies.go`, `types.go`, `service.go`, `store.go`, `handler.go`, `business_error.go`, `constant.go`; no separate `config.go`). `store.go` declares the module's own `Store` interface and its Postgres-backed implementation together, constructed via `NewPostgresStore(pool)` and wired in from `server/server.go` — copy this shape for any module needing persistence. `service.go` declares the module's `Service` interface (what it *provides* to other modules, as opposed to `Store`, which is what it *requires*) alongside `*dependencies`' implementation of it. `business_error.go` holds the module's own domain sentinels (plain `errors.New(...)`, no non-stdlib imports); the handler checks `errors.Is` against them and picks the HTTP status itself.

Full required/optional file table: `modules/README.md`.

## Cross-module communication

Modules call each other in-process through an interface — never by importing and holding a sibling module's concrete `*dependencies` type directly. `modules/foo/` calling `modules/example/` via `example.Service` is the worked example; see `modules/README.md` for the full pattern (why `Service` is owned by the provider, not hand-rolled per-consumer, and only added once a real second module needs to call in).

## Mocks

`.mockery.yaml` has one `packages.brook` entry (module root) with `recursive: true` + `all: true` — it walks the whole module and mocks every interface it finds, so new packages don't need a config entry. Since the generated mock imports the module package it mocks, tests using it must live in `package <mod>_test` (external test package) to avoid an import cycle — see `modules/example/service_test.go`.

## Middleware order (in `server/server.go`)

```
gin.CustomRecovery → RequestID → RequestLog → handler
```

All in `middleware/`. Request ID stored in `context.Context` (shared HTTP/gRPC); retrieve via `middleware.GetRequestID(ctx)`.

## Creating a new module

Copy `modules/example/`, rename the package, wire deps in `dependencies.go`, and register its handlers in `server/server.go`. No config section needed (shared config is flat).

## Renaming project

```bash
scripts/rename-module.sh <new-module-name>
```

Updates `go.mod`, all import paths, `.golangci.yml` `local-prefixes`, the `cmd/example/` directory name, and the Makefile's `swag`/`run` entrypoint references. Verify with `go build ./...`.