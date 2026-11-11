# Modules

Each module is a Go package under `internal/<name>/`. A module owns its domain
logic, application behavior, transport, and dependency wiring.

## Required files

| File | Purpose |
|------|---------|
| `dependencies.go` | Exported `DependenciesConfig` struct callers fill in; unexported `dependencies` struct holding the module's wired deps (e.g. `logger`, `store`); `NewDependencies(*DependenciesConfig) *dependencies` constructor. For a module with persistence, `DependenciesConfig` takes the shared `*sql.DB` (SQLite) and `NewDependencies` builds the store internally (see `internal/example/dependencies.go`) — callers never construct the store directly. Handlers are methods on `*dependencies`. |
| `types.go` | Domain types, structs, constants |

## Optional files

| File | Purpose |
|------|---------|
| `interface.go` | The provider-owned `Service` interface (what the module provides to real sibling consumers), with its compile-time assertion |
| `handler.go` | Module entrypoint — HTTP handlers |
| `business_error.go` | Domain sentinels (plain `errors.New(...)`, no non-stdlib imports) |
| `constant.go` | Unexported package constants |
| `<action>.go` | One file per handler/operation (e.g. `create_example.go`); holds the `*dependencies` method and its persistence operation |

There is no separate `service.go`/`store.go`; operation logic and persistence
live in per-action files like `create_example.go`. `Service` is exported only
when a real sibling consumer needs it.

## Cross-module communication

Modules call each other in-process, through an interface — never by importing
and holding a sibling module's concrete `*dependencies` type directly.

- **`Service`** (in `interface.go`) is what a module *provides* to other
  modules. It's the module's own public contract:

```go
// internal/example/interface.go — example provides this to callers
type Service interface {
    CreateExample(ctx context.Context, name string) (*Example, error)
}

var _ Service = (*dependencies)(nil)
```

A module that consumes a sibling depends on that sibling's `Service`
interface, wired in via its own `DependenciesConfig`.

`server/server.go` constructs the concrete `*dependencies` for each module,
passes infrastructure into `NewDependencies`, and passes the returned value to
siblings as the provider's interface type (e.g. `exampleDeps` used as
`example.Service`). HTTP handlers are then handed to `router/` for route
registration.

Only add a `Service` interface for a real cross-module contract. Do not impose a
universal one-public-interface rule: expose small provider-owned contracts when
there are multiple meaningful consumers, and keep persistence concrete unless
a real substitution boundary needs an interface.

External sibling modules should mock the exported `Service` contract.
