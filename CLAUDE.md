# Brook — Current Project Guidance

Brook is a Go 1.27 modular monolith: one binary with domain modules as Go
packages. Keep this file aligned with `AGENTS.md`; architectural rationale is
recorded in [`docs/adr/0001-modular-monolith-production-boundaries.md`](docs/adr/0001-modular-monolith-production-boundaries.md).

## Source of truth

- Repository workflow and implementation constraints: `AGENTS.md`
- Module structure and cross-module contracts: `modules/README.md`
- Coding conventions: `docs/style.md`
- Error handling: `docs/errors.md`
- Logging and redaction: `docs/logging.md`
- Architectural decisions: `docs/adr/`

When these documents disagree with the code, inspect the code and update the
stale documentation as part of the change.

## Project map

```text
cmd/example/main.go     entrypoint
server/                  composition root and graceful shutdown
router/                  Gin middleware and route registration
modules/<name>/          domain module, handler, service, storage port
middleware/              recovery, request ID, body limit, request logging
config/                  YAML loader, env overrides, validation
logger/                  zap configuration
store/                   SQLite and Badger constructors plus migrations
docs/adr/                accepted architectural decisions
```

## Working rules

- Construct dependencies in `server/` and inject them through constructors.
- Keep module storage interfaces package-private and module domain errors free
  from HTTP knowledge.
- Use provider-owned `Service` interfaces only for real in-process consumers;
  do not couple modules to sibling concrete dependency structs.
- Register module handlers in `router/` through `router.NewDependencies`.
- Validate configuration before opening SQLite or Badger. `APP_ENVIRONMENT`
  must be `dev` or `prd`; required datastore values must be non-empty.
- Run migrations as a deployment step before rollout, never from application
  startup. The runtime image contains the migration SQL files.
- Keep request logging centralized. Handlers call `c.Error(err)`; lower layers
  wrap and return without logging the same error again.
- Never log request/response bodies, secrets, raw credentials, or unrestricted
  query strings. Production query logging is off by default.
- Keep production access-log sampling explicit and configurable; audit events
  must use a separate unsampled policy if they are introduced.
- Use Mockery-generated interface mocks under `mocks/`; do not hand-write
  duplicate fakes in individual test files.
- Use `go test -short -race -count=1 ./...`, `go vet ./...`, and `make lint` for
  proportionate verification.
- Do not add metrics/tracing/profiling stacks to this repository; those are
  deployment-owned concerns.

## Scale and extraction boundary

SQLite and Badger are embedded single-node stores. This architecture favors
simple operation, low infrastructure overhead, and fast modular development.
Move to networked/replicated storage when workload, availability, or deployment
independence requires it—not merely because the repository has more modules or
engineers.

The current module packages are prepared for in-process composition, not
independent microservice deployment. Extraction requires an external API
contract, isolated data and migrations, a service entrypoint, network policies,
and independent operational controls.
