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
