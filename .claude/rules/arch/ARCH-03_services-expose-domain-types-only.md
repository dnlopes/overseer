# ARCH-03: Services Expose Domain Types Only

## Rule
Service method signatures use domain types or service-local Request/Response structs — never adapter types.

## Why
Prevents adapter concerns from leaking into the service layer, keeping the service API stable across adapter changes.

## Example
✅ Good:
```go
// internal/core/service/session.go
type CreateSessionRequest struct {
    Name        string
    ProjectName string
}
type CreateSessionResponse struct {
    Session domain.Session
}
func (s *SessionService) Create(ctx context.Context, req CreateSessionRequest) (CreateSessionResponse, error)
```

❌ Bad:
```go
// WRONG: adapter type leaking into service signature
func (s *SessionService) Create(ctx context.Context, body HTTPRequestBody) (JSONResponse, error)
```
