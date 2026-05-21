# OpenCode Session Storage Architecture - Complete Reference

**Report Date**: May 21, 2026  
**OpenCode Repository**: https://github.com/anomalyco/opencode  
**Commit**: b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a  
**Official Docs**: https://opencode.ai/docs/cli/

---

## 1. DEFAULT STORAGE PATHS

### By Platform

OpenCode uses **XDG Base Directory Specification** on all platforms (Linux, macOS, Windows), via the `xdg-basedir` npm package (v5.1.0).

#### Linux
```
Data (SQLite DB):     ~/.local/share/opencode/opencode.db
Config:               ~/.config/opencode/
State:                ~/.local/state/opencode/
Cache:                ~/.cache/opencode/
Logs:                 ~/.local/share/opencode/log/
```

#### macOS
```
Data (SQLite DB):     ~/Library/Application Support/opencode/opencode.db
Config:               ~/Library/Preferences/opencode/
State:                ~/Library/Application Support/opencode/
Cache:                ~/Library/Caches/opencode/
Logs:                 ~/Library/Application Support/opencode/log/
```

#### Windows (CLI/WSL)
```
Data (SQLite DB):     %APPDATA%\opencode\opencode.db
Config:               %APPDATA%\opencode\
State:                %APPDATA%\opencode\
Cache:                %LOCALAPPDATA%\cache\opencode\
Logs:                 %APPDATA%\opencode\log\
```

#### Windows (Desktop App)
```
Data (SQLite DB):     %APPDATA%\ai.opencode.desktop\opencode.db
Global State:         %APPDATA%\ai.opencode.desktop\opencode.global.dat
```

