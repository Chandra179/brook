# ADR 0001: Modular-monolith production boundaries

- Status: Accepted
- Date: 2026-09-12

## Context

Brook is an early Go service with one binary, a small number of domain modules,
embedded SQLite and Badger stores, and no repository-owned observability stack.
The code should remain easy for a small team to change while avoiding unsafe
defaults and coupling that would make a later service extraction needlessly
expensive.

## Decisions

### 1. Keep a modular monolith

Each domain module is a Go package that owns its domain behavior, transport
handlers, dependency wiring, and storage port. The `server/` package is the
composition root and `router/` owns HTTP route registration.

Modules communicate in-process through a small provider-owned `Service`
interface only when a real sibling consumer exists. A module does not hold a
sibling's concrete dependency struct.

This gives the team package-level ownership and test seams without paying the
deployment, network, consistency, and operational cost of microservices before
those costs solve a real problem.

### 2. Keep storage ports private to modules

Each module defines the package-private `store` interface it needs. The shared
`store/` package exposes infrastructure constructors but no domain repository
contract. The module constructor receives the shared SQLite connection and
builds its concrete storage implementation internally.

The storage interface itself stays private. Mockery generates its mock under
`mocks/`; its method names are exported when an external generated mock must
satisfy the private interface. A test-only `export_test.go` wrapper exposes the
package-private constructor seam to external tests without adding it to the
production API.

### 3. Validate configuration before dependency startup

Environment-specific YAML contains safe operational values. Environment
variables override deployment-specific datastore locations. The merged config
is validated before SQLite or Badger opens. `APP_ENVIRONMENT` must be `dev` or
`prd`, and required datastore paths/DSNs cannot be blank, including when an
environment variable is explicitly set to an empty value.

This makes configuration failure deterministic and visible at startup while
avoiding a blanket rule that every non-secret value must be supplied through an
environment variable.

### 4. Use application correlation and centralized request logging

HTTP and gRPC requests accept a bounded safe `X-Request-ID` or receive a new
128-bit random hexadecimal ID. The ID is stored in context and returned in the
response. Request completion logging is centralized in middleware; handlers
attach errors with `c.Error(err)` and lower layers wrap and return without
logging duplicates.

Query logging is disabled in production and, when enabled, uses an allowlist
with sensitive-key redaction. Request bodies and response bodies are never
logged. Recovery logs panic details with request correlation because recovery
occurs outside the normal completion middleware.

The sampling window is explicit in logger configuration. This is a correlation
policy, not a tracing implementation. If distributed
tracing is later adopted, W3C `traceparent` should be evaluated as the
inter-service propagation contract rather than overloading `X-Request-ID`.

### 5. Keep migrations outside application startup

Goose migrations remain an explicit deployment step, never an application
startup side effect. The runtime image includes the checked-in migration SQL so
a release migration job can use the same artifact. Deployments must apply
backward-compatible migrations before rolling out code that requires them.

This avoids multiple replicas racing to migrate and keeps schema changes under
release control. The trade-off is that deployment automation must treat a
successful migration as a prerequisite for rollout.

### 6. Separate liveness from readiness

`/health` is a cheap process liveness endpoint. `/ready` checks SQLite and
Badger before admitting traffic. This prevents a running but unusable process
from being treated as ready.

### 7. Keep edge and application responsibilities distinct

An API gateway may own TLS termination, WAF, rate limiting, authentication
verification, routing, load balancing, and edge access logs. The application
retains authorization, validation, domain errors, recovery, local request
correlation, and behavior-specific logs because a gateway cannot observe those
inside-process concerns.

### 8. Treat embedded stores as an explicit scale boundary

SQLite and Badger are intentionally single-node. They reduce operational
overhead and are suitable while the workload and availability requirements fit
that model. The trigger to replace them is workload, write contention, data
size, availability, or independent deployment—not the number of engineers or
module packages alone.

## Consequences

Positive consequences:

- Small teams get clear package ownership without distributed-system overhead.
- Module business code depends on ports rather than concrete storage details.
- Startup failures, request correlation, error logging, and readiness are
  explicit operational contracts.
- A future extraction has a clearer seam and a known list of work.

Trade-offs:

- The monolith still shares a process and embedded storage; modules cannot be
  independently deployed or horizontally scaled.
- The composition root and router require manual wiring as modules grow.
- External migration automation is mandatory.
- In-process `Service` interfaces are not network contracts. Extraction still
  requires wire DTOs, protocol/versioning, independent persistence and
  migrations, timeouts/retries/idempotency, and service operations.

## Revisit triggers

Revisit this ADR when any of the following becomes true:

- SQLite write contention or availability requirements exceed a single node.
- A module requires independent deployment, ownership, or scaling.
- Cross-module calls require network boundaries.
- Request volume makes synchronous access logging exceed the logging budget.
- The deployment adopts a repository-owned tracing or metrics policy.
