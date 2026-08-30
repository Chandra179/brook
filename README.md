# Brook

Go modular monolith skeleton. One binary, domain modules as Go packages. Split to microservices later — not before.

## Layout

```
cmd/example/main.go     # entrypoint — starts HTTP server
server/                 # assembles Gin router, middleware, graceful shutdown
modules/                # domain modules
  example/              #   reference module
    dependencies.go     #     wire deps, construct services/store
    types.go             #    domain types
    service.go            #   business logic
    store.go               #  Store interface + Postgres-backed implementation
    handler.go              # HTTP handlers
    business_error.go       # domain sentinels/custom errors
    constant.go             # module constants
middleware/              # shared: recovery, request ID, request logging, gRPC interceptor
config/                  # YAML loader + config_dev.yaml / config_prd.yaml
logger/                  # zap logger constructor
store/                   # shared postgres pool + ArangoDB client + goose migrations (store/migrations/)
docs/                    # generated swagger output (do not hand-edit)
```

## Renaming the project

This skeleton uses `brook` as the Go module name. When cloning for a new project,
rename it with:

```bash
scripts/rename-module.sh <new-module-name>
```

Example:

```bash
scripts/rename-module.sh github.com/myorg/myproject
```

Updates `go.mod`, all import paths (e.g. `"brook/middleware"` → `"github.com/myorg/myproject/middleware"`), and `.golangci.yml`'s `local-prefixes`.
Run `go build ./...` after to verify.

## Commands

```bash
make run              # kills :8080, starts postgres+arangodb via docker compose, then APP_ENVIRONMENT=dev go run ./cmd/example/
make test             # go test -short -race -count=1 ./...
make lint             # golangci-lint run  (v2 config in .golangci.yml)
make vendor           # go mod tidy && go mod vendor
make swag             # swag init -g cmd/example/main.go -o docs
make mocks            # go tool mockery
make up / down        # docker compose up/down (up starts the brook app container too)
make modernize        # go fix ./...
make align            # fieldalignment -fix ./...
make re               # scripts/rename-module.sh example
make ci               # act workflow_dispatch (runs GitHub Actions locally via act)
make migrate-up       # goose -dir store/migrations postgres "$POSTGRES_DSN" up
make migrate-down     # goose -dir store/migrations postgres "$POSTGRES_DSN" down
make migrate-create name=<name>  # goose -dir store/migrations create <name> sql
```

Real CI is `.github/workflows/ci.yml` (go mod verify → golangci-lint → test → build → docker build).

To run a single test: `go test -run TestName ./modules/example/...`.

## Design choices

- Framework: Gin. Handlers are `gin.HandlerFunc` methods on a module's unexported `*dependencies` struct.
- Validation via `c.ShouldBindJSON(&req)` + `binding` struct tags inside handlers.
- No `internal/` sub-packages inside modules.
- Shared config is flat (`http`, `logger`, `middleware`, `postgres`, `arangodb`) — not nested per-module.
- No global state — deps injected via constructor.
- Logger: `go.uber.org/zap` used directly (no wrapper). Built via `logger.NewLogger(appEnvironment)` — Development for `dev`, else Production. Not handled in `config/`.
- Persistence: two stores in `store/` (shared infra, no domain knowledge):
  - Postgres via `pgx`/`pgxpool` — `store/postgres.go` exports only `NewPool`. Each module owns its own `Store` interface + Postgres implementation together in its own `store.go`. Migrations are goose SQL files in `store/migrations/`, applied explicitly via `make migrate-up` (never at server startup). `goose` is intentionally **not** a go.mod dependency — installed on demand via `go install`, same as `golangci-lint`/`swag`.
  - ArangoDB via `github.com/arangodb/go-driver/v2` (v2 API — `arangodb` + `connection` packages, HTTP/2, `NewClient(conn)`). `store/arango.go` exports `NewArangoClient`, which pings the server and auto-creates the configured database if missing (no migrations for ArangoDB).
- No observability stack: metrics (Prometheus), tracing (OTel), and profiling (Pyroscope) have all been removed. Do not re-add `/metrics`, `otelgin`, `tracing/`, or Pyroscope.

See `CLAUDE.md` / `AGENTS.md` for full architectural and workflow details.
