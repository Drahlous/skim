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

func run(filter_file string, log_file string) {

	// Read filter settings from the XML file
	filterSettings, err := filterfiles.ReadFilterFile(filter_file)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Compile the extracted filters into regular expressions
	filters, err := filterfiles.CompileFilterRegularExpressions(filterSettings)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Read the log line-by-line, from stdin (-log -) or from a file
	logfile, usingStdinLog, err := openLogSource(log_file)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer logfile.Close()
	scanner := bufio.NewScanner(logfile)

	runUI(filters, scanner, filter_file, filterSettings, usingStdinLog)
}

// runFn is a seam for testing: main() calls this rather than run() directly,
// so a test can substitute a no-op and exercise main()'s flag parsing without
// actually reading a log file or starting the UI.
var runFn = run

func main() {

	// Parse Command Line Options
	filter_file := flag.String("filter", "./examples/simple_filter_two.tat", "supply the path to a TAT filter file")
	log_file := flag.String("log", "./examples/simple_longer.log", "supply the path to the input log file, or - to read from stdin")
	flag.Parse()

	// Run the program
	runFn(*filter_file, resolveLogFile(*log_file, flag.CommandLine, os.Stdin))
}
