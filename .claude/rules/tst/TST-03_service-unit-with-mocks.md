---
paths:
  - "**/*_test.go"
---
# TST-03: Service Unit Tests with Mocks

## Rule
Service tests use mockery-generated mocks; one test file per service file; cover happy path, each error branch, and context cancellation.

## Why
Mocks isolate the service from real I/O; generated mocks stay in sync with port interfaces automatically.

## Example
✅ Good:
```go
// internal/core/service/session_test.go
func TestSessionService_Create_HappyPath(t *testing.T) {
    repo := mocks.NewMockSessionRepository(t)
    tmux := mocks.NewMockTmuxAdapter(t)
    git  := mocks.NewMockGitAdapter(t)
    svc  := service.NewSessionService(repo, tmux, git, slog.Default())
    // set up expectations, call svc.Create, assert
}
```

❌ Bad:
```go
func TestCreate(t *testing.T) {
    store, _ := storage.New(t.TempDir()+"/data.json", slog.Default()) // real I/O in unit test
}
```
