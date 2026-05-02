# simpler2sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a multi-platform GUI app for bidirectional local-to-Cloudflare R2 folder sync with system tray icon, scheduled sync, and in-app configuration.

**Architecture:** Pure Go with Fyne GUI, AWS SDK v2 for R2 S3-compatible storage, SQLite for sync state tracking. Modular internal packages with clear boundaries: config, r2client, sync engine, scheduler, store, gui.

**Tech Stack:** Go 1.26, Fyne v2, AWS SDK v2 (S3), modernc.org/sqlite (CGo-free), robfig/cron v3, gopkg.in/ini.v1

**Modules:** config → r2client + store → sync → scheduler → gui → main

---

### Task 1: Project Scaffolding

**Files:**
- Create: `main.go`
- Create: `go.mod`
- Create: `assets/icon.png`

- [ ] **Step 1: Initialize Go module**

```bash
cd C:\Users\39\workspace\simpler2sync
go mod init simpler2sync
```

- [ ] **Step 2: Create main.go skeleton**

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	return nil
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build -o simpler2sync.exe .
```
Expected: Builds successfully with no output.

- [ ] **Step 4: Create assets directory and placeholder icon**

```bash
mkdir -p assets
```

Create `assets/icon.png` (a simple 128x128 PNG placeholder).

---

### Task 2: Config Module

**Files:**
- Create: `internal/config/config.go`

- [ ] **Step 1: Add ini dependency**

```bash
go get gopkg.in/ini.v1
```

- [ ] **Step 2: Implement config types and parser**

```go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

type R2Config struct {
	Type            string `ini:"type"`
	Provider        string `ini:"provider"`
	AccessKeyID     string `ini:"access_key_id"`
	SecretAccessKey string `ini:"secret_access_key"`
	Endpoint        string `ini:"endpoint"`
	Region          string `ini:"region"`
	ACL             string `ini:"acl"`
}

type SyncTask struct {
	Name          string `json:"name"`
	LocalPath     string `json:"local_path"`
	RemoteBucket  string `json:"remote_bucket"`
	RemotePrefix  string `json:"remote_prefix"`
	Enabled       bool   `json:"enabled"`
}

type AppConfig struct {
	R2        R2Config   `json:"r2"`
	Tasks     []SyncTask `json:"sync_tasks"`
	Settings  Settings   `json:"settings"`
	configDir string     `json:"-"`
}

type Settings struct {
	IntervalSeconds     int    `json:"interval_seconds"`
	CronExpression      string `json:"cron_expression"`
	ConflictStrategy    string `json:"conflict_strategy"`
	ConcurrentTransfers int    `json:"concurrent_transfers"`
}

func DefaultSettings() Settings {
	return Settings{
		IntervalSeconds:     300,
		ConflictStrategy:    "newer",
		ConcurrentTransfers: 3,
	}
}

func configDir() string {
	d, _ := os.UserConfigDir()
	return filepath.Join(d, "simpler2sync")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func Load() (*AppConfig, error) {
	cfg := &AppConfig{
		configDir: configDir(),
		Settings:  DefaultSettings(),
	}
	if err := os.MkdirAll(cfg.configDir, 0700); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	data, err := os.ReadFile(configPath())
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func (c *AppConfig) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(configPath(), data, 0600)
}

func LoadR2FromINIPath(path string) (*R2Config, error) {
	f, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load ini: %w", err)
	}
	sec := f.Section("r2")
	cfg := &R2Config{
		Type:            sec.Key("type").String(),
		Provider:        sec.Key("provider").String(),
		AccessKeyID:     sec.Key("access_key_id").String(),
		SecretAccessKey: sec.Key("secret_access_key").String(),
		Endpoint:        sec.Key("endpoint").String(),
		Region:          sec.Key("region").String(),
		ACL:             sec.Key("acl").String(),
	}
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("missing endpoint in [r2] section")
	}
	return cfg, nil
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```
Expected: No errors.

---

### Task 3: Store Module (SQLite State DB)

**Files:**
- Create: `internal/store/store.go`

- [ ] **Step 1: Add SQLite dependency (CGo-free)**

```bash
go get modernc.org/sqlite
```

- [ ] **Step 2: Implement store**

```go
package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SyncState struct {
	TaskID       string
	LocalPath    string
	RemoteKey    string
	LocalHash    string
	RemoteETag   string
	LocalMtime   int64
	RemoteMtime  int64
	LastSyncAt   int64
}

