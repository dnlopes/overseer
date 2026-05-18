# ARCH-07: Error Wrapping with fmt.Errorf

## Rule
Canonical error wrapping is `fmt.Errorf("context: %w", err)`; `errs.Wrap` is DEPRECATED — do not use it in new code.

## Why
The codebase has 17 `fmt.Errorf` usages vs 4 `errs.Wrap` usages (all in one file). `fmt.Errorf` is stdlib, idiomatic Go; `errs.Wrap` is a thin wrapper that adds no value and is reserved for backward compatibility only.

## Example
✅ Good:
```go
// internal/core/service/session.go
return CreateSessionResponse{}, fmt.Errorf("list sessions: %w", err)
```

❌ Bad:
```go
return CreateSessionResponse{}, errs.Wrap(err, "list sessions") // deprecated
```
