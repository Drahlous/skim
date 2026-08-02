// Command gentestlogs deterministically generates synthetic .log/.tat fixture
// files at various sizes, for manual testing and benchmarking against
// larger-than-example-sized logs without checking megabytes of generated
// text into git. Output goes under testdata/ by default, which is
// gitignored -- run this whenever you need the files locally.
//
// Usage:
//
//	go run ./tools/gentestlogs
//	go run ./tools/gentestlogs -sizes tiny=100,medium=10000
//	go run ./tools/gentestlogs -out /tmp/logs
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// filterFileXML is a fixed .tat filter file matching the log lines writeLog
// produces (one filter per rotating level), so the generated files are
// usable together out of the box: `go run . -filter testdata/logs/filters.tat -log testdata/logs/huge.log`.
const filterFileXML = `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<TextAnalysisTool.NET version="2023-04-25" showOnlyFilteredLines="False">
  <filters>
    <filter enabled="y" excluding="n" description="" backColor="87cefa" type="matches_text" case_sensitive="n" regex="y" text="^debug" />
    <filter enabled="y" excluding="n" description="" backColor="90ee90" type="matches_text" case_sensitive="n" regex="y" text="^info" />
    <filter enabled="y" excluding="n" description="" backColor="ffd700" type="matches_text" case_sensitive="n" regex="y" text="^warn" />
    <filter enabled="y" excluding="n" description="" backColor="ff6347" type="matches_text" case_sensitive="n" regex="y" text="^error" />
  </filters>
</TextAnalysisTool.NET>
`

var levels = []string{"debug", "info", "warn", "error"}

// writeLog deterministically writes n synthetic log lines to path: level
// rotates through levels and every line carries an id, so the output is
// varied enough to exercise multiple filters realistically, and identical
// byte-for-byte across runs for a given n.
func writeLog(path string, n int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for i := 0; i < n; i++ {
		level := levels[i%len(levels)]
		if _, err := fmt.Fprintf(w, "%s: line %d handling request id %d something happened\n", level, i+1, i*7); err != nil {
			return err
		}
	}
	return w.Flush()
}

// parseSizes parses a "name=count,name=count" spec into line counts, and
// the order the names were given in (so output is generated in a
// predictable order regardless of map iteration).
func parseSizes(spec string) (counts map[string]int, order []string, err error) {
	counts = map[string]int{}
	for _, pair := range strings.Split(spec, ",") {
		name, countStr, ok := strings.Cut(pair, "=")
		if !ok {
			return nil, nil, fmt.Errorf("invalid size spec %q, want name=count", pair)
		}
		n, err := strconv.Atoi(countStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid line count in %q: %w", pair, err)
		}
		counts[name] = n
		order = append(order, name)
	}
	return counts, order, nil
}

func main() {
	outDir := flag.String("out", "testdata/logs", "directory to write the generated .log/.tat files to")
	sizeSpec := flag.String("sizes", "medium=10000,large=100000,huge=1000000", "comma-separated name=line-count pairs to generate")
	flag.Parse()

	counts, order, err := parseSizes(*sizeSpec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	filterPath := filepath.Join(*outDir, "filters.tat")
	if err := os.WriteFile(filterPath, []byte(filterFileXML), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("wrote", filterPath)

	for _, name := range order {
		path := filepath.Join(*outDir, name+".log")
		if err := writeLog(path, counts[name]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s (%d lines)\n", path, counts[name])
	}
}
