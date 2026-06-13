# Middleware

Gin middleware used by the HTTP server. Registered in order via `r.Use(...)`:

```
g.CustomRecovery → RequestID → RequestLog → Timeout → handler
```

## Files

| File | Kind | Description |
|------|------|-------------|
| `dependencies.go` | infra | Holds `*zlogger.Logger` for stateful middleware |
| `request_id.go` | middleware | Reads/reuses `X-Request-ID` header, stores in context, echoes in response. Also exports `RequestIDUnaryInterceptor` for gRPC and `GetRequestID(ctx)` for handlers. |
| `request_log.go` | middleware | Logs method, path, status, duration, request ID, and query params. Skips configured paths. Request bodies are intentionally NOT logged (see comment in file). |
| `timeout.go` | middleware | Sets a `context.WithTimeout` deadline on the request. |

## Why no body logging?

After JSON decoding, the body seen in middleware is not what the client sent (whitespace stripped, keys reordered, unknown fields dropped). Logging it provides no debugging value and risks PII leakage. Use structured error responses returned to the client instead.

## Why no RealIP middleware?

Gin's `engine.SetTrustedProxies()` + `c.ClientIP()` handle `X-Forwarded-For` / `X-Real-IP` with CIDR-based trust filtering. No custom middleware needed.

## Why no Recovery middleware?

`gin.CustomRecovery` handles panics and writes a 500 response. The recovery callback uses `zlogger` so stack traces go to structured logs, not stdout.

## Why no validation middleware?

`c.ShouldBindJSON(&req)` + `binding` struct tags replace the old `DecodeAndValidate[T]` helper. Keep validation logic in handlers, not middleware.
