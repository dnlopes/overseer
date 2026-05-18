---
paths:
  - "**/*_test.go"
---
# TST-04: Secondary Integration Tests When Feasible

## Rule
Secondary adapters get integration tests against real I/O where reasonable (tempfs for storage, ephemeral tmux for tmux); use `t.TempDir()`; fall back to unit tests with fakes only when real I/O is impractical.

## Why
Integration tests catch real-world bugs that mocks miss (file permissions, JSON encoding, process lifecycle).

## Example
✅ Good:
```go
// internal/adapters/secondary/storage/store_test.go
func TestStore_SaveAndGet(t *testing.T) {
    path := filepath.Join(t.TempDir(), "data.json")
    store, _ := storage.New(path, slog.Default())
    sess, _ := domain.NewSession("test", "proj")
    _ = store.Save(ctx, sess)
    got, err := store.Get(ctx, sess.ID)
    // assert got == sess, err == nil
}
```

❌ Bad:
```go
func TestStore_Save(t *testing.T) {
    mockFS := &MockFileSystem{} // mocking os.WriteFile — tests nothing real
}
```
