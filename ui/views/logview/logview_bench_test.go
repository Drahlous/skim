package logview

import (
	"fmt"
	"skim/filterfiles"
	"testing"
)

// genBenchLines returns n synthetic, varied log lines for benchmarking:
// varied enough (four rotating levels, an id in every line) to exercise
// several filters rather than trivially short-circuiting on the first one.
func genBenchLines(n int) []string {
	levels := []string{"debug", "info", "warn", "error"}
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("%s: line %d handling request id %d", levels[i%len(levels)], i, i*7)
	}
	return lines
}

// genBenchFilters returns a handful of enabled filters representative of a
// real .tat file: one per log level plus one that matches broadly, so a
// benchmark run touches a realistic mix of match/no-match lines.
func genBenchFilters(b *testing.B) []filterfiles.Filter {
	b.Helper()
	specs := []struct{ text, backColor string }{
		{"^debug", "#87CEFA"},
		{"^info", "#90EE90"},
		{"^warn", "#FFD700"},
		{"^error", "#FF6347"},
		{"request id", "#DDA0DD"},
	}
	filters := make([]filterfiles.Filter, len(specs))
	for i, s := range specs {
		re, err := filterfiles.CompileRegex(s.text, false)
		if err != nil {
			b.Fatalf("CompileRegex(%q) failed: %v", s.text, err)
		}
		filters[i] = filterfiles.Filter{
			XML:       filterfiles.FilterXML{Text: s.text},
			Regex:     re,
			IsEnabled: true,
			BackColor: s.backColor,
		}
	}
	return filters
}

// benchmarkMakeTable measures repeated MakeTable calls against the same
// LogView with an unchanged filter set -- the common case of cursor
// movement or a window resize between keystrokes, where nothing about the
// filters has changed since the last render.
func benchmarkMakeTable(b *testing.B, n int) {
	lines := genBenchLines(n)
	filters := genBenchFilters(b)
	v := LogView{Lines: lines}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.MakeTable(200, 60, filters, true, 1)
	}
}

func BenchmarkMakeTable_Medium(b *testing.B) { benchmarkMakeTable(b, 10_000) }
func BenchmarkMakeTable_Large(b *testing.B)  { benchmarkMakeTable(b, 100_000) }
func BenchmarkMakeTable_Huge(b *testing.B)   { benchmarkMakeTable(b, 1_000_000) }

// benchmarkMakeTableColdCache measures a single MakeTable call against a
// fresh LogView every iteration, i.e. the worst case where the filter set
// (or the log itself) has just changed and nothing can be reused from a
// previous render.
func benchmarkMakeTableColdCache(b *testing.B, n int) {
	lines := genBenchLines(n)
	filters := genBenchFilters(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		v := LogView{Lines: lines}
		b.StartTimer()
		v.MakeTable(200, 60, filters, true, 1)
	}
}

func BenchmarkMakeTableColdCache_Medium(b *testing.B) { benchmarkMakeTableColdCache(b, 10_000) }
func BenchmarkMakeTableColdCache_Large(b *testing.B)  { benchmarkMakeTableColdCache(b, 100_000) }
func BenchmarkMakeTableColdCache_Huge(b *testing.B)   { benchmarkMakeTableColdCache(b, 1_000_000) }