type Store struct {
	db *sql.DB
}

func Open(dbDir string) (*Store, error) {
	dbPath := filepath.Join(dbDir, "sync_state.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	q := `CREATE TABLE IF NOT EXISTS sync_state (
		task_id TEXT NOT NULL,
		local_path TEXT NOT NULL,
		remote_key TEXT NOT NULL,
		local_hash TEXT NOT NULL DEFAULT '',
		remote_etag TEXT NOT NULL DEFAULT '',
		local_mtime INTEGER NOT NULL DEFAULT 0,
		remote_mtime INTEGER NOT NULL DEFAULT 0,
		last_sync_at INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (task_id, local_path, remote_key)
	)`
	_, err := db.Exec(q)
	return err
}

func (s *Store) GetState(taskID, localPath, remoteKey string) (*SyncState, error) {
	row := s.db.QueryRow(
		`SELECT task_id, local_path, remote_key, local_hash, remote_etag, local_mtime, remote_mtime, last_sync_at
		 FROM sync_state WHERE task_id=? AND local_path=? AND remote_key=?`,
		taskID, localPath, remoteKey,
	)
	st := &SyncState{}
	err := row.Scan(&st.TaskID, &st.LocalPath, &st.RemoteKey, &st.LocalHash, &st.RemoteETag, &st.LocalMtime, &st.RemoteMtime, &st.LastSyncAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return st, err
}

func (s *Store) UpsertState(st *SyncState) error {
	st.LastSyncAt = time.Now().Unix()
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO sync_state (task_id, local_path, remote_key, local_hash, remote_etag, local_mtime, remote_mtime, last_sync_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		st.TaskID, st.LocalPath, st.RemoteKey, st.LocalHash, st.RemoteETag, st.LocalMtime, st.RemoteMtime, st.LastSyncAt,
	)
	return err
}

func (s *Store) DeleteState(taskID, localPath, remoteKey string) error {
	_, err := s.db.Exec(
		`DELETE FROM sync_state WHERE task_id=? AND local_path=? AND remote_key=?`,
		taskID, localPath, remoteKey,
	)
	return err
}

func (s *Store) ListStatesForTask(taskID string) ([]SyncState, error) {
	rows, err := s.db.Query(
		`SELECT task_id, local_path, remote_key, local_hash, remote_etag, local_mtime, remote_mtime, last_sync_at
		 FROM sync_state WHERE task_id=?`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var states []SyncState
	for rows.Next() {
		var st SyncState
		if err := rows.Scan(&st.TaskID, &st.LocalPath, &st.RemoteKey, &st.LocalHash, &st.RemoteETag, &st.LocalMtime, &st.RemoteMtime, &st.LastSyncAt); err != nil {
			return nil, err
		}
		states = append(states, st)
	}
	return states, rows.Err()
}

func (s *Store) Close() error {
	return s.db.Close()
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```
Expected: No errors.

---

### Task 4: R2 Client Module

**Files:**
- Create: `internal/r2client/client.go`

- [ ] **Step 1: Add AWS SDK v2 dependencies**

```bash
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/s3
go get github.com/aws/aws-sdk-go-v2/credentials
```

- [ ] **Step 2: Implement R2 client**

```go
package r2client

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Client struct {
	client *s3.Client
}

type ObjectInfo struct {
	Key          string
	ETag         string
	Size         int64
	LastModified int64
}

func New(endpoint, accessKey, secretKey, region string) (*Client, error) {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint,
			SigningRegion:     region,
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(resolver),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &Client{client: s3.NewFromConfig(cfg)}, nil
}

func (c *Client) ListObjects(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	var objects []ObjectInfo
	paginator := s3.NewListObjectsV2Paginator(c.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}
		for _, obj := range page.Contents {
			var lastMod int64
			if obj.LastModified != nil {
				lastMod = obj.LastModified.Unix()
			}
			objects = append(objects, ObjectInfo{
				Key:          aws.ToString(obj.Key),
				ETag:         strings.Trim(aws.ToString(obj.ETag), `"`),
				Size:         aws.ToInt64(obj.Size),
				LastModified: lastMod,
			})
		}
	}
	return objects, nil
}

