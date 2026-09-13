# Brook — Agent Guide

Brook is a Go 1.27 modular monolith: one binary, with domain modules as Go
packages. The accepted architectural rationale is in
[`docs/adr/0001-modular-monolith-production-boundaries.md`](docs/adr/0001-modular-monolith-production-boundaries.md).

## Commands

```bash
make run                         # run the dev server; .env supplies APP_ENVIRONMENT=dev
make test                        # go test -short -race -count=1 ./...
make lint                        # golangci-lint run
make vendor                      # go mod tidy && go mod vendor
make swag                        # regenerate docs/ from cmd/example/main.go
make mocks                       # regenerate Mockery mocks
make lefthook-install            # install the pinned Lefthook binary locally
make lefthook                    # install Git hooks using lefthook.yml
make check                       # format-check, lint, vet, and race tests
make modernize                   # go fix ./...
make align                       # fieldalignment -fix ./...
make rename name=<new-name>     # scripts/rename-module.sh <new-name>
make ci                          # act workflow_dispatch
make migrate-up                 # apply SQLite migrations; requires SQLITE_DSN
make migrate-down               # roll back one SQLite migration; requires SQLITE_DSN
make migrate-create name=<name> # create a SQLite migration
```

Run a focused test with `go test -run TestName ./path/to/package`.

## Architecture

- The composition root is `server/`: it loads configuration, opens embedded
  stores, constructs modules, creates the router, and owns graceful shutdown.
- `router/` owns HTTP middleware registration and route mapping. It receives
  module handler functions from `server/`; it does not construct domain modules.
- `internal/<name>/` owns a bounded domain area, its application behavior, HTTP
  handlers, and the storage port it needs. Modules communicate in-process via
  an exported provider-owned `Service` interface only when a real consumer
  exists.
- `store/` contains infrastructure constructors only. It has no domain
  knowledge: SQLite is the relational store and Badger is the embedded key-value
  store.
- There is no global mutable application state. Dependencies are passed through
  constructors and the composition root.
- Embedded SQLite and Badger are single-node stores. They are appropriate for a
  small deployment and local operation, but they are not a horizontal
  availability or independently scalable module boundary.

## Configuration and operations

- `APP_ENVIRONMENT` is required and must be exactly `dev` or `prd`. It selects
  `config/config_dev.yaml` or `config/config_prd.yaml`.
- `config.Load` applies `SQLITE_DSN` and `BADGER_DIR` overrides, including an
  explicitly empty override, and validates the merged configuration before the
  server opens dependencies.
- Safe operational values live in the environment-specific YAML files.
  Deployment-specific datastore paths may come from environment variables.
  Secrets must come from the deployment secret mechanism, not committed YAML.
- `logger.level` and production sampling settings are applied when the zap
  logger is built. Production uses structured JSON logging.
- SQLite migrations are never run by the application at startup. The deployment
  must run `make migrate-up` (or an equivalent Goose migration job) before
  rolling out a compatible application version. The Docker image includes the
  checked-in SQL files for that job.
- `/health` is a liveness endpoint. `/ready` checks SQLite connectivity and the
  Badger lifecycle state and should be used for traffic readiness.

## Middleware and logging

Middleware order is:

```text
gin.CustomRecovery → RequestID → RequestBodyLimit → RequestLog → handler
```

- `RequestID` accepts a bounded safe `X-Request-ID` value or generates a
  128-bit random hexadecimal identifier. It stores the value in `context.Context`
  and echoes it in the response. The gRPC interceptor follows the same rule.
- `RequestLog` emits one structured completion log for normal HTTP requests.
  It logs method, route pattern, status, duration, and request ID. It never logs
  request or response bodies.
- Query logging is disabled in production. If enabled, only configured
  allowlisted keys are logged and sensitive-looking keys are redacted.
- Handlers attach errors with `c.Error(err)` before writing an error response.
  The request logger is the HTTP logging boundary; lower layers wrap and return
  errors but do not log them. Error messages must not contain secrets.
- Recovery logs panic details with method, route, and request ID. It remains in
  the application because a gateway cannot observe failures inside the process.
- Gateways may own TLS, WAF, edge authentication, rate limiting, routing, and
  edge access logs. They do not replace application authorization, validation,
  recovery, domain errors, or local correlation.

## Module pattern

Copy `internal/example/` as the reference shape:

- `dependencies.go`: exported `DependenciesConfig`, unexported wired
  `dependencies`, and `NewDependencies`.
- `types.go`: domain types.
- `interface.go`: the provider-owned exported `Service` contract and the
  package-private `store` port, each with a compile-time assertion.
- `handler.go`: transport and HTTP status mapping.
- `<action>.go`: service and storage implementation for one operation.
- `business_error.go`: domain sentinels with no HTTP dependencies.

The package-private `store` interface deliberately keeps persistence out of the
module's public API. Mockery generates its mock under `mocks/`; storage method
names are exported when the external generated mock must satisfy the private
interface. `newDependencies` is package-private, with an `export_test.go`
wrapper for external tests.

Do not impose a universal “one public interface per module” rule. Expose only
small provider-owned contracts that have real consumers; keep implementation
ports private where possible.

## Testing and boundaries

Every new module or middleware behavior should have unit tests; persistence
behavior should also have SQLite integration coverage where useful. Tests must
cover configuration validation, request correlation, redaction, error mapping,
and readiness behavior as those are operational contracts.

Use Mockery-generated mocks for reusable interface seams; keep them under
`mocks/` and do not hand-write duplicate fakes in each test file.

Use package boundaries and review rules to prevent modules from importing
another module's concrete dependencies. A future microservice extraction will
also require an external API contract, independent persistence and migrations,
its own process/deployment, and network timeout/retry/idempotency policies. The
in-process `Service` interface is not a wire protocol.

## Repository rules

- Use `apply_patch` for source and documentation edits.
- Preserve unrelated working-tree changes.
- Do not hand-edit generated Swagger files; run `make swag`.
- Do not reintroduce Prometheus, OpenTelemetry, Pyroscope, `/metrics`, or other
  observability services into this repository. Operational telemetry is owned by
  the deployment environment.
