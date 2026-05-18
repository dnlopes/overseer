---
paths:
  - "internal/adapters/primary/tui/**/*.go"
---
# TUI-10: Themes in Theme Files

## Rule
Theme structs in `styles/theme.go`; per-theme palettes in `styles/theme_<name>.go`; `styles.New()` consumes a theme; never hard-code colors elsewhere.

## Why
All color decisions in one place enables theme switching without touching feature code.

## Example
✅ Good:
```go
// internal/adapters/primary/tui/styles/theme_dark.go
func DarkTheme() Theme { return Theme{Primary: lipgloss.Color("#7C3AED"), ...} }

// internal/adapters/primary/tui/styles/styles.go
func New() *Styles { theme := LoadTheme("dark"); return &Styles{...} }
```

❌ Bad:
```go
// internal/adapters/primary/tui/session/list.go
selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")) // WRONG
```
