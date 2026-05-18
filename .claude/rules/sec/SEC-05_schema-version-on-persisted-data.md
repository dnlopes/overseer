---
paths:
  - "internal/adapters/secondary/**/*.go"
---
# SEC-05: Schema Version on Persisted Data

## Rule
Every persisted file format carries a schema version number; loaders read the version first and dispatch to a versioned parser.

## Why
Schema versioning enables safe migrations when the data format changes without breaking existing user data.

## Example
✅ Good:
```go
// internal/adapters/secondary/storage/store.go
type fileSchema struct {
    SchemaVersion int              `json:"schemaVersion"`
    Sessions      []domain.Session `json:"sessions"`
}
// s.schemaVersion = 1
```

❌ Bad:
```go
data, _ := json.Marshal(sessions) // no schema version — can't migrate
```