func (c *Client) UploadFile(ctx context.Context, bucket, key string, reader io.Reader, size int64) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          reader,
		ContentLength: aws.Int64(size),
	}
	result, err := c.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("upload %s: %w", key, err)
	}
	return strings.Trim(aws.ToString(result.ETag), `"`), nil
}

func (c *Client) DownloadFile(ctx context.Context, bucket, key string) (io.ReadCloser, int64, error) {
	result, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("download %s: %w", key, err)
	}
	return result.Body, aws.ToInt64(result.ContentLength), nil
}

func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}

func (c *Client) BucketNameFromEndpoint(endpoint string) string {
	u := strings.TrimPrefix(endpoint, "https://")
	u = strings.TrimPrefix(u, "http://")
	return strings.Split(u, ".")[0]
}

func RemoteKey(localRoot, filePath, remotePrefix string) string {
	rel, _ := filepath.Rel(localRoot, filePath)
	rel = filepath.ToSlash(rel)
	return filepath.Join(remotePrefix, rel)
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```
Expected: No errors.

---

### Task 5: Sync Engine — Indexer

**Files:**
- Create: `internal/sync/indexer.go`
- Create: `internal/sync/remote.go`

- [ ] **Step 1: Implement local file indexer**

```go
package sync

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type FileInfo struct {
	Path    string
	Size    int64
	ModTime int64
	Hash    string
}

func WalkLocal(root string) ([]FileInfo, error) {
	var files []FileInfo
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		hash, err := fileHash(path)
		if err != nil {
			return fmt.Errorf("hash %s: %w", path, err)
		}
		files = append(files, FileInfo{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
			Hash:    hash,
		})
		return nil
	})
	return files, err
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	buf := make([]byte, 1<<20)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			h.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
```

- [ ] **Step 2: Implement remote file indexer wrapper**

```go
package sync

import (
	"context"
	"fmt"

	"simpler2sync/internal/r2client"
)

func ListRemote(ctx context.Context, client *r2client.Client, bucket, prefix string) (map[string]r2client.ObjectInfo, error) {
	objects, err := client.ListObjects(ctx, bucket, prefix)
	if err != nil {
		return nil, fmt.Errorf("list remote: %w", err)
	}
	result := make(map[string]r2client.ObjectInfo, len(objects))
	for _, obj := range objects {
		result[obj.Key] = obj
	}
	return result, nil
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```
Expected: No errors.

---

### Task 6: Sync Engine — Diff, Executor, Orchestrator

**Files:**
- Create: `internal/sync/diff.go`
- Create: `internal/sync/executor.go`
- Create: `internal/sync/engine.go`

- [ ] **Step 1: Implement diff engine**

```go
package sync

import (
	"simpler2sync/internal/r2client"
	"simpler2sync/internal/store"
)

type ActionType string

const (
	ActionUpload       ActionType = "upload"
	ActionDownload     ActionType = "download"
	ActionDeleteLocal  ActionType = "delete_local"
	ActionDeleteRemote ActionType = "delete_remote"
)

type Action struct {
	Type       ActionType
	LocalPath  string
	RemoteKey  string
}

func Diff(localFiles []FileInfo, remoteObjects map[string]r2client.ObjectInfo, states map[string]*store.SyncState, taskID, localRoot, remotePrefix string, conflictStrategy string) []Action {
	var actions []Action
	localMap := make(map[string]FileInfo)
	for _, f := range localFiles {
		key := r2client.RemoteKey(localRoot, f.Path, remotePrefix)
		localMap[key] = f
	}
	remoteKeys := make(map[string]bool)
	for key := range remoteObjects {
		remoteKeys[key] = true
	}

	for key, local := range localMap {
		remote, hasRemote := remoteObjects[key]
		st := states[key]
		if !hasRemote {
			actions = append(actions, Action{Type: ActionUpload, LocalPath: local.Path, RemoteKey: key})
		} else if hasRemote && st != nil {
			localChanged := local.Hash != st.LocalHash
			remoteChanged := remote.ETag != st.RemoteETag
			if localChanged && !remoteChanged {
				actions = append(actions, Action{Type: ActionUpload, LocalPath: local.Path, RemoteKey: key})
			} else if !localChanged && remoteChanged {
				actions = append(actions, Action{Type: ActionDownload, LocalPath: local.Path, RemoteKey: key})
			} else if localChanged && remoteChanged {
				switch conflictStrategy {
				case "newer":
					if local.ModTime > st.RemoteMtime {
						actions = append(actions, Action{Type: ActionUpload, LocalPath: local.Path, RemoteKey: key})
					} else {
						actions = append(actions, Action{Type: ActionDownload, LocalPath: local.Path, RemoteKey: key})
					}
				case "mirror":
					actions = append(actions, Action{Type: ActionUpload, LocalPath: local.Path, RemoteKey: key})
				}
			}
		} else {
			actions = append(actions, Action{Type: ActionUpload, LocalPath: local.Path, RemoteKey: key})
		}
		remoteKeys[key] = false
	}

	for key, exists := range remoteKeys {
		if !exists {
			continue
		}
		if _, hasLocal := localMap[key]; !hasLocal {
			actions = append(actions, Action{Type: ActionDownload, LocalPath: "", RemoteKey: key})
		}
	}

	return actions
}

func stateMap(states []store.SyncState) map[string]*store.SyncState {
	m := make(map[string]*store.SyncState)
	for i := range states {
		m[states[i].RemoteKey] = &states[i]
	}
	return m
}
```

- [ ] **Step 2: Implement executor**

```go
package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"simpler2sync/internal/r2client"
	"simpler2sync/internal/store"
)

