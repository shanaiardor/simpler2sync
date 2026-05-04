package sync

import (
	"path/filepath"
	"testing"

	"simpler2sync/internal/r2client"
	"simpler2sync/internal/store"
)

func TestDiffReconcilesExistingFilesWithoutState(t *testing.T) {
	root := filepath.Join("C:", "vault")
	localPath := filepath.Join(root, "note.md")
	local := FileInfo{Path: localPath, Size: 5, ModTime: 100, Hash: "local-sha"}
	remote := r2client.ObjectInfo{Key: "obsidian/note.md", ETag: "remote-etag", Size: 5, LastModified: 90}

	actions, err := Diff(
		[]FileInfo{local},
		map[string]r2client.ObjectInfo{remote.Key: remote},
		map[string]*store.SyncState{},
		"task",
		root,
		"obsidian/",
		"newer",
	)
	if err != nil {
		t.Fatalf("Diff returned error: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("len(actions) = %d, want 1", len(actions))
	}
	if actions[0].Type != ActionReconcile {
		t.Fatalf("action type = %q, want %q", actions[0].Type, ActionReconcile)
	}
	if actions[0].Local.Hash != local.Hash {
		t.Fatalf("action local hash = %q, want %q", actions[0].Local.Hash, local.Hash)
	}
	if actions[0].Remote.ETag != remote.ETag {
		t.Fatalf("action remote etag = %q, want %q", actions[0].Remote.ETag, remote.ETag)
	}
}

func TestDiffDownloadsRemoteETagChangeWithRemoteMetadata(t *testing.T) {
	root := filepath.Join("C:", "vault")
	localPath := filepath.Join(root, "note.md")
	local := FileInfo{Path: localPath, Size: 5, ModTime: 100, Hash: "same-sha"}
	remote := r2client.ObjectInfo{Key: "obsidian/note.md", ETag: "new-etag", Size: 5, LastModified: 110}
	st := &store.SyncState{
		TaskID:     "task",
		LocalPath:  localPath,
		RemoteKey:  remote.Key,
		LocalHash:  "same-sha",
		RemoteETag: "old-etag",
	}

	actions, err := Diff(
		[]FileInfo{local},
		map[string]r2client.ObjectInfo{remote.Key: remote},
		map[string]*store.SyncState{remote.Key: st},
		"task",
		root,
		"obsidian/",
		"newer",
	)
	if err != nil {
		t.Fatalf("Diff returned error: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("len(actions) = %d, want 1", len(actions))
	}
	if actions[0].Type != ActionDownload {
		t.Fatalf("action type = %q, want %q", actions[0].Type, ActionDownload)
	}
	if actions[0].Remote.ETag != remote.ETag {
		t.Fatalf("action remote etag = %q, want %q", actions[0].Remote.ETag, remote.ETag)
	}
}
