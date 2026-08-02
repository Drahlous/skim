package main

import (
	"io"
	"os"
	"testing"
)

func TestOpenLogSourceStdin(t *testing.T) {
	r, usingStdin, err := openLogSource(stdinPath)
	if err != nil {
		t.Fatalf("openLogSource(%q) returned unexpected error: %v", stdinPath, err)
	}
	defer r.Close()

	if !usingStdin {
		t.Error("usingStdin = false for the stdin path, want true")
	}
}

func TestOpenLogSourceFile(t *testing.T) {
	path := t.TempDir() + "/test.log"
	if err := os.WriteFile(path, []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}

	r, usingStdin, err := openLogSource(path)
	if err != nil {
		t.Fatalf("openLogSource(%q) returned unexpected error: %v", path, err)
	}
	defer r.Close()

	if usingStdin {
		t.Error("usingStdin = true for a regular file path, want false")
	}

	content, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read from opened log source: %v", err)
	}
	if string(content) != "hello\nworld\n" {
		t.Errorf("read content = %q, want %q", content, "hello\nworld\n")
	}
}

func TestOpenLogSourceMissingFile(t *testing.T) {
	if _, _, err := openLogSource("/nonexistent/path/to.log"); err == nil {
		t.Fatal("openLogSource with a missing file returned no error")
	}
}