type ExecCallback func(action Action, progress int, total int, err error)

func Execute(ctx context.Context, client *r2client.Client, st *store.Store, bucket, localRoot string, actions []Action, taskID string, cb ExecCallback) (int, int, error) {
	success, failed := 0, 0
	total := len(actions)
	for i, a := range actions {
		select {
		case <-ctx.Done():
			return success, failed, ctx.Err()
		default:
		}
		var actErr error
		switch a.Type {
		case ActionUpload:
			actErr = execUpload(ctx, client, st, bucket, a.LocalPath, a.RemoteKey, taskID)
		case ActionDownload:
			targetPath := filepath.Join(localRoot, a.RemoteKey)
			actErr = execDownload(ctx, client, st, bucket, a.RemoteKey, targetPath, taskID, localRoot)
		case ActionDeleteRemote:
			actErr = client.DeleteObject(ctx, bucket, a.RemoteKey)
			if actErr == nil {
				st.DeleteState(taskID, a.LocalPath, a.RemoteKey)
			}
		case ActionDeleteLocal:
			actErr = os.Remove(a.LocalPath)
			if actErr == nil {
				st.DeleteState(taskID, a.LocalPath, a.RemoteKey)
			}
		}
		if actErr != nil {
			failed++
			cb(a, i+1, total, actErr)
		} else {
			success++
			cb(a, i+1, total, nil)
		}
	}
	return success, failed, nil
}

func execUpload(ctx context.Context, client *r2client.Client, st *store.Store, bucket, localPath, remoteKey, taskID string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local: %w", err)
	}
	defer f.Close()
	fi, _ := f.Stat()
	etag, err := client.UploadFile(ctx, bucket, remoteKey, f, fi.Size())
	if err != nil {
		return err
	}
	hash, err := fileHash(localPath)
	if err != nil {
		return err
	}
	return st.UpsertState(&store.SyncState{
		TaskID:      taskID,
		LocalPath:   localPath,
		RemoteKey:   remoteKey,
		LocalHash:   hash,
		RemoteETag:  etag,
		LocalMtime:  fi.ModTime().Unix(),
		RemoteMtime: fi.ModTime().Unix(),
	})
}

func execDownload(ctx context.Context, client *r2client.Client, st *store.Store, bucket, remoteKey, localPath, taskID, localRoot string) error {
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	body, size, err := client.DownloadFile(ctx, bucket, remoteKey)
	if err != nil {
		return err
	}
	defer body.Close()
	f, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, body); err != nil {
		return fmt.Errorf("write local: %w", err)
	}
	hash, _ := fileHash(localPath)
	return st.UpsertState(&store.SyncState{
		TaskID:      taskID,
		LocalPath:   localPath,
		RemoteKey:   remoteKey,
		LocalHash:   hash,
		RemoteETag:  "",
		LocalMtime:  time.Now().Unix(),
		RemoteMtime: time.Now().Unix(),
	})
}
```

- [ ] **Step 3: Implement sync engine orchestrator**

```go
package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"simpler2sync/internal/r2client"
	"simpler2sync/internal/store"
)

