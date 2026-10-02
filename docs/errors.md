# Error Handling Standards

How we wrap, inspect, and surface errors across layers.

---

## Rules

* **Never swallow errors** — use error wrapping (%w)
* **Add context on the way up.** Each layer an error crosses, wrap with
  what was being attempted, without embedding submitted values or secrets.
* **Sanitize external errors.** Never expose a raw DB/system error to an
  API client or request log.

## Wrapping: `%w` vs `%v`

Use `fmt.Errorf("...: %w", err)` when a caller up the chain might need
`errors.Is`/`errors.As` on the root cause. `%v` drops error identity but does
not sanitize the error message. The HTTP layer uses a static response and an
explicitly approved `middleware.SafeErrorMessage` for request logs.

## Layered pattern

1. **Infra layer** recognizes a driver/library sentinel (e.g. `sql.ErrNoRows`)
   with `errors.Is` and translates it into a domain sentinel owned by that
   module (e.g. `ErrUserNotFound`, in that module's own `business_error.go`). A
   domain sentinel is a plain `errors.New(...)` value — it imports nothing
   beyond stdlib `errors`, so the module stays fully unaware of HTTP.
2. **Business layer** wraps it with context: `fmt.Errorf("fetch user: %w", err)`.
   It doesn't inspect or decide status codes — it has no notion of HTTP.
3. **Transport layer** is the only layer allowed to know about HTTP status
   codes, so it's the only layer allowed to inspect the error: the handler
   uses `errors.Is` against the module's own domain sentinels, attaches the
   original error with safe log metadata, and calls `c.JSON` with the status
   itself — see [`logging.md`](logging.md).

```go
// internal/example/business_error.go — domain sentinel, no non-stdlib imports
var ErrUserNotFound = errors.New("user not found")

// infra layer
func (r *Repository) FetchUser(id string) (*User, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("fetch user: %w", ErrUserNotFound)
	}
	...
}

// business layer
func (s *Service) GetUserProfile(id string) (*Profile, error) {
	user, err := s.repo.FetchUser(id)
	if err != nil {
		return nil, fmt.Errorf("load profile: %w", err)
	}
	return &Profile{Name: user.Name}, nil
}

// transport layer — the only place that maps a domain condition to a status
func (h *handler) HandleGetUser(c *gin.Context) {
	profile, err := h.service.GetUserProfile(c.Param("id"))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_ = c.Error(err).SetMeta(middleware.SafeErrorMessage("user not found"))
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		_ = c.Error(err).SetMeta(middleware.SafeErrorMessage("get user profile failed"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, profile)
}
```

A request-binding failure follows the same shape — no sentinel to check,
just `c.JSON(http.StatusBadRequest, ...)` directly — see
`internal/example/handler.go`.
