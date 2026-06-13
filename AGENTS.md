# Brook — Agent Guide

Go modular monolith skeleton. Module `brook`, Go 1.26.3.

## Quick commands

```bash
go run cmd/example/main.go   # start HTTP server
go build ./...                # build check
make vendor                   # go mod tidy && go mod vendor
```

No test/lint/CI infrastructure exists.

## Architecture

- **Framework**: Gin (`github.com/gin-gonic/gin`). Handlers are `gin.HandlerFunc`.
- **Logger**: `go.uber.org/zap` used directly (no wrapper).
- **gRPC** dependency present (for `middleware.RequestIDUnaryInterceptor`) but no gRPC server is wired in the current entrypoint.
- **Vendor excluded from `.gitignore`** — `make vendor` runs `go mod vendor` but result is not committed.
- No global state. Dependencies injected via struct fields.

## Layout

| Path | Role |
|------|------|
| `cmd/example/main.go` | Entrypoint, calls `server.RunHttpServer()` |
| `modules/server/http_server.go` | Assembles Gin router, registers middleware and routes, starts `http.Server` |
| `modules/<name>/` | Flat domain module. `config.go` + `dependencies.go` + handler files |
| `middleware/` | Gin middleware + gRPC interceptor. See `middleware/README.md` |
| `config/` | Shared config loader (`config.Load("config/config.yaml")`) |

## Middleware order (in `http_server.go`)

```
gin.CustomRecovery → RequestID → RequestLog → Timeout → handler
```

All in `middleware/` package. Request ID stored in `context.Context` — shared between HTTP and gRPC. Retrieve via `middleware.GetRequestID(ctx)`.

## Module pattern (`modules/<name>/`)

Flat Go package. Defines own `Config` struct (independent of shared `config/`). Handlers use `gin.HandlerFunc`. Validation via `c.ShouldBindJSON(&req)` + `binding` struct tags inside handlers.

**Gotcha**: the `example` module has its own `modules/example/config.go` with a different struct layout than `config/config.go`. The server wires `example.Handle` directly without using `example.Dependencies`. If adding a new module, copy `modules/example/` but the server assembly pattern in `modules/server/` is the real wiring reference — not the module's own `dependencies.go`.

## Config

Shared `config/config.yaml` loaded by `config.Load("config/config.yaml")`. Each module's config section nested under module key in the YAML. Module also defines its own `Config` struct — zero coupling between module configs.

See `example` section in `config/config.yaml` + `modules/example/config.go` for the pattern.

## Creating a new module

Copy `modules/example/`, rename package, add route in `modules/server/http_server.go`, add config section in `config/config.yaml`.

## Renaming project

```bash
scripts/rename-module.sh <new-module-name>
```
Updates `go.mod` and all import paths. Run `go build ./...` to verify.
