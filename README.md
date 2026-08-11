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
    interface.go         #    Store interface
    service.go            #   business logic
    store.go               #  Postgres-backed Store implementation
    http_handler.go        #  HTTP handlers
    errors.go              #  sentinel/custom errors
    constant.go            #  module constants
middleware/              # shared: recovery, request ID, request logging
config/                  # YAML loader + config_dev.yaml / config_prd.yaml
logger/                  # zap logger constructor
tracing/                 # OTel TracerProvider constructor (OTLP/gRPC exporter)
store/                   # shared pgx pool constructor + goose migrations (store/migrations/)
docs/                    # generated swagger output (do not hand-edit)
test/integration/        # build-tagged (integration) tests exercising real infra
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
make run              # go run ./cmd/example/
make test             # go test -short -race -count=1 ./...  (unit only, skips integration)
make test-integration # go test -tags=integration -race -count=1 -v ./...
make lint             # golangci-lint run
make vendor           # go mod tidy && go mod vendor
make swag             # swag init -g cmd/example/main.go -o docs
make mocks            # go tool mockery
make up / down        # docker compose up/down
make modernize        # go fix ./...
make align            # fieldalignment -fix ./...
make migrate-up       # goose -dir store/migrations postgres "$POSTGRES_DSN" up
make migrate-down     # goose -dir store/migrations postgres "$POSTGRES_DSN" down
make migrate-create name=<name>  # goose -dir store/migrations create <name> sql
```

To run a single test: `go test -run TestName ./modules/example/...`.

## Design choices

- Framework: Gin. Handlers are `gin.HandlerFunc` methods on a module's unexported `*dependencies` struct.
- Validation via `c.ShouldBindJSON(&req)` + `binding` struct tags inside handlers.
- No `internal/` sub-packages inside modules.
- Shared config is flat (`http`, `logger`, `middleware`, `profiling`, `postgres`) — not nested per-module.
- No global state — deps injected via constructor.
- Persistence: `pgx`/`pgxpool`, wrapped behind a per-module `Store` interface so a future DB swap is contained to one module. `store/` only owns the shared pool constructor and goose migrations — no domain knowledge. This repo only knows Pyroscope's address (continuous profiling), not how/where the collector runs — that's owned externally.
- Observability: OTel tracing (`tracing/`, OTLP/gRPC export, gated by `tracing.enabled`) and a Prometheus-exposition-format `/metrics` endpoint — both scraped/received by Grafana Alloy, same "address only, collector owned externally" rule as Pyroscope. No Prometheus server, no Alloy config, lives in this repo. `middleware.RequestID` reuses the request's OTel trace ID when no `X-Request-ID` header is supplied, so logs/traces/responses correlate on one ID.

See `CLAUDE.md` / `AGENTS.md` for full architectural and workflow details.
