# simpler2sync Design Spec

## Overview

Multi-platform GUI desktop app for bidirectional folder synchronization between local filesystem and Cloudflare R2. Written in Go, uses Fyne for GUI, AWS SDK v2 for R2 operations.

## Tech Stack

- **Language**: Go 1.26+
- **GUI**: Fyne v2 (cross-platform widgets + system tray)
- **R2/S3 Client**: AWS SDK v2 (`github.com/aws/aws-sdk-go-v2/service/s3`)
- **State Storage**: SQLite (`github.com/mattn/go-sqlite3`)
- **Scheduling**: Custom (interval timer + `github.com/robfig/cron/v3` for cron expressions)
- **Config Format**: rclone-compatible INI format

## Architecture

### Components

| Module | Purpose |
|---|---|
| `internal/config` | Parse rclone-style `[r2]` section config, manage app settings |
| `internal/r2client` | Wrap AWS SDK v2 S3 client for R2: list, upload, download, delete objects |
| `internal/sync` | Sync engine: file indexer, remote indexer, diff analyzer, executor |
| `internal/scheduler` | Interval-based + cron-based scheduling, triggers sync tasks |
| `internal/store` | SQLite DB for sync state (local hash, remote ETag, last-sync timestamp) |
| `internal/gui` | Fyne windows: task list, config, logs, settings, system tray |

### Directory Structure

```
simpler2sync/
├── main.go                  # Entry point, app lifecycle
├── go.mod
├── internal/
│   ├── config/
│   │   └── config.go        # INI config parser, R2 profile
│   ├── r2client/
│   │   └── client.go        # S3-compatible client for R2
│   ├── sync/
│   │   ├── engine.go        # Core sync orchestrator
│   │   ├── indexer.go       # Local file scanner + hash
│   │   ├── remote.go        # R2 object lister
│   │   ├── diff.go          # Diff engine (upload/download/delete decisions)
│   │   └── task.go          # Sync task model
│   ├── scheduler/
│   │   └── scheduler.go     # Interval + cron scheduler
│   ├── store/
│   │   └── store.go         # SQLite state persistence
│   └── gui/
│       ├── app.go           # Fyne app + system tray
│       ├── tasklist.go      # Sync task list view
│       ├── taskdialog.go    # Add/edit sync task dialog
│       ├── configpage.go    # R2 config page
│       ├── logpage.go       # Log viewer page
│       ├── settings.go      # Scheduler + conflict settings
│       └── icon.go          # Embedded tray icon
├── assets/
│   └── icon.png             # App icon
└── README.md
```

## Data Flow

```
[Config File] --> Config Module --> R2 Client + Scheduler
                                         |
[Scheduler triggers] --> Sync Engine --> File Indexer (local hash)
                            |              Remote Indexer (ETag list)
                            |
                            v
                      Diff Engine (compare vs State DB)
                            |
                            v
                      Executor (upload/download/delete)
                            |
                            v
                      State DB update (SQLite)
                            |
                            v
                      GUI progress/log update
```

## GUI Design

4-tab main window:

1. **同步任务** - List all sync tasks (local folder, R2 bucket/prefix, status), add/edit/delete/enable-disable per task, manual sync button
2. **R2 配置** - Fields from rclone `[r2]` config: type, provider, access_key_id, secret_access_key, endpoint, region, acl
3. **日志** - Scrollable real-time log viewer with search/filter
4. **设置** - Sync interval (seconds), cron expression, conflict strategy (newer/mirror/ask), concurrent transfers

**System Tray** - Status icon (syncing/idle/error), menu: show window, start sync now, pause/resume, quit

## Sync Engine

### Local Indexer
- Walk directory tree
- Compute SHA256 hash per file
- Track mtime, size, hash

### Remote Indexer
- List objects in bucket prefix
- Collect Key, ETag, LastModified, Size

### Diff Engine
- Compare local index vs remote index vs last state DB
- Output actions: `upload`, `download`, `delete_local`, `delete_remote`
- Conflict: if both local and remote changed since last sync, keep newer by default

### State Tracking (SQLite)
```sql
CREATE TABLE sync_state (
    task_id TEXT,
    local_path TEXT,
    remote_key TEXT,
    local_hash TEXT,
    remote_etag TEXT,
    local_mtime INTEGER,
    remote_mtime INTEGER,
    last_sync_at INTEGER,
    PRIMARY KEY (task_id, local_path, remote_key)
);
```

## Config File Format

```ini
[r2]
type = s3
provider = Cloudflare
access_key_id = YOUR_KEY
secret_access_key = YOUR_SECRET
endpoint = https://<account_id>.r2.cloudflarestorage.com
region = auto
acl = private
```

App config stored alongside in same file or separate JSON:
```json
{
  "sync_tasks": [
    {
      "name": "photos",
      "local_path": "/home/user/photos",
      "remote_bucket": "my-bucket",
      "remote_prefix": "photos/",
      "enabled": true
    }
  ],
  "settings": {
    "interval_seconds": 300,
    "cron_expression": "",
    "conflict_strategy": "newer",
    "concurrent_transfers": 3
  }
}
```

## Cross-Platform Notes

- **Windows**: `.exe` binary, system tray notification area
- **macOS**: `.app` bundle, menu bar icon
- **Linux**: binary, system tray (requires libayatana-appindicator or similar)
- Build with CGO enabled for SQLite + Fyne
- Cross-compile: `GOOS=windows/darwin/linux GOARCH=amd64/arm64`
