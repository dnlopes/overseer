---
paths:
  - "internal/adapters/secondary/**/*.go"
---
# SEC-03: One Package Per Technology

## Rule
One Go package per integrated technology: `storage` for persistence, `tmux` for tmux, `git` for git, `agent` for agent backends; don't mix.

## Why
Technology isolation enables replacing one adapter without touching others.

## Example
✅ Good:
```
internal/adapters/secondary/
  storage/   — JSON persistence
  tmux/      — tmux process management
  git/       — git worktree management
  agent/     — agent launcher
```

❌ Bad:
```
internal/adapters/secondary/
  infra/     — storage + tmux + git all in one package
```