type SyncResult struct {
	TaskID     string
	Success    int
	Failed     int
	ActionLog  []string
}

func RunSync(ctx context.Context, client *r2client.Client, st *store.Store, taskID, localRoot, bucket, remotePrefix, conflictStrategy string, cb ExecCallback) (*SyncResult, error) {
	if _, err := os.Stat(localRoot); os.IsNotExist(err) {
		return nil, fmt.Errorf("local path does not exist: %s", localRoot)
	}

	localFiles, err := WalkLocal(localRoot)
	if err != nil {
		return nil, fmt.Errorf("walk local: %w", err)
	}

	remoteObjects, err := ListRemote(ctx, client, bucket, remotePrefix)
	if err != nil {
		return nil, fmt.Errorf("list remote: %w", err)
	}

	states, err := st.ListStatesForTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("load states: %w", err)
	}

	actions := Diff(localFiles, remoteObjects, stateMap(states), taskID, localRoot, remotePrefix, conflictStrategy)

	for _, a := range actions {
		if a.Type == ActionDownload && a.LocalPath == "" {
			a.LocalPath = filepath.Join(localRoot, a.RemoteKey)
		}
	}

	success, failed, err := Execute(ctx, client, st, bucket, localRoot, actions, taskID, cb)
	if err != nil {
		return nil, err
	}

	return &SyncResult{
		TaskID:  taskID,
		Success: success,
		Failed:  failed,
	}, nil
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./...
```
Expected: No errors. Fix any missing imports.

---

### Task 7: Scheduler Module

**Files:**
- Create: `internal/scheduler/scheduler.go`

- [ ] **Step 1: Add cron dependency**

```bash
go get github.com/robfig/cron/v3
```

- [ ] **Step 2: Implement scheduler**

```go
package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
	running  bool
	onTick   func()
	cronSched *cron.Cron
}

func New(onTick func()) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		ctx:    ctx,
		cancel: cancel,
		onTick: onTick,
	}
}

func (s *Scheduler) StartInterval(seconds int) {
	s.Stop()
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()
	go func() {
		ticker := time.NewTicker(time.Duration(seconds) * time.Second)
		defer ticker.Stop()
		s.onTick()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.onTick()
			}
		}
	}()
}

func (s *Scheduler) StartCron(expr string) {
	s.Stop()
	if expr == "" {
		return
	}
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()
	s.cronSched = cron.New()
	s.cronSched.AddFunc(expr, s.onTick)
	s.cronSched.Start()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cronSched != nil {
		s.cronSched.Stop()
		s.cronSched = nil
	}
	s.cancel()
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.running = false
}

func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```
Expected: No errors.

---

### Task 8: GUI — Main Window Layout and System Tray

**Files:**
- Create: `internal/gui/app.go`
- Create: `internal/gui/icon.go`

- [ ] **Step 1: Add Fyne dependency**

```bash
go get fyne.io/fyne/v2
```

- [ ] **Step 2: Create embedded icon**

```go
package gui

import (
	_ "embed"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

//go:embed ../../assets/icon.png
var iconData []byte

func AppIcon() fyne.Resource {
	return fyne.NewStaticResource("icon.png", iconData)
}

func trayIcon() fyne.Resource {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(24, 24))
	return nil
}
```

- [ ] **Step 3: Create Fyne app with system tray**

```go
package gui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type App struct {
	fyneApp fyne.App
	window  fyne.Window
	onQuit  func()
}

func NewApp(name string, onQuit func()) *App {
	a := &App{
		fyneApp: app.NewWithID("simpler2sync"),
		onQuit:  onQuit,
	}
	a.fyneApp.SetIcon(AppIcon())
	return a
}