**Source**: 
- [packages/core/src/global.ts](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/core/src/global.ts#L10-L14)
- [xdg-basedir source](https://github.com/sindresorhus/xdg-basedir/blob/main/index.js)

---

## 2. ENVIRONMENT VARIABLES THAT CHANGE STORAGE LOCATION

### Data Directory Variables

| Variable | Type | Effect | Precedence |
|----------|------|--------|-----------|
| `XDG_DATA_HOME` | string | Overrides default data directory (Linux/macOS/Windows) | **Highest** |
| `XDG_CONFIG_HOME` | string | Overrides default config directory | **Highest** |
| `XDG_STATE_HOME` | string | Overrides default state directory | **Highest** |
| `XDG_CACHE_HOME` | string | Overrides default cache directory | **Highest** |
| `HOME` | string | Base home directory (used if XDG vars not set) | **High** |
| `OPENCODE_CONFIG_DIR` | string | Override config directory (evaluated at runtime) | **Very High** |
| `OPENCODE_DB` | string | **Direct database file path override** | **Absolute Highest** |

### Database-Specific Variables

#### `OPENCODE_DB` (Most Important)
- **Type**: string
- **Effect**: Directly specifies the SQLite database file path
- **Behavior**:
  - If value is `:memory:` → uses in-memory database
  - If value is absolute path → uses that path directly
  - If value is relative path → resolved relative to `Global.Path.data`
  - If not set → uses channel-based path (see below)
- **Precedence**: **ABSOLUTE HIGHEST** — overrides all other path logic
- **Source**: [packages/core/src/flag/flag.ts:42](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/core/src/flag/flag.ts#L42)
- **Implementation**: [packages/opencode/src/storage/db.ts:38-44](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/storage/db.ts#L38-L44)

```typescript
export const getPath = (flags?: Pick<DatabaseFlags, "disableChannelDb">) => {
  if (Flag.OPENCODE_DB) {
    if (Flag.OPENCODE_DB === ":memory:" || path.isAbsolute(Flag.OPENCODE_DB)) 
      return Flag.OPENCODE_DB
    return path.join(Global.Path.data, Flag.OPENCODE_DB)
  }
  return getChannelPath(flags)
}
```

#### `OPENCODE_CONFIG_DIR`
- **Type**: string
- **Effect**: Overrides config directory path
- **Precedence**: Very high (used in `Global.make()`)
- **Source**: [packages/core/src/global.ts:63](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/core/src/global.ts#L63)

### Channel-Based Database Naming

If `OPENCODE_DB` is **not** set, the database filename depends on the **installation channel**:

```typescript
export function getChannelPath(flags: Pick<DatabaseFlags, "disableChannelDb"> = readRuntimeFlags()) {
  if (["latest", "beta", "prod"].includes(InstallationChannel) || flags.disableChannelDb)
    return path.join(Global.Path.data, "opencode.db")
  const safe = InstallationChannel.replace(/[^a-zA-Z0-9._-]/g, "-")
  return path.join(Global.Path.data, `opencode-${safe}.db`)
}
```

**Channel Mapping**:
- `latest` → `opencode.db`
- `beta` → `opencode.db`
- `prod` → `opencode.db`
- `stable` → `opencode-stable.db`
- `local` → `opencode-local.db`
- `dev` → `opencode-dev.db`
- Any other channel → `opencode-{sanitized-channel}.db`

**Source**: [packages/opencode/src/storage/db.ts:31-36](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/storage/db.ts#L31-L36)

### Other Environment Variables (Config/Behavior, Not Storage Location)

| Variable | Type | Effect |
|----------|------|--------|
| `OPENCODE_CONFIG` | string | Path to config file (not data directory) |
| `OPENCODE_CONFIG_CONTENT` | string | Inline JSON config content |
| `OPENCODE_TUI_CONFIG` | string | Path to TUI-specific config |
| `OPENCODE_PERMISSION` | string | Inline JSON permissions config |
| `OPENCODE_DISABLE_PROJECT_CONFIG` | boolean | Disable project-level config files |
| `OPENCODE_DISABLE_CHANNEL_DB` | boolean | Force use of `opencode.db` regardless of channel |
| `OPENCODE_TEST_HOME` | string | Override home directory (testing only) |

**Source**: [packages/core/src/flag/flag.ts](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/core/src/flag/flag.ts)

---

## 3. CLI FLAGS THAT AFFECT STORAGE

**None.** OpenCode CLI does not expose `--data-dir`, `--config-dir`, or `--db-path` flags.

Storage location is **exclusively** controlled via environment variables.

**Source**: [packages/opencode/src/cli/cmd/db.ts](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/cli/cmd/db.ts) — the `db` command only provides `path`, `migrate`, and query subcommands; no path configuration flags.

---

## 4. SQLITE SCHEMA FOR SESSIONS

### Core Tables

#### `session` Table
**Primary table for session metadata.**

```sql
CREATE TABLE session (
  id TEXT PRIMARY KEY,                    -- Session ID (e.g., "sess_abc123")
  project_id TEXT NOT NULL,               -- Foreign key to project table
  workspace_id TEXT,                      -- Optional workspace ID
  parent_id TEXT,                         -- Parent session ID (for forked sessions)
  slug TEXT NOT NULL,                     -- URL-safe session slug
  directory TEXT NOT NULL,                -- Working directory where session was created
  path TEXT,                              -- Optional file path
  title TEXT NOT NULL,                    -- Session title
  version TEXT NOT NULL,                  -- Schema version
  share_url TEXT,                         -- Share link URL
  summary_additions INTEGER,              -- Code diff stats
  summary_deletions INTEGER,
  summary_files INTEGER,
  summary_diffs TEXT,                     -- JSON array of file diffs
  cost REAL DEFAULT 0,                    -- Token cost
  tokens_input INTEGER DEFAULT 0,         -- Input tokens
  tokens_output INTEGER DEFAULT 0,        -- Output tokens
  tokens_reasoning INTEGER DEFAULT 0,     -- Reasoning tokens
  tokens_cache_read INTEGER DEFAULT 0,    -- Cache read tokens
  tokens_cache_write INTEGER DEFAULT 0,   -- Cache write tokens
  revert TEXT,                            -- JSON: revert state
  permission TEXT,                        -- JSON: permission ruleset
  agent TEXT,                             -- Agent name
  model TEXT,                             -- JSON: {id, providerID, variant}
  time_created INTEGER NOT NULL,          -- Creation timestamp (ms)
  time_updated INTEGER NOT NULL,          -- Last update timestamp (ms)
  time_compacting INTEGER,                -- Compaction timestamp
  time_archived INTEGER,                  -- Archive timestamp
  
  FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
  INDEX session_project_idx ON (project_id),
  INDEX session_workspace_idx ON (workspace_id),
  INDEX session_parent_idx ON (parent_id)
);
```

**Key Columns**:
- `directory` — **The working directory where the session was created** (used to bind sessions to projects)
- `parent_id` — Links to parent session (for forked/continued sessions)
- `workspace_id` — Optional workspace association

**Source**: [packages/opencode/src/session/session.sql.ts:16-59](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/session/session.sql.ts#L16-L59)

#### `message` Table
**Stores individual messages in a session.**

```sql
CREATE TABLE message (
  id TEXT PRIMARY KEY,                    -- Message ID
  session_id TEXT NOT NULL,               -- Foreign key to session
  time_created INTEGER NOT NULL,          -- Creation timestamp
  time_updated INTEGER NOT NULL,          -- Update timestamp
  data TEXT NOT NULL,                     -- JSON: message content/metadata
  
  FOREIGN KEY (session_id) REFERENCES session(id) ON DELETE CASCADE,
  INDEX message_session_time_created_id_idx ON (session_id, time_created, id)
);
```

**Source**: [packages/opencode/src/session/session.sql.ts:61-73](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/session/session.sql.ts#L61-L73)

#### `part` Table
**Stores message parts (tool outputs, code blocks, etc.).**

```sql
CREATE TABLE part (
  id TEXT PRIMARY KEY,                    -- Part ID
  message_id TEXT NOT NULL,               -- Foreign key to message
  session_id TEXT NOT NULL,               -- Denormalized session ID
  time_created INTEGER NOT NULL,          -- Creation timestamp
  time_updated INTEGER NOT NULL,          -- Update timestamp
  data TEXT NOT NULL,                     -- JSON: part content
  
  FOREIGN KEY (message_id) REFERENCES message(id) ON DELETE CASCADE,
  INDEX part_message_id_id_idx ON (message_id, id),
  INDEX part_session_idx ON (session_id)
);
```

**Source**: [packages/opencode/src/session/session.sql.ts:75-91](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/session/session.sql.ts#L75-L91)

#### `todo` Table
**Stores session-level todos.**

```sql
CREATE TABLE todo (
  session_id TEXT NOT NULL,               -- Foreign key to session
  content TEXT NOT NULL,                  -- Todo text
  status TEXT NOT NULL,                   -- Status (pending, completed, etc.)
  priority TEXT NOT NULL,                 -- Priority level
  position INTEGER NOT NULL,              -- Order in list
  time_created INTEGER NOT NULL,
  time_updated INTEGER NOT NULL,
  
  PRIMARY KEY (session_id, position),
  FOREIGN KEY (session_id) REFERENCES session(id) ON DELETE CASCADE,
  INDEX todo_session_idx ON (session_id)
);
```

**Source**: [packages/opencode/src/session/session.sql.ts:93-110](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/session/session.sql.ts#L93-L110)

#### `project` Table
**Stores project metadata (directory, VCS info, etc.).**

```sql
CREATE TABLE project (
  id TEXT PRIMARY KEY,                    -- Project ID
  worktree TEXT NOT NULL,                 -- Git worktree path
  vcs TEXT,                               -- VCS type (git, etc.)
  name TEXT,                              -- Project name
  icon_url TEXT,                          -- Icon URL
  icon_url_override TEXT,                 -- Custom icon URL
  icon_color TEXT,                        -- Icon color
  time_created INTEGER NOT NULL,
  time_updated INTEGER NOT NULL,
  time_initialized INTEGER,               -- Initialization timestamp
  sandboxes TEXT,                         -- JSON: sandbox list
  commands TEXT                           -- JSON: {start?: string}
);
```

**Key Column**:
- `worktree` — **The project's root directory** (sessions are bound to projects via `project_id`)

**Source**: [packages/opencode/src/project/project.sql.ts](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/project/project.sql.ts)

#### `workspace` Table
**Stores workspace metadata (branches, directories within a project).**

```sql
CREATE TABLE workspace (
  id TEXT PRIMARY KEY,                    -- Workspace ID
  type TEXT NOT NULL,                     -- Workspace type
  name TEXT DEFAULT "",                   -- Workspace name
  branch TEXT,                            -- Git branch
  directory TEXT,                         -- Directory within project
  extra TEXT,                             -- JSON: extra metadata
  project_id TEXT NOT NULL,               -- Foreign key to project
  time_used INTEGER NOT NULL,             -- Last used timestamp
  
  FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
);
```

**Source**: [packages/opencode/src/control-plane/workspace.sql.ts](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/control-plane/workspace.sql.ts)

### Relationship Diagram

```
project (worktree)
  ├── session (project_id, directory, parent_id)
  │   ├── message (session_id)
  │   │   └── part (message_id, session_id)
  │   └── todo (session_id)
  └── workspace (project_id, directory, branch)
```

---

## 5. HOW OPENCODE DISCOVERS/USES A PROJECT

### Project Discovery

1. **Working Directory Binding**: When OpenCode starts in a directory, it:
   - Walks up the directory tree to find `.git` (or other VCS markers)
   - Creates/retrieves a `project` record with `worktree` = git root
   - All sessions created in that directory are bound to that project via `project_id`

2. **Workspace Concept**: 
   - A **workspace** is a sub-directory or branch within a project
   - Workspaces track `directory` (subdirectory within project) and `branch` (git branch)
   - Sessions can be associated with a workspace via `workspace_id`
   - **Workspaces are optional** — sessions work without them

3. **Session Binding**:
   - Sessions are bound to a project (required: `project_id`)
   - Sessions store the `directory` where they were created
   - Sessions can optionally reference a `workspace_id`
   - **Sessions are directory-scoped**: `opencode session list` only shows sessions for the current directory

### Configuration Files

**Per-Project Config** (`opencode.json` in project root):
- Highest precedence among standard config files
- Overrides global and remote configs
- Stored on disk, not in database

**Global Config** (`~/.config/opencode/opencode.json`):
- User-wide settings
- Stored on disk, not in database

**Project-Specific Agents** (`.opencode/` directory):
- Custom agents, commands, plugins
- Stored on disk, not in database

**Source**: [packages/opencode/src/config/paths.ts](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/config/paths.ts)

---

## 6. PRECEDENCE ORDER FOR STORAGE LOCATION

**From highest to lowest precedence:**

1. **`OPENCODE_DB` environment variable** (if set)
   - Absolute path: used directly
   - Relative path: resolved relative to `Global.Path.data`
   - `:memory:`: in-memory database

2. **`XDG_DATA_HOME` environment variable** (if set)
   - Determines `Global.Path.data`
   - Database path: `$XDG_DATA_HOME/opencode/opencode.db` (or channel variant)

3. **`HOME` environment variable** (if set)
   - Fallback if XDG vars not set
   - Database path: `$HOME/.local/share/opencode/opencode.db` (Linux) or platform-specific

4. **Installation Channel** (if `OPENCODE_DB` not set)
   - `latest`, `beta`, `prod` → `opencode.db`
   - Other channels → `opencode-{channel}.db`

5. **Platform Defaults** (if no env vars set)
   - Linux: `~/.local/share/opencode/opencode.db`
   - macOS: `~/Library/Application Support/opencode/opencode.db`
   - Windows: `%APPDATA%\opencode\opencode.db`

**Source**: [packages/opencode/src/storage/db.ts:31-44](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/storage/db.ts#L31-L44)

---

## 7. HOW TO INSPECT A RUNNING OPENCODE PROCESS

### CLI Command: `opencode db path`

**Prints the database path that OpenCode is using:**

```bash
$ opencode db path
/Users/dnl/.local/share/opencode/opencode.db
```

**Source**: [packages/opencode/src/cli/cmd/db.ts:57-63](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/cli/cmd/db.ts#L57-L63)

### Query the Database

**List all sessions:**
```bash
sqlite3 ~/.local/share/opencode/opencode.db "SELECT id, directory, title FROM session ORDER BY time_created DESC;"
```

**List sessions for a specific directory:**
```bash
sqlite3 ~/.local/share/opencode/opencode.db "SELECT id, title FROM session WHERE directory = '/path/to/project';"
```

**Check database file size and row counts:**
```bash
sqlite3 ~/.local/share/opencode/opencode.db << EOF
.tables
SELECT COUNT(*) as session_count FROM session;
SELECT COUNT(*) as message_count FROM message;
SELECT COUNT(*) as part_count FROM part;
PRAGMA page_count;
PRAGMA page_size;
EOF
```

### Logs

OpenCode writes logs to:
- `~/.local/share/opencode/log/` (Linux)
- `~/Library/Application Support/opencode/log/` (macOS)
- `%APPDATA%\opencode\log\` (Windows)

Logs include database initialization messages:
```
[db] opening database { path: '/Users/dnl/.local/share/opencode/opencode.db' }
[db] applying migrations { count: 22, mode: 'dev' }
```

**Source**: [packages/opencode/src/storage/db.ts:100](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/storage/db.ts#L100)

### Process Inspection (Linux/macOS)

```bash
# Find OpenCode process
ps aux | grep opencode

# Check environment variables
cat /proc/<pid>/environ | tr '\0' '\n' | grep OPENCODE

# Check open files
lsof -p <pid> | grep opencode.db
```

---

## 8. MULTIPLE OPENCODE VERSION COEXISTENCE

### Can Two Installs Share the Same Database?

**No, not safely.** Here's why:

1. **Channel-Based Separation**: Different installation channels use different database files:
   - `latest` → `opencode.db`
   - `stable` → `opencode-stable.db`
   - `local` → `opencode-local.db`
   - `dev` → `opencode-dev.db`

2. **Schema Migrations**: Each version may have different database schemas. Concurrent access to the same database file can cause:
   - `SQLITE_BUSY` errors (WAL lock contention)
   - Schema mismatch errors
   - Data corruption

3. **WAL Mode**: OpenCode uses SQLite WAL (Write-Ahead Logging) with `busy_timeout = 5000ms`:
   ```sql
   PRAGMA journal_mode = WAL;
   PRAGMA synchronous = NORMAL;
   PRAGMA busy_timeout = 5000;
   ```
   This allows concurrent readers but only **one writer at a time**. Multiple processes writing simultaneously will experience lock contention.

### Recommended Approach for Multiple Versions

**Use separate databases per version:**

```bash
# Stable version
OPENCODE_DB=opencode-stable.db opencode ...

# Dev version
OPENCODE_DB=opencode-dev.db opencode ...

# Or use XDG_DATA_HOME isolation
XDG_DATA_HOME=/tmp/opencode-dev opencode ...
```

**Merge sessions after completion:**
```bash
sqlite3 ~/.local/share/opencode/opencode.db << EOF
ATTACH DATABASE '/tmp/opencode-dev/.local/share/opencode/opencode.db' AS dev;
INSERT OR IGNORE INTO main.session SELECT * FROM dev.session;
INSERT OR IGNORE INTO main.message SELECT * FROM dev.message;
INSERT OR IGNORE INTO main.part SELECT * FROM dev.part;
DETACH DATABASE dev;
EOF
```

**Source**: 
- [packages/opencode/src/storage/db.ts:31-36](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/storage/db.ts#L31-L36) (channel-based naming)
- [packages/opencode/src/storage/db.ts:104-109](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/storage/db.ts#L104-L109) (WAL configuration)
- GitHub Issue [#21215](https://github.com/anomalyco/opencode/issues/21215) (SQLITE_BUSY with concurrent sessions)
- GitHub Issue [#21790](https://github.com/anomalyco/opencode/issues/21790) (database file migration issues)

---

## 9. SUMMARY TABLE: ENVIRONMENT VARIABLE PRECEDENCE

| Rank | Variable | Type | Effect | Example |
|------|----------|------|--------|---------|
| 1 | `OPENCODE_DB` | string | Direct DB path override | `OPENCODE_DB=/tmp/test.db opencode` |
| 2 | `XDG_DATA_HOME` | string | Data directory override | `XDG_DATA_HOME=/custom/data opencode` |
| 3 | `XDG_CONFIG_HOME` | string | Config directory override | `XDG_CONFIG_HOME=/custom/config opencode` |
| 4 | `HOME` | string | Home directory (fallback) | `HOME=/home/user opencode` |
| 5 | Installation Channel | (built-in) | DB filename variant | `opencode-stable.db` vs `opencode-dev.db` |
| 6 | Platform Default | (built-in) | OS-specific path | `~/.local/share/opencode/` (Linux) |

---

## 10. OVERSEER.TUI INTEGRATION GUIDE

For Overseer.TUI launching OpenCode as a tmux agent:

### Step 1: Determine the Database Path

```bash
# Get the path OpenCode will use
DB_PATH=$(OPENCODE_DB="${CUSTOM_DB:-}" opencode db path)
echo "OpenCode will use: $DB_PATH"
```

### Step 2: Set Environment Variables Before Launch

```bash
# Option A: Use default database
tmux new-session -d -s opencode-agent "opencode serve"

# Option B: Use custom database (isolated per session)
tmux new-session -d -s opencode-agent \
  "OPENCODE_DB=/tmp/opencode-session-$$.db opencode serve"

# Option C: Use custom data directory
tmux new-session -d -s opencode-agent \
  "XDG_DATA_HOME=/tmp/opencode-$$ opencode serve"
```

### Step 3: Query Sessions After Completion

```bash
# Get the database path from the running process
DB_PATH=$(lsof -p $(pgrep -f "opencode serve") 2>/dev/null | grep "\.db" | awk '{print $NF}' | head -1)

# Or use the path you set:
DB_PATH="/tmp/opencode-session-$$.db"

# Query sessions
sqlite3 "$DB_PATH" "SELECT id, title, directory FROM session ORDER BY time_created DESC LIMIT 10;"
```

### Step 4: Merge Sessions Back to Global Database

```bash
# After OpenCode process exits
GLOBAL_DB="$HOME/.local/share/opencode/opencode.db"
SESSION_DB="/tmp/opencode-session-$$.db"

sqlite3 "$GLOBAL_DB" << EOF
ATTACH DATABASE '$SESSION_DB' AS session_db;
INSERT OR IGNORE INTO main.session SELECT * FROM session_db.session;
INSERT OR IGNORE INTO main.message SELECT * FROM session_db.message;
INSERT OR IGNORE INTO main.part SELECT * FROM session_db.part;
INSERT OR IGNORE INTO main.project SELECT * FROM session_db.project;
DETACH DATABASE session_db;
EOF

# Clean up
rm "$SESSION_DB"
```

---

## 11. KNOWN ISSUES & WORKAROUNDS

### Issue: Sessions Lost After Update (Channel DB Mismatch)

**Problem**: Upgrading OpenCode switches database file from `opencode.db` to `opencode-prod.db`, orphaning old sessions.

**Workaround**:
```bash
# Merge old data into new database
sqlite3 ~/.local/share/opencode/opencode-prod.db << EOF
ATTACH DATABASE '$HOME/.local/share/opencode/opencode.db' AS old;
INSERT OR IGNORE INTO main.project SELECT * FROM old.project;
INSERT OR IGNORE INTO main.session SELECT * FROM old.session;
INSERT OR IGNORE INTO main.message SELECT * FROM old.message;
INSERT OR IGNORE INTO main.part SELECT * FROM old.part;
DETACH DATABASE old;
EOF
```

**Source**: GitHub Issue [#21790](https://github.com/anomalyco/opencode/issues/21790)

### Issue: SQLITE_BUSY with Concurrent Sessions

**Problem**: Multiple `opencode run` instances writing to the same database simultaneously fail with `SQLITE_BUSY`.

**Workaround**: Use separate databases per worker:
```bash
# Worker 1
XDG_DATA_HOME=/tmp/worker1 opencode run "task 1"

# Worker 2
XDG_DATA_HOME=/tmp/worker2 opencode run "task 2"

# Merge after completion
```

**Source**: GitHub Issue [#21215](https://github.com/anomalyco/opencode/issues/21215)

### Issue: JSON→SQLite Migration Silently Skips

**Problem**: Incremental upgrades skip JSON→SQLite migration if `opencode.db` already exists from prior schema migrations.

**Workaround**: Manually run migration:
```bash
opencode db migrate
```

**Source**: GitHub Issue [#13654](https://github.com/anomalyco/opencode/issues/13654)

---

## 12. REFERENCES

### Official Documentation
- [OpenCode CLI Docs](https://opencode.ai/docs/cli/)
- [OpenCode Config Docs](https://opencode.ai/docs/config/)

### Source Code
- [Global Path Configuration](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/core/src/global.ts)
- [Database Path Logic](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/storage/db.ts)
- [Flag Definitions](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/core/src/flag/flag.ts)
- [Session Schema](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/session/session.sql.ts)
- [DB CLI Command](https://github.com/anomalyco/opencode/blob/b4a01cc3cd6a90fe94fd1b1fc985ddcf9bbd172a/packages/opencode/src/cli/cmd/db.ts)

### Related Issues
- [#21215](https://github.com/anomalyco/opencode/issues/21215) - SQLITE_BUSY with concurrent sessions
- [#21790](https://github.com/anomalyco/opencode/issues/21790) - Database file migration issues
- [#13654](https://github.com/anomalyco/opencode/issues/13654) - JSON→SQLite migration skips
- [#16885](https://github.com/anomalyco/opencode/issues/16885) - Channel-specific DB path issues
- [#25978](https://github.com/anomalyco/opencode/issues/25978) - Session list directory binding

