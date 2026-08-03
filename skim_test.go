package main

import (
	"flag"
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

func TestIsLogFlagSetWhenProvided(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("log", "default", "")
	if err := fs.Parse([]string{"-log", "somefile.log"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if !isLogFlagSet(fs) {
		t.Error("isLogFlagSet = false when -log was explicitly provided, want true")
	}
}

func TestIsLogFlagSetWhenOmitted(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("log", "default", "")
	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if isLogFlagSet(fs) {
		t.Error("isLogFlagSet = true when -log was omitted, want false")
	}
}

func TestStdinIsPipedForPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	if !stdinIsPiped(r) {
		t.Error("stdinIsPiped = false for a pipe, want true")
	}
}

func TestStdinIsPipedForRegularFile(t *testing.T) {
	path := t.TempDir() + "/test.log"
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open test fixture: %v", err)
	}
	defer f.Close()

	if !stdinIsPiped(f) {
		t.Error("stdinIsPiped = false for a redirected regular file, want true")
	}
}
