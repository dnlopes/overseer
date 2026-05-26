# Go Pointer Receivers for Mutating Methods

## Rule

Any Go method that mutates struct fields, sets `activePopup`, assigns to embedded models, or otherwise changes the receiver's state MUST use a pointer receiver (`*Model`). Methods that only read state MAY use value receivers.

## Why

Value receivers operate on a copy. Assignments to `m.field` are silently discarded when the method returns, causing state changes to vanish. This leads to bugs that compile fine but fail at runtime with no error — popups don't open, flags don't toggle, forms don't initialize.

## Incorrect

```go
// BUG: activePopup is set on a copy, never persists
func (m Model) showKillPreviewPopupCmd() tea.Cmd {
    m.activePopup = popupKillPreview  // lost!
    m.killPreviewForm = sessionui.NewKillPreviewForm(...)
    return m.killPreviewForm.Init()
}
```

## Correct

```go
func (m *Model) showKillPreviewPopupCmd() tea.Cmd {
    m.activePopup = popupKillPreview  // persists on the real model
    m.killPreviewForm = sessionui.NewKillPreviewForm(...)
    return m.killPreviewForm.Init()
}
```

## Checklist

Before committing a new method, ask:
1. Does this method assign to any field on the receiver?
2. Does it call another method that mutates the receiver?
3. If yes → use `*Model`. If no → value receiver is safe.