func (a *App) Run() {
	a.window = a.fyneApp.NewWindow("simpler2sync")
	a.window.SetMaster()
	a.window.Resize(fyne.NewSize(800, 600))

	if desk, ok := a.fyneApp.(desktop.App); ok {
		m := fyne.NewMenu("simpler2sync",
			fyne.NewMenuItem("Show", func() {
				a.window.Show()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				if a.onQuit != nil {
					a.onQuit()
				}
				a.fyneApp.Quit()
			}),
		)
		desk.SetSystemTrayMenu(m)
	}

	a.window.SetCloseIntercept(func() {
		a.window.Hide()
	})
	a.window.SetContent(widget.NewLabel("simpler2sync"))
	a.window.ShowAndRun()
}

func (a *App) Window() fyne.Window {
	return a.window
}

func (a *App) Quit() {
	a.fyneApp.Quit()
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./...
```
Expected: No errors.

---

### Task 9: GUI — Task List and Task Dialog

**Files:**
- Create: `internal/gui/tasklist.go`
- Create: `internal/gui/taskdialog.go`

- [ ] **Step 1: Implement task list view**

```go
package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"simpler2sync/internal/config"
)

type TaskListCallback func(action string, task *config.SyncTask)

type TaskList struct {
	cfg      *config.AppConfig
	list     *widget.List
	onAction TaskListCallback
}

func NewTaskList(cfg *config.AppConfig, onAction TaskListCallback) *TaskList {
	tl := &TaskList{
		cfg:      cfg,
		onAction: onAction,
	}
	tl.list = widget.NewList(
		func() int { return len(cfg.Tasks) },
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			t := cfg.Tasks[id]
			status := "off"
			if t.Enabled {
				status = "on"
			}
			obj.(*widget.Label).SetText(fmt.Sprintf("[%s] %s -> %s/%s", status, t.Name, t.RemoteBucket, t.RemotePrefix))
		},
	)
	return tl
}

func (tl *TaskList) Content() fyne.CanvasObject {
	addBtn := widget.NewButton("Add Task", func() {
		tl.onAction("add", nil)
	})
	editBtn := widget.NewButton("Edit", func() {
		idx := tl.list.Selected()
		if idx < 0 || idx >= len(tl.cfg.Tasks) {
			return
		}
		task := tl.cfg.Tasks[idx]
		tl.onAction("edit", &task)
	})
	removeBtn := widget.NewButton("Remove", func() {
		idx := tl.list.Selected()
		if idx < 0 || idx >= len(tl.cfg.Tasks) {
			return
		}
		tl.cfg.Tasks = append(tl.cfg.Tasks[:idx], tl.cfg.Tasks[idx+1:]...)
		tl.cfg.Save()
		tl.list.Refresh()
	})
	toggleBtn := widget.NewButton("Enable/Disable", func() {
		idx := tl.list.Selected()
		if idx < 0 || idx >= len(tl.cfg.Tasks) {
			return
		}
		tl.cfg.Tasks[idx].Enabled = !tl.cfg.Tasks[idx].Enabled
		tl.cfg.Save()
		tl.list.Refresh()
	})
	syncBtn := widget.NewButton("Sync Now", func() {
		idx := tl.list.Selected()
		if idx < 0 || idx >= len(tl.cfg.Tasks) {
			return
		}
		task := tl.cfg.Tasks[idx]
		tl.onAction("sync", &task)
	})

	btns := container.NewHBox(addBtn, editBtn, removeBtn, toggleBtn, syncBtn)
	return container.NewBorder(nil, btns, nil, nil, tl.list)
}

func (tl *TaskList) Refresh() {
	tl.list.Refresh()
}
```

- [ ] **Step 2: Implement task dialog**

```go
package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"simpler2sync/internal/config"
)

type TaskDialogCallback func(task config.SyncTask)

