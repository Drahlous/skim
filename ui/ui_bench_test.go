package ui

import (
	"bufio"
	"fmt"
	"path/filepath"
	"skim/filterfiles"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// genBenchLog returns n synthetic, varied log lines (newline-separated) for
// benchmarking: varied enough (four rotating levels, an id in every line)
// to exercise several filters rather than trivially short-circuiting on the
// first one.
func genBenchLog(n int) string {
	levels := []string{"debug", "info", "warn", "error"}
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "%s: line %d handling request id %d\n", levels[i%len(levels)], i, i*7)
	}
	return b.String()
}

// genBenchModelFilters returns a handful of enabled filters representative
// of a real .tat file: one per log level plus one that matches broadly, so
// a benchmark run touches a realistic mix of match/no-match lines.
func genBenchModelFilters(b *testing.B) []filterfiles.Filter {
	b.Helper()
	specs := []string{"^debug", "^info", "^warn", "^error", "request id"}
	filters := make([]filterfiles.Filter, len(specs))
	for i, text := range specs {
		re, err := filterfiles.CompileRegex(text, false)
		if err != nil {
			b.Fatalf("CompileRegex(%q) failed: %v", text, err)
		}
		filters[i] = filterfiles.Filter{
			XML:       filterfiles.FilterXML{Text: text},
			Regex:     re,
			IsEnabled: true,
			BackColor: "#87CEFA",
		}
	}
	return filters
}

// benchmarkView measures repeated View() calls against a model with an
// unchanged filter set -- the common case of cursor movement or a window
// resize between keystrokes, where nothing about the filters has changed
// since the last render.
func benchmarkView(b *testing.B, n int) {
	b.Helper()
	b.Setenv("XDG_CONFIG_HOME", b.TempDir())

	filters := genBenchModelFilters(b)
	scanner := bufio.NewScanner(strings.NewReader(genBenchLog(n)))
	m := initialModel(filters, scanner, filepath.Join(b.TempDir(), "filters.tat"), filterfiles.TextAnalysisToolSettings{})

	newM, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	m = newM.(model)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkView_Medium(b *testing.B) { benchmarkView(b, 10_000) }
func BenchmarkView_Large(b *testing.B)  { benchmarkView(b, 100_000) }
func BenchmarkView_Huge(b *testing.B)   { benchmarkView(b, 1_000_000) }
