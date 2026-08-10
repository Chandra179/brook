# Brook — Agent Guide

Go modular monolith skeleton. Module `brook`, Go `1.26.5`. One binary, domain modules as Go packages.

## Commands

```bash
make run              # go run ./cmd/example/
make test             # go test -short -race -count=1 ./...  (unit only, skips integration)
make test-integration # go test -tags=integration -race -count=1 -v ./...
make lint             # golangci-lint run  (v2 config in .golangci.yml)
make vendor           # go mod tidy && go mod vendor
make swag             # swag init -g cmd/example/main.go -o docs
make mocks            # go tool mockery
make up / down        # docker compose up/down
make modernize        # go fix ./...
make align            # fieldalignment -fix ./...
make re               # scripts/rename-module.sh example
make profiler         # docker run -p 4040:4040 grafana/pyroscope (continuous profiling server)
make ci               # act workflow_dispatch  (runs GitHub Actions locally via act)
```

Integration tests are gated by `-tags=integration`; plain `make test` skips them. Real CI is `.github/workflows/ci.yml` (go mod verify → golangci-lint → test → test-integration → build → docker build).

## Architecture

- **Framework**: Gin. Handlers are `gin.HandlerFunc` methods on a module's unexported `*dependencies` struct.
- **Logger**: `go.uber.org/zap` used directly (no wrapper). Built via `logger.NewLogger(appEnvironment)` in `logger/logger.go` — Development for `dev`, else Production. **Not** handled in `config/`.
- **gRPC** dependency present (for `middleware.RequestIDUnaryInterceptor`), but no gRPC server is wired in the entrypoint.
- **No global state**; deps injected via constructor.

## Layout

| Path | Role |
|------|------|
| `cmd/example/main.go` | Entrypoint → `server.RunHttpServer()` |
| `server/http_server.go` | Assembles Gin router, registers middleware/routes, graceful shutdown |
| `modules/<name>/` | Flat domain module package |
| `middleware/` | Gin middleware + gRPC interceptor. See `middleware/README.md` |
| `config/` | Shared YAML loader + `config_prd.yaml` / `config_dev.yaml` |
| `logger/` | zap logger constructor |
| `docs/` | Generated swagger output (do not hand-edit) |
| `test/integration` | Currently empty; future integration tests |

## Config & env

- `APP_ENVIRONMENT` selects the config file: `dev` → `config/config_dev.yaml`, anything else (incl. unset) → `config/config_prd.yaml`. Load with `config.Load(path)` (`config/config.go`).
- Shared config is **flat** (`http`, `logger`, `middleware`, `profiling`) — not nested per-module. Modules receive only what they need via constructor args.
- `profiling` (Pyroscope push SDK) is gated by `enabled`; run `make profiler` to start the server it pushes to.

## Module pattern (`modules/<name>/`)

Flat package. Wires deps in `dependencies.go` via `NewDependencies(&XConfig{...})` returning an unexported `*dependencies`. Handlers are methods on `*dependencies` registered directly in `server/http_server.go` (e.g. `exampleDeps.HandleExample` — there is no `mod.Handle` helper). Validation via `c.ShouldBindJSON(&req)` + `binding` struct tags inside handlers.

Reference module: `modules/example/` (files: `dependencies.go`, `types.go`, `interface.go`, `service.go`, `http_handler.go`, `errors.go`, `constant.go`; no separate `config.go`).

## Middleware order (in `server/http_server.go`)

```
gin.CustomRecovery → RequestID → RequestLog → handler
```

All in `middleware/`. Request ID stored in `context.Context` (shared HTTP/gRPC); retrieve via `middleware.GetRequestID(ctx)`.

## Creating a new module

Copy `modules/example/`, rename the package, wire deps in `dependencies.go`, and register its handlers in `server/http_server.go`. No config section needed (shared config is flat).

## Renaming project

```bash
scripts/rename-module.sh <new-module-name>
```

Updates `go.mod`, all import paths, and `.golangci.yml` `local-prefixes`. Verify with `go build ./...`.