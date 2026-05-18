---
paths:
  - "internal/adapters/secondary/**/*.go"
---
# SEC-04: Atomic Writes

## Rule
Persistence operations that mutate files use `internal/shared/paths.AtomicWrite` (write to `.tmp`, then `os.Rename`); NEVER `os.WriteFile` on a path that holds existing user data.

## Why
`os.WriteFile` on an existing file can corrupt data if the process is killed mid-write; atomic rename is crash-safe.

## Example
✅ Good:
```go
// internal/adapters/secondary/storage/store.go
return paths.AtomicWrite(s.path, data)
// writes to s.path+".tmp" then os.Rename — crash-safe
```

❌ Bad:
```go
return os.WriteFile(s.path, data, 0644) // WRONG: not crash-safe
```
