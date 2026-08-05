package main

import (
	"bufio"
	"flag"
	"io"
	"os"
	"path/filepath"
	"skim/filterfiles"
	"strings"
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

// captureStdout redirects os.Stdout for the duration of fn and returns
// whatever was written to it, so run()'s fmt.Println error paths (which
// don't return an error the caller can inspect) can still be asserted on.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	return string(out)
}

func TestRunPrintsErrorAndReturnsNonZeroOnUnreadableFilterFile(t *testing.T) {
	var code int
	out := captureStdout(t, func() {
		code = run("/nonexistent/path/to/filters.tat", "./examples/simple_longer.log")
	})
	if out == "" {
		t.Error("run() with a missing filter file printed nothing, want an error message")
	}
	if code != 1 {
		t.Errorf("run() with a missing filter file returned exit code %d, want 1", code)
	}
}

// TestRunDisablesInvalidFilterRegexAndStillLaunchesUI covers the issue this
// fixed: a single filter with an invalid regex anywhere in the file used to
// abort the whole load (three redundant error logs, no TUI, exit code 0).
// It should instead be disabled, logged once as a warning, passed through
// to the UI, and not prevent the other (valid) filters or the TUI itself
// from loading -- with exit code 0, since this is a recoverable problem.
func TestRunDisablesInvalidFilterRegexAndStillLaunchesUI(t *testing.T) {
	origRunUI := runUI
	defer func() { runUI = origRunUI }()

	path := filepath.Join(t.TempDir(), "bad.tat")
	xml := `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<TextAnalysisTool.NET version="2023-04-25" showOnlyFilteredLines="False">
  <filters>
    <filter enabled="y" excluding="n" description="good" backColor="87cefa" type="matches_text" case_sensitive="n" regex="y" text="^debug" />
    <filter enabled="y" excluding="n" description="bad" backColor="ff0000" type="matches_text" case_sensitive="n" regex="y" text="(" />
  </filters>
</TextAnalysisTool.NET>`
	if err := os.WriteFile(path, []byte(xml), 0o644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}

	var called bool
	var gotFilters []filterfiles.Filter
	var gotWarnings []error
	runUI = func(filters []filterfiles.Filter, scanner *bufio.Scanner, filterFilePath string, fileMeta filterfiles.TextAnalysisToolSettings, usingStdinLog bool, warnings []error) {
		called = true
		gotFilters = filters
		gotWarnings = warnings
	}

	var code int
	out := captureStdout(t, func() {
		code = run(path, "./examples/simple_longer.log")
	})

	if code != 0 {
		t.Errorf("run() with one invalid filter regex returned exit code %d, want 0 (recoverable)", code)
	}
	if !called {
		t.Fatal("run() with one invalid filter regex among otherwise-valid ones did not call runUI, want the TUI to still launch")
	}
	if len(gotFilters) != 2 {
		t.Fatalf("got %d filters passed to runUI, want 2 (both filters, one disabled)", len(gotFilters))
	}
	if !gotFilters[0].IsEnabled {
		t.Error("gotFilters[0].IsEnabled = false, want the valid filter to stay enabled")
	}
	if gotFilters[1].IsEnabled {
		t.Error("gotFilters[1].IsEnabled = true, want the invalid-regex filter forced disabled")
	}
	if len(gotWarnings) != 1 {
		t.Fatalf("got %d warnings, want 1 for the single invalid filter", len(gotWarnings))
	}
	if !strings.Contains(gotWarnings[0].Error(), "bad") {
		t.Errorf("warning %q does not identify the offending filter by its description", gotWarnings[0].Error())
	}

	if strings.Count(out, "invalid regex") != 1 {
		t.Errorf("run() logged the invalid-regex warning %d times to stdout, want exactly once (was 3x before this fix)", strings.Count(out, "invalid regex"))
	}
}

