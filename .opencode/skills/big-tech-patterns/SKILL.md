---
name: big-tech-patterns
description: > 
  Reference for how Google, Uber, Stripe, Amazon, and Spotify approach. Use when asking 'how does [big tech] handle [topic]?' or exploring industry alternatives.
---

# Big Tech Patterns

Reference for architectural patterns used by major tech companies across common categories.
Use when you want to know how a specific company approaches a topic, or when exploring alternatives before making a design decision.

## Module Layout

- **Google**: Deep directory trees per team, Bazel `go_library`/`go_binary` rules. Monorepo with `google.golang.org` root.
- **Uber**: Monorepo with `go.uber.org/...` paths. Service per team, flat packages. Created `zap`, `fx`, `dig` from internal patterns.
- **Stripe**: One package per resource (payment, customer, etc.). No deep nesting. Strong API boundary between packages.
- **Amazon**: Two-pizza team owns one service. Each service is its own repo. Service boundaries are strongly API-enforced.
- **Spotify**: Squad owns a module/namespace. Module boundaries align with squad ownership. Backend-for-frontend per client type.

| Approach | Pros | Cons |
|----------|------|------|
| Flat module | Simple, easy to navigate, no circular deps | Can't hide internal details |
| Deep tree (Google) | Clear ownership, hides internals | Complex build system, slow navigation |
| Per-resource (Stripe) | Natural API surface, testable | Duplication across resources |

## Dependency Injection

- **Google**: Constructor injection via `google-wire` (compile-time code generation). No runtime reflection. Dependencies are interfaces.
- **Uber**: Created `fx`/`dig` (runtime reflection). Many teams now moving back toward explicit constructors due to debugging difficulty.
- **Stripe**: Explicit constructors. Dependencies are plain Go interfaces. Simple and explicit is the rule.
- **Amazon**: Explicit injection via config objects passed at construction. No DI framework in Go services.
- **Spotify**: Java: annotation-based (Spring). Go: explicit wiring with `wire` or hand-rolled.

| Approach | Pros | Cons |
|----------|------|------|
| Explicit constructors | Simple to debug, clear ownership | Verbose wiring for deep graphs |
| Code-gen (Google wire) | Type-safe, fast at runtime | Build step, generated code in repo |
| Runtime reflection (Uber fx) | Auto-wiring, less boilerplate | Hard to debug, magic errors |

## Middleware

- **Google**: gRPC interceptor chains. Unary and stream interceptors compose via `grpc-go`. HTTP middleware via `google-api-go-client`.
- **Uber**: Internal `go-http` framework with middleware chains. Observability (Jaeger tracing), rate limiting, circuit breaking all as middleware.
- **Stripe**: Minimal stack. Nginx → custom Go backend. Idempotency key middleware is critical. Most logic in handlers.
- **Amazon**: API Gateway handles auth, rate limiting at infrastructure layer. Lambda middleware layers for application concerns.
- **Spotify**: Apollo Server plugins as middleware. Go services use Gin-like chains.

| Approach | Pros | Cons |
|----------|------|------|
| Thin middleware | Predictable, low latency | More handler boilerplate |
| Rich middleware (Google/Uber) | Cross-cutting concerns centralized | Debugging across layers harder |
| Infra middleware (Amazon) | Offloads auth/throttling | Ties to platform, harder to test locally |

## Config Management

- **Google**: Structured config protos. Central config service (Chubby/Borg). Configs are versioned and deployed with the binary.
- **Uber**: Environment variables + YAML/JSON files. Dynamic config updates via internal system. Feature flags.
- **Stripe**: Environment-based. Secrets in Vault. Minimal file config — most parameters are API-driven.
- **Amazon**: SSM Parameter Store / AppConfig externalized from binary. DynamoDB for feature flags.
- **Spotify**: LaunchDarkly for feature toggles. Per-environment YAML files. Dynamic via config service.

| Approach | Pros | Cons |
|----------|------|------|
| File-based | Simple, no infra dependency | Requires restart for changes |
| Central service (Google) | Dynamic, auditable | Dependency on config service availability |
| Externalized (Amazon) | No binary rebuild needed | Latency on config reads, complexity |

## Error Handling

- **Google**: Structured gRPC status codes with typed error details. Unstructured errors are bugs.
- **Uber**: `go.uber.org/multierr` for combining errors. Transient vs permanent classification for retry logic.
- **Stripe**: Well-defined error response envelope (`type`, `code`, `message`, `param`). HTTP status codes map 1:1 to error types.
- **Amazon**: Each service defines its error codes. `aws-sdk-go` returns typed errors with request IDs for debugging.
- **Spotify**: GraphQL error paths with structured extensions. Domain-specific error codes in REST responses.

| Approach | Pros | Cons |
|----------|------|------|
| Ad-hoc JSON errors | Simple, fast to write | No contract for clients, untestable error paths |
| Typed errors (Google/Stripe) | Client can handle by type, testable | Boilerplate for error types |
| Classified errors (Uber) | Retry logic built-in | More infrastructure needed |

## API Design

- **Google**: Proto-based API definitions. gRPC primary transport, HTTP/JSON via `google.api.http` annotations.
- **Uber**: REST + gRPC hybrid. Thrift historically, migrating to gRPC. API version in URL path.
- **Stripe**: Versioned REST API (version in `Stripe-Version` header). Idempotency keys. Resource-oriented URLs. Consistent response envelope.
- **Amazon**: Query-protocol (XML/JSON). SigV4 request signing. Resource-based IAM policies.
- **Spotify**: Web API with OAuth. GraphQL for internal services. BFF per client type.

| Approach | Pros | Cons |
|----------|------|------|
| REST + Swagger | Simple, well-understood, easy to debug | Manual annotation, no type-safe clients |
| Protobuf + gRPC (Google) | Type-safe, streaming, polyglot codegen | Build step, harder to debug |
| Versioned REST (Stripe) | Backward compatible, explicit | Header-based versioning complex for caching |

## Testing

- **Google**: Table-driven tests (idiomatic Go). Integration tests with real services in prod-like environments.
- **Uber**: `go.uber.org/goleak` for goroutine leaks. Heavy fuzzing. Docker-compose for integration tests.
- **Stripe**: Regression test suite with request recording/replay. Idempotency testing. Chaos testing.
- **Amazon**: Per-service test suites. Canaries in production. Fault injection testing.
- **Spotify**: A/B testing. Squad-owned test suites. Contract testing between services.

| Approach | Pros | Cons |
|----------|------|------|
| Table-driven + integration | Covers unit + DB state | Integration tests slower, need infra |
| Canaries (Amazon) | Real traffic validation | Complex setup, risk of user-facing failures |
| Contract testing (Spotify) | Catches inter-service breaks | Maintenance cost for contracts |
