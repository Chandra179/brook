# Brook

Go modular monolith skeleton. One binary, domain modules as Go packages. Split to microservices later — not before.

## Layout

```
cmd/example/main.go   # entrypoint — starts HTTP + gRPC
api/
modules/              # domain modules
  example/            #   example module
    config.go         #     module-specific config struct
    dependencies.go   #     wire deps, load own config
    http_handler.go   #     HTTP handlers
middleware/           # shared: recovery, request ID, timeout, validation
config/               # YAML loader + config.yaml
test/integration      # integration test decoupled from core business
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

Updates `go.mod` and all import paths (e.g. `"brook/middleware"` → `"github.com/myorg/myproject/middleware"`).
Run `go build ./...` after to verify.

## Commands

```bash
make vendor          # go mod tidy && go mod vendor
go run cmd/example/main.go
go build ./...
go test ./modules/...
```

## State

Mid-restructure. Single module (`example`). One entrypoint binary. Basic test coverage on config loading + DI wiring.

## Design choices

- Validation via `middleware.DecodeAndValidate[T](r)` inside handlers
- No `internal/` sub-packages inside modules
- Config struct per module, YAML section per module key
- No global state — deps injected via closure or struct field
