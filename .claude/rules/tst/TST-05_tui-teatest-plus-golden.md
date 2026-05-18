---
paths:
  - "**/*_test.go"
---
# TST-05: TUI Tests with teatest + Golden Files

## Rule
TUI feature packages tested with `charmbracelet/x/exp/teatest/v2` for behavior + `charmbracelet/x/exp/golden` for visual regressions; golden files live in `testdata/golden/`.

## Why
teatest drives the model through real message sequences; golden files catch unintended visual changes.

## Example
✅ Good:
```go
// internal/adapters/primary/tui/session/list_test.go
func TestSessionList_RendersEmpty(t *testing.T) {
    m := session.New(styles.New(), mockSvc)
    tm := teatest.NewTestModel(t, m)
    tm.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
    golden.RequireEqual(t, []byte(tm.FinalView()))
}
```

❌ Bad:
```go
func TestSessionList(t *testing.T) {
    m := session.New(styles.New(), mockSvc)
    view := m.View() // calling View() directly without driving through Update
}
```
