---
paths:
  - "internal/core/domain/**/*.go"
---
# DOM-01: Restricted Imports

## Rule
Domain may import ONLY: Go standard library, `github.com/google/uuid`, and `internal/shared/errs` / `internal/shared/paths` for cross-cutting types; NEVER service, adapter, or third-party I/O libraries.

## Why
Domain independence is the core invariant of hexagonal architecture; I/O imports would couple domain to infrastructure.

## Example
✅ Good:
```go
// internal/core/domain/session.go
import (
    "context"
    "errors"
    "strings"
    "time"
    "github.com/google/uuid"
)
```

❌ Bad:
```go
import (
    "github.com/dnlopes/overseer/internal/adapters/secondary/storage" // FORBIDDEN
    "encoding/json"  // I/O library — FORBIDDEN in domain
)
```
