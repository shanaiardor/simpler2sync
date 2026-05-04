package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectContentTypeUsesMarkdownExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.md")
	if err := os.WriteFile(path, []byte("# Hello\n"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	got, err := detectContentType(path, f)
	if err != nil {
		t.Fatalf("detectContentType returned error: %v", err)
	}

	want := "text/markdown; charset=utf-8"
	if got != want {
		t.Fatalf("detectContentType() = %q, want %q", got, want)
	}
}

func TestDetectContentTypeSniffsUnknownExtensionAndResetsReader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unknown")
	data := []byte("<!doctype html><html><body>Hello</body></html>")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	got, err := detectContentType(path, f)
	if err != nil {
		t.Fatalf("detectContentType returned error: %v", err)
	}
	if got != "text/html; charset=utf-8" {
		t.Fatalf("detectContentType() = %q, want text/html; charset=utf-8", got)
	}

	pos, err := f.Seek(0, 1)
	if err != nil {
		t.Fatalf("seek current: %v", err)
	}
	if pos != 0 {
		t.Fatalf("file position = %d, want 0", pos)
	}
}
