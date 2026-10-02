# Logging Standards

Level and content rules for logs. Builds on [`errors.md`](errors.md) (wrapping
on the way up) — this covers what happens once an error reaches a logger. We
use [`go.uber.org/zap`](https://pkg.go.dev/go.uber.org/zap), configured in
`logger/logger.go`.

---

## Rules

* **One line per normal request.** All request-scoped completion logging happens
  in `RequestLog` (`middleware/request_log.go`) — don't log elsewhere. Recovery
  logs panics separately because Gin recovers before the completion middleware
  can resume.
* **Never log request/response bodies, at any status.** A decoded body
  isn't what the client sent anyway, and 4xx bodies are exactly where
  secrets (a failed login's password) show up. Use the request ID to
  correlate instead.
* **Never log secrets** — auth headers, tokens, passwords, API keys — at
  any level.
* **Only approved error text enters request logs.** A handler attaches the
  original error with `c.Error(err)` for error identity and sets static
  `middleware.SafeErrorMessage` metadata. Missing, empty, or plain-string
  metadata logs `request failed`. The logger never formats the original error.
* **Query logging is opt-in.** Production disables it. When enabled, only
  configured allowlisted keys are included, and keys containing sensitive
  fragments are logged as `[REDACTED]`.
* **Log route patterns, not arbitrary URL paths.** Unmatched requests are
  recorded as `<unmatched>` so path parameters cannot accidentally become log
  fields.

## Surfacing an error from a handler

Attach the original error with a reviewed, static log message before writing
the response. `RequestLog` logs that message; handlers do not log separately:

```go
if err != nil {
	_ = c.Error(fmt.Errorf("get example: %w", err)).SetMeta(
		middleware.SafeErrorMessage("get example failed"),
	)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	return
}
```

## Stack traces

zap attaches these automatically by level. Recovery also records a stack,
method, route, and request ID. Gin's separate recovery writer is disabled and
the panic value is omitted because either can contain request data. Never add
a stack in a handler:

* **Production**: `Error`+ only (5xx, panics).
* **Development**: `Warn`+ (4xx too).

The configured `logger.level` controls the zap threshold. Production uses
structured JSON output and the configured access-log sampling policy. Sampling
is an operational volume control, not a replacement for unsampled audit logs.