func ShowTaskDialog(win fyne.Window, task *config.SyncTask, onSave TaskDialogCallback) {
	name := widget.NewEntry()
	localPath := widget.NewEntry()
	bucket := widget.NewEntry()
	prefix := widget.NewEntry()
	enabled := widget.NewCheck("Enabled", nil)

	if task != nil {
		name.SetText(task.Name)
		localPath.SetText(task.LocalPath)
		bucket.SetText(task.RemoteBucket)
		prefix.SetText(task.RemotePrefix)
		enabled.Checked = task.Enabled
	}

	form := []*widget.FormItem{
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Local Path", localPath),
		widget.NewFormItem("R2 Bucket", bucket),
		widget.NewFormItem("R2 Prefix", prefix),
	}
	content := container.NewVBox(
		container.New(form...),
		enabled,
	)

	d := dialog.NewCustomConfirm("Sync Task", "Save", "Cancel", content, func(ok bool) {
		if !ok {
			return
		}
		nt := config.SyncTask{
			Name:         name.Text,
			LocalPath:    localPath.Text,
			RemoteBucket: bucket.Text,
			RemotePrefix: prefix.Text,
			Enabled:      enabled.Checked,
		}
		onSave(nt)
	}, win)
	d.Resize(fyne.NewSize(400, 300))
	d.Show()
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```
Expected: No errors.

---

### Task 10: GUI — Config Page, Log Page, Settings Page

**Files:**
- Create: `internal/gui/configpage.go`
- Create: `internal/gui/logpage.go`
- Create: `internal/gui/settings.go`

- [ ] **Step 1: Implement R2 config page**

```go
package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"simpler2sync/internal/config"
)

type ConfigPage struct {
	cfg *config.AppConfig
}

func NewConfigPage(cfg *config.AppConfig) *ConfigPage {
	return &ConfigPage{cfg: cfg}
}

func (cp *ConfigPage) Content(win fyne.Window) fyne.CanvasObject {
	endpoint := widget.NewEntry()
	endpoint.SetText(cp.cfg.R2.Endpoint)
	accessKey := widget.NewEntry()
	accessKey.SetText(cp.cfg.R2.AccessKeyID)
	secretKey := widget.NewPasswordEntry()
	secretKey.SetText(cp.cfg.R2.SecretAccessKey)
	region := widget.NewEntry()
	region.SetText(cp.cfg.R2.Region)

	saveBtn := widget.NewButton("Save", func() {
		cp.cfg.R2.Endpoint = endpoint.Text
		cp.cfg.R2.AccessKeyID = accessKey.Text
		cp.cfg.R2.SecretAccessKey = secretKey.Text
		cp.cfg.R2.Region = region.Text
		cp.cfg.R2.Type = "s3"
		cp.cfg.R2.Provider = "Cloudflare"
		cp.cfg.R2.ACL = "private"
		if err := cp.cfg.Save(); err != nil {
			dialog.ShowError(err, win)
		} else {
			dialog.ShowInformation("Success", "Configuration saved", win)
		}
	})

	loadBtn := widget.NewButton("Load from rclone config", func() {
		dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
			if err != nil || uc == nil {
				return
			}
			defer uc.Close()
			r2, err := config.LoadR2FromINIPath(uc.URI().Path())
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			endpoint.SetText(r2.Endpoint)
			accessKey.SetText(r2.AccessKeyID)
			secretKey.SetText(r2.SecretAccessKey)
			region.SetText(r2.Region)
		}, win)
	})

	form := container.New(
		&widget.Form{
			Items: []*widget.FormItem{
				widget.NewFormItem("Endpoint", endpoint),
				widget.NewFormItem("Access Key ID", accessKey),
				widget.NewFormItem("Secret Access Key", secretKey),
				widget.NewFormItem("Region", region),
			},
		},
	)

	btns := container.NewHBox(saveBtn, loadBtn)
	return container.NewBorder(nil, btns, nil, nil, form)
}
```

- [ ] **Step 2: Implement log page**

```go
package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type LogPage struct {
	log *widget.Entry
}

func NewLogPage() *LogPage {
	lp := &LogPage{
		log: widget.NewMultiLineEntry(),
	}
	lp.log.Disable()
	return lp
}

func (lp *LogPage) Content() fyne.CanvasObject {
	clearBtn := widget.NewButton("Clear", func() {
		lp.log.SetText("")
	})
	return container.NewBorder(nil, container.NewHBox(clearBtn), nil, nil, lp.log)
}

func (lp *LogPage) Append(text string) {
	lp.log.SetText(lp.log.Text + "\n" + text)
}
```

- [ ] **Step 3: Implement settings page**

```go
package gui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"simpler2sync/internal/config"
)

type SettingsPage struct {
	cfg *config.AppConfig
}

func NewSettingsPage(cfg *config.AppConfig) *SettingsPage {
	return &SettingsPage{cfg: cfg}
}

