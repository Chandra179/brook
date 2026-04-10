# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Vendor dependencies
make vendor           # runs: go mod tidy && go mod vendor

# Run the server
go run cmd/app/main.go

# Build
go build ./...

# Test
go test ./...
go test ./internal/middleware/...   # test a specific package

# Single test
go test -run TestFunctionName ./internal/middleware/...
```

The server requires `JWT_SECRET` env var at runtime.

## Architecture

This is a Go HTTP server skeleton using only the standard library (`net/http`) plus a shared logger from `github.com/Chandra179/gosdk`.

**Entry point**: `cmd/app/main.go` → `internal.Server()`

**`internal/server.go`** wires everything together:
- Creates a `Dependencies` struct (holds the logger, injected into middleware that need it)
- Builds a `globalChain` applied to all routes: Recovery → RequestID → Timeout → Logger → Auth → RateLimit
- Per-route chains add Authorization on top (e.g., `deps.Authorization(policy, "orders", "write")`)

**`internal/middleware/`** contains all middleware:
- `chain.go` — `Chain(handler, ...Middleware)` applies middlewares outermost-first
- `dependencies.go` — `Dependencies` struct; middleware needing shared state (logger) are methods on it
- `context.go` — helpers to store/retrieve user ID and request ID from `context.Context`
- `ratelimit.go` — per-key token bucket rate limiting (configurable via `RateLimitConfig`)
- `request_validation.go` — generic `Validate[T]` helper using struct tags
- `logger.go`, `recovery.go`, `request_id.go`, `timeout.go` — standard middleware

**Pattern**: stateless middleware (no shared deps) are plain `func(...) Middleware` constructors. Middleware needing the logger are methods on `*Dependencies`.
