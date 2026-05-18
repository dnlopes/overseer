---
paths:
  - "internal/adapters/secondary/**/*.go"
---
# SEC-02: I/O Isolation

## Rule
All filesystem / network / process I/O is contained in secondary adapters; service and domain never call `os.*`, `net.*`, or `exec.*`.

## Why
I/O isolation makes service and domain layers testable without real infrastructure.

## Example
✅ Good:
```go
// internal/adapters/secondary/storage/store.go
func (s *Store) load() error {
    data, err := os.ReadFile(s.path) // I/O only in secondary adapter
}
// service/session.go has NO os.* calls
```

❌ Bad:
```go
// internal/core/service/session.go
func (s *SessionService) Create(ctx context.Context, req CreateSessionRequest) ... {
    os.WriteFile("/tmp/session.json", data, 0644) // WRONG: I/O in service
}
```