func TestRunPrintsErrorAndReturnsNonZeroOnUnreadableLogFile(t *testing.T) {
	var code int
	out := captureStdout(t, func() {
		code = run("./examples/simple_filter_two.tat", "/nonexistent/path/to.log")
	})
	if out == "" {
		t.Error("run() with a missing log file printed nothing, want an error message")
	}
	if code != 1 {
		t.Errorf("run() with a missing log file returned exit code %d, want 1", code)
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

func TestStdinIsPipedReturnsFalseWhenStatFails(t *testing.T) {
	path := t.TempDir() + "/test.log"
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open test fixture: %v", err)
	}
	f.Close() // Stat on a closed file returns an error.

	if stdinIsPiped(f) {
		t.Error("stdinIsPiped = true for a closed file whose Stat() errors, want false")
	}
}

func TestResolveLogFile(t *testing.T) {
	newFlagSet := func(args ...string) *flag.FlagSet {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.String("log", "default.log", "")
		if err := fs.Parse(args); err != nil {
			t.Fatalf("failed to parse flags: %v", err)
		}
		return fs
	}
	pipe := func(t *testing.T) *os.File {
		t.Helper()
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		t.Cleanup(func() { r.Close(); w.Close() })
		return r
	}
	// charDevice opens /dev/null, standing in for an interactive terminal:
	// stdinIsPiped only cares whether os.ModeCharDevice is set on the file,
	// which is true for /dev/null just as it is for a real tty, without
	// depending on the test having a controlling terminal to open.
	charDevice := func(t *testing.T) *os.File {
		t.Helper()
		f, err := os.Open(os.DevNull)
		if err != nil {
			t.Fatalf("failed to open %s: %v", os.DevNull, err)
		}
		t.Cleanup(func() { f.Close() })
		return f
	}

	t.Run("log flag explicitly set wins even if stdin is piped", func(t *testing.T) {
		fs := newFlagSet("-log", "explicit.log")
		got := resolveLogFile("explicit.log", fs, pipe(t))
		if got != "explicit.log" {
			t.Errorf("resolveLogFile() = %q, want %q", got, "explicit.log")
		}
	})

	t.Run("log flag omitted and stdin piped reads from stdin", func(t *testing.T) {
		fs := newFlagSet()
		got := resolveLogFile("default.log", fs, pipe(t))
		if got != stdinPath {
			t.Errorf("resolveLogFile() = %q, want %q", got, stdinPath)
		}
	})

	t.Run("log flag omitted and stdin is a terminal keeps the default", func(t *testing.T) {
		fs := newFlagSet()
		got := resolveLogFile("default.log", fs, charDevice(t))
		if got != "default.log" {
			t.Errorf("resolveLogFile() = %q, want %q", got, "default.log")
		}
	})
}

func TestRunSuccessPathCallsRunUI(t *testing.T) {
	origRunUI := runUI
	defer func() { runUI = origRunUI }()

	var called bool
	runUI = func(filters []filterfiles.Filter, scanner *bufio.Scanner, filterFilePath string, fileMeta filterfiles.TextAnalysisToolSettings, usingStdinLog bool, warnings []error) {
		called = true
		if len(filters) != 3 {
			t.Errorf("got %d filters, want 3 (from examples/simple_filter_two.tat)", len(filters))
		}
		if usingStdinLog {
			t.Error("usingStdinLog = true for a regular log file, want false")
		}
		if len(warnings) != 0 {
			t.Errorf("got %d warnings, want 0 for a filter file with no invalid regexes", len(warnings))
		}
	}

	if code := run("./examples/simple_filter_two.tat", "./examples/simple_longer.log"); code != 0 {
		t.Errorf("run() with valid filter and log files returned exit code %d, want 0", code)
	}

	if !called {
		t.Error("run() with valid filter and log files did not call runUI")
	}
}

func TestMainParsesFlagsAndCallsRunFn(t *testing.T) {
	origArgs := os.Args
	origRunFn := runFn
	origCommandLine := flag.CommandLine
	defer func() {
		os.Args = origArgs
		runFn = origRunFn
		flag.CommandLine = origCommandLine
	}()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"skim", "-filter", "myfilters.tat", "-log", "mylog.log"}

	var gotFilter, gotLog string
	runFn = func(filterFile, logFile string) int {
		gotFilter = filterFile
		gotLog = logFile
		return 0
	}

	// mainWithExitCode, not main(), so this test doesn't call os.Exit and
	// kill the test process.
	if code := mainWithExitCode(); code != 0 {
		t.Errorf("mainWithExitCode() = %d, want 0", code)
	}

	if gotFilter != "myfilters.tat" {
		t.Errorf("main() called runFn with filter_file = %q, want %q", gotFilter, "myfilters.tat")
	}
	if gotLog != "mylog.log" {
		t.Errorf("main() called runFn with log_file = %q, want %q", gotLog, "mylog.log")
	}
}

func TestMainWithExitCodePropagatesRunFnExitCode(t *testing.T) {
	origArgs := os.Args
	origRunFn := runFn
	origCommandLine := flag.CommandLine
	defer func() {
		os.Args = origArgs
		runFn = origRunFn
		flag.CommandLine = origCommandLine
	}()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"skim", "-filter", "myfilters.tat", "-log", "mylog.log"}

	runFn = func(filterFile, logFile string) int { return 1 }

	if code := mainWithExitCode(); code != 1 {
		t.Errorf("mainWithExitCode() = %d, want 1 when runFn fails", code)
	}
}
