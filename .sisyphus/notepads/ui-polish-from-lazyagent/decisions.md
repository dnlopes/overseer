# Decisions — ui-polish-from-lazyagent

## Palette Design
- 15 semantic color tokens (not 14 as initially estimated)
- Primary: "#7C3AED" (deep purple)
- Accent: "#10B981" (green)
- Warning: "#F59E0B" (amber)
- Muted: "#6B7280" (mid-gray)
- Text: "#F9FAFB" (near-white)
- Subtext: "#9CA3AF" (light gray)
- Border: "#374151" (dark gray — used for BOTH normal and dimmed borders)
- BorderFocus: "#7C3AED" (matches Primary)
- SelectionBg: "#3730A3" (dark purple, distinct from Primary)
- TitleText: "#F9FAFB" (white on Primary background)
- TitleSubtext: "#E0E7FF" (light purple)
- HelpBg: "#111827" (very dark, near-black)
- HelpKeyBg: "#1F2937" (one shade lighter than HelpBg)
- ModalBg: "#1F2937" (modal interior fill)
- OverlayBg: "#111827" (whitespace fill around centered modals)

## Architecture Decisions
- `components/` package = pure style-consumers (no lipgloss.NewStyle() calls)
- `titlebar/` = sub-model (stateful: branding + active pane indicator) — NOT in components/
- No palette.go — only theme.go + theme_dark.go
- No light theme implementation — only LoadTheme switch shape
- No activity color tokens (YAGNI)
- Help bar uses bubblehelp.Styles{} (NOT custom rendering)
- Blurred borders = visible-but-dimmed RoundedBorder (NOT HiddenBorder)

## Scope Boundaries
- NO layout math changes (40/60 split stays)
- NO keybinding changes
- NO new screens or features
- NO domain or service layer changes
- NO secondary adapter changes
- NO new go.mod dependencies

## Task 4 — ANSI-preserving golden helper
- Chose a separate `RequireEqualColor` helper instead of changing existing layout golden behavior so strip-ANSI tests remain stable.
- Color golden files live under `testdata/golden/color/` to avoid colliding with existing Charm `x/exp/golden` path conventions.
