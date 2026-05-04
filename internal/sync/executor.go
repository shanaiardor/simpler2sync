package sync

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"simpler2sync/internal/r2client"
	"simpler2sync/internal/store"
)

type ExecCallback func(action Action, progress int, total int, err error)

func Execute(ctx context.Context, client *r2client.Client, st *store.Store, bucket, localRoot string, actions []Action, taskID, conflictStrategy string, cb ExecCallback) (int, int, error) {
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
		case ActionReconcile:
			actErr = execReconcile(ctx, client, st, bucket, a, taskID, conflictStrategy)
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
	contentType, err := detectContentType(localPath, f)
	if err != nil {
		return err
	}
	etag, err := client.UploadFile(ctx, bucket, remoteKey, f, fi.Size(), contentType)
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
	tmpPath, remote, hash, err := downloadToTemp(ctx, client, bucket, remoteKey, localPath)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	oldHash, sameContent := "", false
	if _, err := os.Stat(localPath); err == nil {
		oldHash, err = fileHash(localPath)
		if err != nil {
			return err
		}
		sameContent = oldHash == hash
	}
	if !sameContent {
		if err := replaceFile(tmpPath, localPath); err != nil {
			return err
		}
	}
	fi, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("stat local: %w", err)
	}
	return st.UpsertState(&store.SyncState{
		TaskID:      taskID,
		LocalPath:   localPath,
		RemoteKey:   remoteKey,
		LocalHash:   hash,
		RemoteETag:  remote.ETag,
		LocalMtime:  fi.ModTime().Unix(),
		RemoteMtime: remote.LastModified,
	})
}

func execReconcile(ctx context.Context, client *r2client.Client, st *store.Store, bucket string, a Action, taskID, conflictStrategy string) error {
	tmpPath, remote, remoteHash, err := downloadToTemp(ctx, client, bucket, a.RemoteKey, a.LocalPath)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	localHash := a.Local.Hash
	if localHash == "" {
		localHash, err = fileHash(a.LocalPath)
		if err != nil {
			return err
		}
	}
	if localHash == remoteHash {
		return st.UpsertState(&store.SyncState{
			TaskID:      taskID,
			LocalPath:   a.LocalPath,
			RemoteKey:   a.RemoteKey,
			LocalHash:   localHash,
			RemoteETag:  remote.ETag,
			LocalMtime:  a.Local.ModTime,
			RemoteMtime: remote.LastModified,
		})
	}

	if conflictStrategy == "mirror" || a.Local.ModTime >= remote.LastModified {
		return execUpload(ctx, client, st, bucket, a.LocalPath, a.RemoteKey, taskID)
	}
	if err := replaceFile(tmpPath, a.LocalPath); err != nil {
		return err
	}
	fi, err := os.Stat(a.LocalPath)
	if err != nil {
		return fmt.Errorf("stat local: %w", err)
	}
	return st.UpsertState(&store.SyncState{
		TaskID:      taskID,
		LocalPath:   a.LocalPath,
		RemoteKey:   a.RemoteKey,
		LocalHash:   remoteHash,
		RemoteETag:  remote.ETag,
		LocalMtime:  fi.ModTime().Unix(),
		RemoteMtime: remote.LastModified,
	})
}

func downloadToTemp(ctx context.Context, client *r2client.Client, bucket, remoteKey, localPath string) (string, r2client.ObjectInfo, string, error) {
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", r2client.ObjectInfo{}, "", fmt.Errorf("mkdir: %w", err)
	}
	body, remote, err := client.DownloadFile(ctx, bucket, remoteKey)
	if err != nil {
		return "", r2client.ObjectInfo{}, "", err
	}
	defer body.Close()
	tmp, err := os.CreateTemp(dir, ".simpler2sync-*")
	if err != nil {
		return "", r2client.ObjectInfo{}, "", fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := io.Copy(tmp, body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", r2client.ObjectInfo{}, "", fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", r2client.ObjectInfo{}, "", fmt.Errorf("close temp: %w", err)
	}
	hash, err := fileHash(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return "", r2client.ObjectInfo{}, "", err
	}
	if remote.LastModified == 0 {
		remote.LastModified = time.Now().Unix()
	}
	return tmpPath, remote, hash, nil
}

func replaceFile(tmpPath, localPath string) error {
	if err := os.Rename(tmpPath, localPath); err == nil {
		return nil
	}
	src, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("open temp: %w", err)
	}
	defer src.Close()
	dst, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return fmt.Errorf("write local: %w", err)
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("close local: %w", err)
	}
	return nil
}

func detectContentType(localPath string, f *os.File) (string, error) {
	if contentType := contentTypeByExtension(localPath); contentType != "" {
		return contentType, nil
	}

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read content type sample: %w", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("reset file after content type sample: %w", err)
	}
	if n == 0 {
		return "application/octet-stream", nil
	}
	return http.DetectContentType(buf[:n]), nil
}

func contentTypeByExtension(localPath string) string {
	ext := strings.ToLower(filepath.Ext(localPath))
	switch ext {
	case ".md", ".markdown":
		return "text/markdown; charset=utf-8"
	case ".mjs", ".js":
		return "text/javascript; charset=utf-8"
	case ".ts":
		return "text/typescript; charset=utf-8"
	case ".tsx":
		return "text/tsx; charset=utf-8"
	case ".jsx":
		return "text/jsx; charset=utf-8"
	case ".yml", ".yaml":
		return "application/yaml"
	case ".toml":
		return "application/toml"
	}
	return mime.TypeByExtension(ext)
}
