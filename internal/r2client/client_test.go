package r2client

import (
	"path/filepath"
	"testing"
)

func TestRemoteKeyUsesSlashSeparators(t *testing.T) {
	root := filepath.Join("C:", "vault")
	file := filepath.Join(root, "history", ".obsidian", "app.json")

	key, err := RemoteKey(root, file, "obsidian/")
	if err != nil {
		t.Fatalf("RemoteKey returned error: %v", err)
	}

	want := "obsidian/history/.obsidian/app.json"
	if key != want {
		t.Fatalf("RemoteKey() = %q, want %q", key, want)
	}
}

func TestRemoteKeyTrimsPrefixSlashes(t *testing.T) {
	root := filepath.Join("C:", "vault")
	file := filepath.Join(root, "note.md")

	key, err := RemoteKey(root, file, `\obsidian\`)
	if err != nil {
		t.Fatalf("RemoteKey returned error: %v", err)
	}

	want := "obsidian/note.md"
	if key != want {
		t.Fatalf("RemoteKey() = %q, want %q", key, want)
	}
}

func TestNormalizePrefix(t *testing.T) {
	tests := map[string]string{
		"":             "",
		"obsidian":     "obsidian/",
		"obsidian/":    "obsidian/",
		`\obsidian\`:   "obsidian/",
		"/vault/notes": "vault/notes/",
	}

	for input, want := range tests {
		if got := NormalizePrefix(input); got != want {
			t.Fatalf("NormalizePrefix(%q) = %q, want %q", input, got, want)
		}
	}
}
