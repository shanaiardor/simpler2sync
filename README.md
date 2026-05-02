# simpler2sync

Multi-platform GUI app for bidirectional folder sync between local filesystem and Cloudflare R2.

## Features

- Bidirectional sync (local <-> Cloudflare R2)
- Multi-platform: Windows, macOS, Linux
- System tray / menu bar icon with quick actions
- GUI configuration (no terminal needed)
- Import rclone-style R2 config
- Scheduled sync (interval timer or cron expressions)
- Conflict resolution (newer wins / mirror local)
- Real-time sync log viewer

## Prerequisites

- **Go 1.22+** (for building from source)
- **C compiler** (GCC/MinGW on Windows, Xcode CLI on macOS, build-essential on Linux)
  - Windows: Install [MinGW-w64](https://www.mingw-w64.org/) or via `scoop install mingw`
  - macOS: `xcode-select --install`
  - Linux: `sudo apt install build-essential libgl1-mesa-dev xorg-dev`

## Installation

### Download Binary

Download the latest binary from [Releases](https://github.com/yourname/simpler2sync/releases).

### Build from Source

```bash
git clone https://github.com/yourname/simpler2sync.git
cd simpler2sync
go mod download
go build -o simpler2sync .
```

On Windows:
```powershell
.\build.ps1
```

On macOS/Linux:
```bash
make
```

## Configuration

### R2 Connection

Open the app, go to **R2 Config** tab, fill in:

| Field | Example |
|-------|---------|
| Endpoint | `https://<account_id>.r2.cloudflarestorage.com` |
| Access Key ID | Your R2 access key |
| Secret Access Key | Your R2 secret key |
| Region | `auto` |

Or click "Load from rclone config" to import from your existing rclone configuration.

### Sync Tasks

Go to **Sync Tasks** tab, click **Add Task** and configure:
- **Name**: A label for this sync task
- **Local Path**: The local folder to sync
- **R2 Bucket**: R2 bucket name
- **R2 Prefix**: Path prefix in the bucket (e.g., `backups/photos/`)

### Scheduling

Go to **Settings** tab:
- **Interval (seconds)**: How often to sync (e.g., 300 for every 5 minutes)
- **Cron Expression**: Exact schedule (e.g., `0 2 * * *` for daily at 2 AM)
- **Conflict Strategy**: `newer` (keep newest) or `mirror` (local wins)
- **Concurrent Transfers**: Number of parallel uploads/downloads

## How It Works

1. **Local Indexing**: Scans local folder, computes SHA256 hash per file
2. **Remote Indexing**: Lists R2 bucket objects with ETag and timestamps
3. **Diff Analysis**: Compares local/remote indices against last sync state (SQLite)
4. **Execution**: Uploads, downloads, and deletes as needed
5. **State Update**: Records new hashes and ETags for next sync

## Project Structure

```
simpler2sync/
├── main.go                    # Entry point
├── internal/
│   ├── config/config.go       # Config parsing (JSON + rclone INI)
│   ├── r2client/client.go     # S3-compatible R2 client (AWS SDK v2)
│   ├── sync/
│   │   ├── engine.go          # Sync orchestrator
│   │   ├── indexer.go         # Local file scanner + hash
│   │   ├── remote.go          # Remote object lister
│   │   ├── diff.go            # Change detection
│   │   └── executor.go        # Upload/download execution
│   ├── scheduler/scheduler.go # Interval + cron scheduling
│   ├── store/store.go         # SQLite sync state storage
│   └── gui/
│       ├── app.go             # Fyne app + system tray
│       ├── tasklist.go        # Task list UI
│       ├── taskdialog.go      # Add/edit task dialog
│       ├── configpage.go      # R2 config page
│       ├── logpage.go         # Log viewer
│       ├── settings.go        # Settings page
│       └── icon.go            # App icon loader
└── assets/icon.png            # App icon
```

## License

MIT
