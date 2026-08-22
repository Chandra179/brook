# Modules

Each module is a Go package under `modules/<name>/`. Module owns its domain logic, transport, and DI.

## Required files

| File | Purpose |
|------|---------|
| `dependencies.go` | Unexported `dependencies` struct holding the module's wired deps (e.g. `logger`, `store`); exported `DependenciesConfig` struct callers fill in; `NewDependencies(*DependenciesConfig) *dependencies` constructor. For a module with persistence, `DependenciesConfig` takes the shared `*pgxpool.Pool` and `NewDependencies` constructs the module's Postgres-backed `store` itself (see `modules/example/dependencies.go`) — callers never build the store directly, and the `store` interface stays unexported since nothing outside the package needs to name it. Handlers are methods on `*dependencies`. |
| `types.go` | Domain types, structs, constants |

## Optional files

| File | Purpose |
|------|---------|
| `store.go` | Unexported `store` interface + its Postgres-backed implementation (only if the module needs persistence); constructed internally by `NewDependencies` from the injected `*pgxpool.Pool` |
| `service.go` | `Service` interface + its implementation on `*dependencies` (only if another module calls into this one) |
| `handler.go` | Module entrypoint — HTTP handlers |
| `business_error.go` | Domain sentinels (plain `errors.New(...)`, no non-stdlib imports) |
| `<action>.go` | One file per handler/operation (e.g. `create_order.go`, `get_order.go`) |

## Cross-module communication

Modules call each other in-process, through an interface — never by importing
and holding a sibling module's concrete `*dependencies` type directly.

- **`Service`** (in `service.go`) is what a module *provides* to other
  modules. It's the module's own public contract

```go
// modules/example/service.go — example provides this to callers
type Service interface {
    CreateExample(ctx context.Context, name string) (*Example, error)
}

var _ Service = (*dependencies)(nil)
```

A module that consumes a sibling depends on that sibling's `Service`
interface, wired in via its own `DependenciesConfig`.

```go
// modules/foo/dependencies.go
type DependenciesConfig struct {
    Logger  *zap.Logger
    Example example.Service
}
```

`server/server.go` constructs the concrete `*dependencies` for each module
and hands it to siblings as the interface type — see `exampleDeps` passed
into `foo.NewDependencies` as `example.Service` in `server/server.go`.

Only add a `Service` interface once a real second module needs to call in
— don't add one to every module speculatively.
