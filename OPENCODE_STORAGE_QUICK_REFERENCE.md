# OpenCode Session Storage - Quick Reference for Overseer.TUI

## TL;DR: How to Find OpenCode's Database

```bash
# Get the path OpenCode will use
opencode db path

# Query sessions
sqlite3 "$(opencode db path)" "SELECT id, title, directory FROM session LIMIT 10;"
```

---

## Environment Variables (Highest to Lowest Precedence)

| Variable | Effect | Example |
|----------|--------|---------|
| `OPENCODE_DB` | **Direct DB path** (absolute, relative, or `:memory:`) | `OPENCODE_DB=/tmp/test.db opencode` |
| `XDG_DATA_HOME` | Data directory (Linux/macOS/Windows) | `XDG_DATA_HOME=/custom opencode` |
| `HOME` | Home directory (fallback) | `HOME=/home/user opencode` |
| Installation Channel | DB filename variant | `opencode.db` (latest/beta/prod) vs `opencode-{channel}.db` |

---

## Default Database Paths

| Platform | Path |
|----------|------|
| **Linux** | `~/.local/share/opencode/opencode.db` |
| **macOS** | `~/Library/Application Support/opencode/opencode.db` |
| **Windows (CLI)** | `%APPDATA%\opencode\opencode.db` |
| **Windows (Desktop)** | `%APPDATA%\ai.opencode.desktop\opencode.db` |

---

## For Overseer.TUI: Launching OpenCode in tmux

### Option 1: Use Default Database
```bash
tmux new-session -d -s opencode-agent "opencode serve"
```

### Option 2: Isolated Database Per Session
```bash
# Launch with custom DB
tmux new-session -d -s opencode-agent \
  "OPENCODE_DB=/tmp/opencode-session-$$.db opencode serve"

# Query sessions
sqlite3 "/tmp/opencode-session-$$.db" \
  "SELECT id, title FROM session ORDER BY time_created DESC;"

# Merge back to global DB after completion
sqlite3 ~/.local/share/opencode/opencode.db << SQL
ATTACH DATABASE '/tmp/opencode-session-$$.db' AS session_db;
INSERT OR IGNORE INTO main.session SELECT * FROM session_db.session;
INSERT OR IGNORE INTO main.message SELECT * FROM session_db.message;
INSERT OR IGNORE INTO main.part SELECT * FROM session_db.part;
INSERT OR IGNORE INTO main.project SELECT * FROM session_db.project;
DETACH DATABASE session_db;
SQL
```

### Option 3: Isolated Data Directory Per Session
```bash
# Launch with custom data directory
tmux new-session -d -s opencode-agent \
  "XDG_DATA_HOME=/tmp/opencode-$$ opencode serve"

# Database will be at: /tmp/opencode-$$/opencode/opencode.db
```

---

## SQLite Schema (Key Tables)

### `session` Table
```sql
-- Primary session metadata
id TEXT PRIMARY KEY,           -- Session ID
project_id TEXT NOT NULL,      -- Project reference
directory TEXT NOT NULL,       -- Working directory (session binding)
parent_id TEXT,                -- Parent session (for forks)
workspace_id TEXT,             -- Optional workspace
title TEXT NOT NULL,
time_created INTEGER,
time_updated INTEGER
```

### `message` Table
```sql
-- Individual messages in a session
id TEXT PRIMARY KEY,
session_id TEXT NOT NULL,      -- Foreign key to session
data TEXT,                     -- JSON message content
time_created INTEGER
```

### `part` Table
```sql
-- Message parts (tool outputs, code blocks)
id TEXT PRIMARY KEY,
message_id TEXT NOT NULL,      -- Foreign key to message
session_id TEXT NOT NULL,      -- Denormalized for queries
data TEXT,                     -- JSON part content
time_created INTEGER
```

### `project` Table
```sql
-- Project metadata
id TEXT PRIMARY KEY,
worktree TEXT NOT NULL,        -- Git root directory
vcs TEXT,                      -- VCS type (git, etc.)
name TEXT
```

---

## Key Insights

1. **Sessions are directory-scoped**: `opencode session list` only shows sessions for the current directory
2. **Sessions bind to projects**: A project is identified by its git root (`worktree`)
3. **No CLI flags for storage**: Use environment variables only
4. **Channel-based DB naming**: Different channels use different DB files to avoid conflicts
5. **WAL mode with 5s timeout**: Concurrent readers OK, but only one writer at a time
6. **No `OPENCODE_DATA_DIR` variable**: Use `XDG_DATA_HOME` instead

---

## Troubleshooting

### Find which DB a running OpenCode process is using
```bash
# Get the PID
PID=$(pgrep -f "opencode serve")

# Check open files
lsof -p $PID | grep "\.db"

# Or check environment
cat /proc/$PID/environ | tr '\0' '\n' | grep OPENCODE
```

### Merge databases after concurrent runs
```bash
# If you ran multiple OpenCode instances with different DBs
sqlite3 ~/.local/share/opencode/opencode.db << EOF
ATTACH DATABASE '/tmp/opencode-1.db' AS db1;
ATTACH DATABASE '/tmp/opencode-2.db' AS db2;
INSERT OR IGNORE INTO main.session SELECT * FROM db1.session;
INSERT OR IGNORE INTO main.session SELECT * FROM db2.session;
INSERT OR IGNORE INTO main.message SELECT * FROM db1.message;
INSERT OR IGNORE INTO main.message SELECT * FROM db2.message;
INSERT OR IGNORE INTO main.part SELECT * FROM db1.part;
INSERT OR IGNORE INTO main.part SELECT * FROM db2.part;
DETACH DATABASE db1;
DETACH DATABASE db2;
EOF
```

---

## References

- **Full Report**: See `OPENCODE_STORAGE_RESEARCH.md`
- **Official Docs**: https://opencode.ai/docs/cli/
- **Source Code**: https://github.com/anomalyco/opencode (commit: b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a)
