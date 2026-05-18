---
paths:
  - "internal/adapters/primary/tui/**/*.go"
---
# TUI-02: Feature Package Shape

## Rule
Each feature package contains: `model.go` (Model + Init/Update/View), `messages.go` (typed messages), keys in `shared/keys.go` or a per-feature KeyMap, and optional `*_form.go` for forms.

## Why
Consistent package shape lets agents navigate any feature without reading the whole codebase.

## Example
✅ Good:
```
internal/adapters/primary/tui/session/
  list.go          — Model + Init/Update/View
  messages.go      — SessionCreatedMsg, CancelFormMsg
  create_form.go   — CreateFormModel
```

❌ Bad:
```
internal/adapters/primary/tui/session/
  session.go       — 500-line file mixing model, messages, form
```
