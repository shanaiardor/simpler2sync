package sync

import (
	"path/filepath"
	"testing"
)

func TestLocalPathForRemoteKeyUsesNativeSeparators(t *testing.T) {
	root := filepath.Join("C:", "vault")

	got, err := localPathForRemoteKey(root, "obsidian/", "obsidian/history/.obsidian/app.json")
	if err != nil {
		t.Fatalf("localPathForRemoteKey returned error: %v", err)
	}

	want := filepath.Join(root, "history", ".obsidian", "app.json")
	if got != want {
		t.Fatalf("localPathForRemoteKey() = %q, want %q", got, want)
	}
}

func TestLocalPathForRemoteKeyNormalizesBackslashKeys(t *testing.T) {
	root := filepath.Join("C:", "vault")

	got, err := localPathForRemoteKey(root, `\obsidian\`, `obsidian\history\note.md`)
	if err != nil {
		t.Fatalf("localPathForRemoteKey returned error: %v", err)
	}

	want := filepath.Join(root, "history", "note.md")
	if got != want {
		t.Fatalf("localPathForRemoteKey() = %q, want %q", got, want)
	}
}

func TestLocalPathForRemoteKeyRejectsTraversal(t *testing.T) {
	root := filepath.Join("C:", "vault")

	if _, err := localPathForRemoteKey(root, "obsidian/", "obsidian/../outside.md"); err == nil {
		t.Fatal("expected traversal key to be rejected")
	}
}

func TestLocalPathForRemoteKeyRejectsOutsidePrefix(t *testing.T) {
	root := filepath.Join("C:", "vault")

	if _, err := localPathForRemoteKey(root, "obsidian/", "other/note.md"); err == nil {
		t.Fatal("expected outside-prefix key to be rejected")
	}
}
