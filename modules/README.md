# Modules

Each module is a Go package under `modules/<name>/`. Module owns its domain logic, transport, and DI.

## Required files

| File | Purpose |
|------|---------|
| `dependencies.go` | Wire dependencies, load config, construct services |
| `types.go` | Domain types, structs, constants |

## Optional files

| File | Purpose |
|------|---------|
| `http_handler.go` | Module entrypoint — HTTP server |
| `<action>.go` | One file per handler/operation (e.g. `create_order.go`, `get_order.go`) |
