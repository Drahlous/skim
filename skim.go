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

	ui.RunUI(filters, scanner, filter_file, filterSettings, usingStdinLog)
}

func main() {

	// Parse Command Line Options
	filter_file := flag.String("filter", "./examples/simple_filter_two.tat", "supply the path to a TAT filter file")
	log_file := flag.String("log", "./examples/simple_longer.log", "supply the path to the input log file, or - to read from stdin")
	flag.Parse()

	// Run the program
	run(*filter_file, *log_file)
}
