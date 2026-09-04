# Brook — Agent Guide

Go modular monolith skeleton. Module `brook`, Go `1.27.0`. One binary, domain modules as Go packages.

## Commands

```bash
make run              # kills :8080, then APP_ENVIRONMENT=dev go run ./cmd/example/ (embedded DBs, no docker)
make test             # go test -short -race -count=1 ./...
make lint             # golangci-lint run  (v2 config in .golangci.yml)
make vendor           # go mod tidy && go mod vendor
make swag             # swag init -g cmd/example/main.go -o docs
make mocks            # go tool mockery
make modernize        # go fix ./...
make align            # fieldalignment -fix ./...
make rename name=<new-name>  # scripts/rename-module.sh <new-name>
make ci               # act workflow_dispatch  (runs GitHub Actions locally via act)
make migrate-up       # goose -dir store/migrations/sqlite sqlite3 "$SQLITE_DSN" up
make migrate-down     # goose -dir store/migrations/sqlite sqlite3 "$SQLITE_DSN" down
make migrate-create name=<name>  # goose -dir store/migrations/sqlite create <name> sql
```

Real CI is `.github/workflows/ci.yml` (go mod verify → golangci-lint → test → migrate → build → docker build).

## Architecture

- **Framework**: Gin. Handlers are `gin.HandlerFunc` methods on a module's unexported `*dependencies` struct. Router construction lives in `router/` (`router.NewDependencies` + `(*dependencies).New()`).
- **Logger**: `go.uber.org/zap` used directly (no wrapper). Built via `logger.NewLogger(appEnvironment)` in `logger/logger.go` — Development for `dev`, else Production. **Not** handled in `config/`.
- **gRPC** dependency present (for `middleware.RequestIDUnaryInterceptor`), but no gRPC server is wired in the entrypoint.
- **No global state**; deps injected via constructor.
- **Persistence**: two embedded stores, both in `store/` (shared infra, no domain knowledge), neither needs a server/container:
  - SQLite via `modernc.org/sqlite` (pure Go, no CGO) — `store/sqlite.go` exports only `NewSQLite(ctx, config.SQLiteConfig)`. Each module owns its own `store` interface + SQLite-backed implementation (see `modules/example/` — interface in `interface.go`, impl in `create_example.go`). Migrations are goose SQL files in `store/migrations/sqlite/`, applied explicitly via `make migrate-up` (never at server startup). `goose` is intentionally **not** a go.mod dependency — installed on demand via `go install`, same as `golangci-lint`/`swag`.
  - Badger via `github.com/dgraph-io/badger/v4` (embedded key-value store) — `store/badger.go` exports `NewBadger(dir)`, no migrations.
- **No observability stack**: metrics (Prometheus), tracing (OTel), and profiling (Pyroscope) have all been removed. Do not re-add `/metrics`, `otelgin`, `tracing/`, or Pyroscope.

## Layout

| Path | Role |
|------|------|
| `cmd/example/main.go` | Entrypoint → `server.RunHttpServer()` |
| `server/server.go` | Assembles config, datastores, modules; graceful shutdown |
| `router/` | Builds the Gin engine (middleware chain + routes) via `NewDependencies`/`New` |
| `modules/<name>/` | Flat domain module package |
| `middleware/` | Gin middleware + gRPC interceptor. See `middleware/README.md` |
| `config/` | Shared YAML loader + `config_prd.yaml` / `config_dev.yaml` |
| `logger/` | zap logger constructor |
| `docs/` | Generated swagger output (do not hand-edit) |
| `store/` | Shared `sqlite.go` + `badger.go` (embedded) + goose migrations (`store/migrations/sqlite/`) |

## Config & env

- `APP_ENVIRONMENT` selects the config file: `dev` → `config/config_dev.yaml`, anything else (incl. unset) → `config/config_prd.yaml`. Load with `config.Load(path)` (`config/config.go`).
- Shared config is **flat** (`http`, `logger`, `middleware`, `sqlite`, `badger`) — not nested per-module. Modules receive only what they need via constructor args.
- Env overrides applied in `config.Load` (same pattern, only when non-empty): `SQLITE_DSN` → `sqlite.dsn`; `BADGER_DIR` → `badger.dir`.

## Module pattern (`modules/<name>/`)

Flat package, no `internal/` sub-packages. Wires deps in `dependencies.go` via `NewDependencies(&XConfig{...})` returning an unexported `*dependencies`. Handlers are methods on `*dependencies` registered in `router/router.go` via `router.NewDependencies` (e.g. `exampleDeps.HandleExample` — there is no `mod.Handle` helper). Validation via `c.ShouldBindJSON(&req)` + `binding` struct tags inside handlers.

Reference module: `modules/example/` (files: `dependencies.go`, `types.go`, `interface.go`, `create_example.go`, `handler.go`, `business_error.go`, `constant.go`; no separate `config.go`). `interface.go` declares the module's `Service` interface (what it *provides* to other modules) and its unexported `store` interface (what it *requires*). The SQLite-backed `store` impl and the `Service` impl live together in per-action files like `create_example.go`, constructed inside `NewDependencies` from the injected `*sql.DB` — copy this shape for any module needing persistence. `business_error.go` holds the module's own domain sentinels (plain `errors.New(...)`, no non-stdlib imports); the handler checks `errors.Is` against them and picks the HTTP status itself.

Full required/optional file table: `modules/README.md`.

## Cross-module communication

Modules call each other in-process through an interface — never by importing and holding a sibling module's concrete `*dependencies` type directly. `modules/foo/` calling `modules/example/` via `example.Service` is the worked example; see `modules/README.md` for the full pattern (why `Service` is owned by the provider, not hand-rolled per-consumer, and only added once a real second module needs to call in). (Note: `modules/foo/` was removed in the SQLite migration; the pattern still stands.)

## Mocks

`.mockery.yaml` has one `packages.brook` entry (module root) with `recursive: true` + `all: true` — it walks the whole module and mocks every interface it finds, so new packages don't need a config entry. Output goes to `mocks/{{.InterfaceDirRelative}}` (e.g. `mocks/modules/example/store_mock.go`). Because the generated mock imports the module package it mocks, tests using it must live in `package <mod>_test` (external test package) to avoid an import cycle.

## Middleware order (in `router/router.go`)

```
gin.CustomRecovery → RequestID → RequestLog → handler
```

All in `middleware/`. Request ID stored in `context.Context` (shared HTTP/gRPC); retrieve via `middleware.GetRequestID(ctx)`.

## Creating a new module

Copy `modules/example/`, rename the package, wire deps in `dependencies.go`, register its handlers in `router/router.go` (pass them to `router.NewDependencies`), and construct the module in `server/server.go`. No config section needed (shared config is flat).

## Renaming project

```bash
scripts/rename-module.sh <new-module-name>
```

Updates `go.mod`, all import paths, `.golangci.yml` `local-prefixes`, the `cmd/example/` directory name, and the Makefile's `swag`/`run` entrypoint references. Verify with `go build ./...`.