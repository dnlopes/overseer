---
paths:
  - "internal/adapters/secondary/**/*.go"
---
# SEC-06: Corruption Recovery

## Rule
On parse failure of user data, rename the corrupt file to `<file>.corrupted.<unix-timestamp>.json` and surface a clear error; NEVER delete user data automatically.

## Why
Renaming preserves user data for recovery; deletion is irreversible and unacceptable.

## Example
✅ Good:
```go
// internal/adapters/secondary/storage/store.go
if err := json.Unmarshal(data, &schema); err != nil {
    corruptedPath := s.path + ".corrupted." + strconv.FormatInt(time.Now().Unix(), 10) + ".json"
    os.Rename(s.path, corruptedPath)
    s.logger.Warn("corrupted data file detected, renamed and starting fresh", ...)
    return nil
}
```

❌ Bad:
```go
if err := json.Unmarshal(data, &schema); err != nil {
    os.Remove(s.path) // WRONG: destroys user data
}
```
