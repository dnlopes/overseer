---
paths:
  - "**/*_test.go"
---
# TST-07: Test Naming

## Rule
Test functions: `TestSubject_Scenario_ExpectedOutcome`; table-driven tests acceptable; each row's name describes the scenario.

## Why
Descriptive names make failing tests self-explanatory without reading the test body.

## Example
✅ Good:
```go
func TestSessionService_Create_DuplicateName_ReturnsAlreadyExistsError(t *testing.T) { ... }

tests := []struct{ name string; input string; wantErr error }{
    {name: "empty name", input: "", wantErr: domain.ErrSessionEmptyName},
    {name: "name too long", input: strings.Repeat("x", 101), wantErr: domain.ErrSessionNameTooLong},
}
```

❌ Bad:
```go
func TestCreate(t *testing.T) { ... }
func TestCreate2(t *testing.T) { ... }
func TestCreate_error(t *testing.T) { ... }
```
