package sync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

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
			actErr = execDownload(ctx, client, st, bucket, a.RemoteKey, a.LocalPath, taskID)
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
	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat local: %w", err)
	}
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
		RemoteMtime: time.Now().Unix(),
	})
}

func execDownload(ctx context.Context, client *r2client.Client, st *store.Store, bucket, remoteKey, localPath, taskID string) error {
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	body, _, err := client.DownloadFile(ctx, bucket, remoteKey)
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
	hash, err := fileHash(localPath)
	if err != nil {
		return err
	}
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