func (sp *SettingsPage) Content(win fyne.Window) fyne.CanvasObject {
	interval := widget.NewEntry()
	interval.SetText(strconv.Itoa(sp.cfg.Settings.IntervalSeconds))
	cronExpr := widget.NewEntry()
	cronExpr.SetText(sp.cfg.Settings.CronExpression)
	concurrent := widget.NewEntry()
	concurrent.SetText(strconv.Itoa(sp.cfg.Settings.ConcurrentTransfers))

	conflict := widget.NewSelect([]string{"newer", "mirror"}, func(v string) {
		sp.cfg.Settings.ConflictStrategy = v
	})
	if sp.cfg.Settings.ConflictStrategy != "" {
		conflict.SetSelected(sp.cfg.Settings.ConflictStrategy)
	} else {
		conflict.SetSelected("newer")
	}

	saveBtn := widget.NewButton("Save", func() {
		if v, err := strconv.Atoi(interval.Text); err == nil {
			sp.cfg.Settings.IntervalSeconds = v
		}
		sp.cfg.Settings.CronExpression = cronExpr.Text
		if v, err := strconv.Atoi(concurrent.Text); err == nil {
			sp.cfg.Settings.ConcurrentTransfers = v
		}
		if err := sp.cfg.Save(); err != nil {
			dialog.ShowError(err, win)
		} else {
			dialog.ShowInformation("Success", "Settings saved", win)
		}
	})

	form := container.New(
		&widget.Form{
			Items: []*widget.FormItem{
				widget.NewFormItem("Interval (seconds)", interval),
				widget.NewFormItem("Cron Expression", cronExpr),
				widget.NewFormItem("Conflict Strategy", conflict),
				widget.NewFormItem("Concurrent Transfers", concurrent),
			},
		},
	)
	return container.NewBorder(nil, saveBtn, nil, nil, form)
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./...
```
Expected: No errors.

---

### Task 11: Integration — Wire Everything in main.go

**Files:**
- Modify: `main.go`
- Modify: `internal/gui/app.go`

- [ ] **Step 1: Update main.go with full integration**

```go
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"simpler2sync/internal/config"
	"simpler2sync/internal/gui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log.Printf("Config loaded from: %s", config.ConfigPath())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	app := gui.NewApp("simpler2sync", func() {
		log.Println("shutting down...")
		cfg.Save()
	})

	go func() {
		<-sigCh
		app.Quit()
	}()

	app.BuildUI(cfg)
	app.Run()
	return nil
}
```

- [ ] **Step 2: Update gui/app.go BuildUI method**

```go
func (a *App) BuildUI(cfg *config.AppConfig) {
	tabs := container.NewAppTabs(
		container.NewTabItem("Sync Tasks", a.buildTaskTab(cfg)),
		container.NewTabItem("R2 Config", a.buildConfigTab(cfg)),
		container.NewTabItem("Log", a.buildLogTab()),
		container.NewTabItem("Settings", a.buildSettingsTab(cfg)),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	a.window.SetContent(tabs)
}

func (a *App) buildTaskTab(cfg *config.AppConfig) fyne.CanvasObject {
	// ... wire task list callbacks to sync engine
}
```

- [ ] **Step 3: Verify compilation and run**

```bash
go build ./...
go build -o simpler2sync.exe .
```
Expected: No errors.

---

### Task 12: Icons, Build Script, and Cross-Compile

**Files:**
- Create: `Makefile` (or `build.sh`)

- [ ] **Step 1: Generate app icons**

Create a simple 128x128 PNG icon at `assets/icon.png`. Use a programmatic approach or download a free icon.

- [ ] **Step 2: Create build script**

```bash
# build.sh / build.ps1
go build -ldflags="-s -w" -o bin/simpler2sync-windows-amd64.exe .
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/simpler2sync-darwin-amd64 .
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/simpler2sync-darwin-arm64 .
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/simpler2sync-linux-amd64 .
```

- [ ] **Step 3: Verify cross-compile on all platforms**

```bash
GOOS=windows GOARCH=amd64 go build ./...
GOOS=darwin GOARCH=amd64 go build ./...
GOOS=linux GOARCH=amd64 go build ./...
```
Expected: All pass.

---

### Task 13: README

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write README**

Write a README.md with:
- Project description
- Features
- Installation (download binary or build from source)
- Configuration guide
- Usage guide
- Build instructions
