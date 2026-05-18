# ARCH-08: Compile-Time Interface Conformance

## Rule
Use `var _ Iface = (*Impl)(nil)` at package level for every port implementation.

## Why
Catches interface signature drift at compile time rather than at runtime when the adapter is first instantiated.

## Example
✅ Good:
```go
// internal/adapters/secondary/storage/store.go
var _ domain.SessionRepository = (*Store)(nil)
```

❌ Bad:
```go
// No conformance check — drift discovered only when Store is used in tests or main
type Store struct { ... }
func (s *Store) Save(...) error { ... }
```
