package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"skim/filterfiles"
	"skim/ui"
)

// stdinPath is the -log value that means "read the log from stdin"
// (kubectl logs -f pod | skim -log -), matching the common Unix convention
// for "-" as a stand-in for stdin/stdout.
const stdinPath = "-"

// openLogSource opens the log input for logFile: stdin if it's stdinPath,
// otherwise the named file. The returned bool reports whether stdin was
// used, which callers need to know since a fully-consumed stdin can no
// longer serve as the TUI's keyboard input source.
func openLogSource(logFile string) (io.ReadCloser, bool, error) {
	if logFile == stdinPath {
		return io.NopCloser(os.Stdin), true, nil
	}
	f, err := os.Open(logFile)
	return f, false, err
}

// isLogFlagSet reports whether -log was explicitly passed on the command
// line (as opposed to left at its default), by checking fs's set-flags list.
func isLogFlagSet(fs *flag.FlagSet) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "log" {
			set = true
		}
	})
	return set
}

// stdinIsPiped reports whether r is fed by a pipe or redirect (e.g.
// `cmd | skim` or `skim < file`) rather than left as an interactive
// terminal, in which case there's nothing useful to read from it.
func stdinIsPiped(r *os.File) bool {
	info, err := r.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}

// resolveLogFile returns the -log value run() should actually use: logFile
// as given, unless -log wasn't explicitly passed on the command line and
// stdin is piped/redirected rather than an interactive terminal, in which
// case it returns stdinPath so `cmd | skim` works without extra flags.
func resolveLogFile(logFile string, fs *flag.FlagSet, stdin *os.File) string {
	if !isLogFlagSet(fs) && stdinIsPiped(stdin) {
		return stdinPath
	}
	return logFile
}

// runUI is a seam for testing: run()'s success path calls this rather than
// ui.RunUI directly, so tests can substitute a no-op and exercise run()'s
// full happy path without handing a real terminal to Bubble Tea.
var runUI = ui.RunUI

// run loads the filter file and log, then launches the TUI, returning the
// process exit code the caller (main, via runFn) should use. A filter whose
// regex fails to compile is a recoverable problem -- it's disabled and
// logged as a warning, and run still launches the TUI, still returning 0 --
// but an unreadable filter or log file is not: run logs it once and returns
// 1 without ever calling runUI, so scripts/CI checking skim's exit code can
// actually tell startup failed (see the issue this fixed: previously every
// failure here printed and returned with exit code 0).
func run(filter_file string, log_file string) int {

	// Read filter settings from the XML file
	filterSettings, err := filterfiles.ReadFilterFile(filter_file)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	// Compile the extracted filters into regular expressions. A filter with
	// an invalid regex is disabled (not fatal); its warning is logged once
	// here and passed through to the UI so it can be surfaced there too.
	filters, warnings := filterfiles.CompileFilterRegularExpressions(filterSettings)
	for _, w := range warnings {
		fmt.Println(w)
	}

	// Read the log line-by-line, from stdin (-log -) or from a file
	logfile, usingStdinLog, err := openLogSource(log_file)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	defer logfile.Close()
	scanner := bufio.NewScanner(logfile)

	runUI(filters, scanner, filter_file, filterSettings, usingStdinLog, warnings)
	return 0
}

// runFn is a seam for testing: mainWithExitCode() calls this rather than
// run() directly, so a test can substitute a no-op and exercise flag
// parsing without actually reading a log file or starting the UI.
var runFn = run

// mainWithExitCode does the real work of main, returning the process exit
// code instead of calling os.Exit itself, so tests can call it directly
// without killing the test process.
func mainWithExitCode() int {

	// Parse Command Line Options
	filter_file := flag.String("filter", "./examples/simple_filter_two.tat", "supply the path to a TAT filter file")
	log_file := flag.String("log", "./examples/simple_longer.log", "supply the path to the input log file, or - to read from stdin")
	flag.Parse()

	// Run the program
	return runFn(*filter_file, resolveLogFile(*log_file, flag.CommandLine, os.Stdin))
}

func main() {
	os.Exit(mainWithExitCode())
}
