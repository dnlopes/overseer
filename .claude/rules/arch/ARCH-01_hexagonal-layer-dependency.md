# ARCH-01: Hexagonal Layer Dependency

## Rule
Dependencies flow inward only: secondary adapters → service → domain ← primary adapters; domain has NO imports of services or adapters.

## Why
Hexagonal architecture keeps domain logic independent of I/O concerns, enabling testability and replaceability of adapters without touching business logic.

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
// No service or adapter imports — domain is self-contained
```

❌ Bad:
```go
// internal/core/domain/session.go
import (
    "github.com/dnlopes/overseer/internal/adapters/secondary/storage" // FORBIDDEN
)
```
