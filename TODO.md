# Production readiness checklist

The modular-monolith architecture baseline is documented in
[`docs/adr/0001-modular-monolith-production-boundaries.md`](docs/adr/0001-modular-monolith-production-boundaries.md).
The items below are release and operations work beyond the code architecture
review.

## P0 — required before production

- [ ] Define and implement application authentication and authorization. A
  gateway may verify identity, but each module must enforce its own permissions.
- [ ] Define the deployment topology for SQLite and Badger: one application
  replica, persistent storage, file ownership, capacity limits, and failure
  recovery. Do not assume horizontal scaling with embedded stores.
- [ ] Add and validate SQLite production settings for the expected workload:
  WAL/busy-timeout policy, connection-pool limits, write-contention handling,
  and load-test evidence.
- [ ] Establish automated backups, restore drills, retention, and disaster
  recovery objectives for both embedded stores.
- [ ] Make database migrations a real deployment job using a pinned Goose
  version, compatibility checks, rollback policy, and release ordering.
- [ ] Provide production rate limiting, TLS termination, WAF/edge controls,
  CORS policy where applicable, and request-size limits at the deployment edge.
- [ ] Disable or protect the production Swagger endpoint.
- [ ] Connect external log aggregation, alerting, dashboards, health probes,
  and SLOs. The repository intentionally does not own a metrics/tracing stack.

## P1 — required before a high-traffic rollout

- [ ] Run representative load, soak, and failure tests, including SQLite lock
  contention, Badger restart, dependency failure, shutdown, and readiness loss.
- [ ] Add handler, router, migration, and persistence integration coverage;
  current unit coverage is strongest in configuration and middleware.
- [ ] Run dependency and container vulnerability scanning, generate an SBOM,
  sign release images, and run the container as a non-root user with a
  read-only filesystem where supported.
- [ ] Pin all CI and developer tools instead of using `@latest`, and verify
  reproducible builds and migration artifacts.
- [ ] Define request-log volume, retention, redaction review, and audit-event
  policies; adjust access-log sampling from measured production traffic.

## P2 — operational maturity

- [ ] Document deploy, rollback, migration, backup/restore, incident, and
  embedded-store replacement runbooks.
- [ ] Revisit the embedded-store decision when write contention, data size,
  availability, or independent module deployment exceeds the single-node
  operating model.
