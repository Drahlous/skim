package filterfiles

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestCompileRegex(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		caseSensitive bool
		matches       string
		wantMatch     bool
	}{
		{"case-insensitive matches different case", "hello", false, "HELLO world", true},
		{"case-insensitive matches same case", "hello", false, "hello world", true},
		{"case-sensitive rejects different case", "hello", true, "HELLO world", false},
		{"case-sensitive matches same case", "hello", true, "hello world", true},
		{"no match", "goodbye", false, "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := CompileRegex(tt.text, tt.caseSensitive)
			if err != nil {
				t.Fatalf("CompileRegex(%q, %v) returned unexpected error: %v", tt.text, tt.caseSensitive, err)
			}
			if got := re.MatchString(tt.matches); got != tt.wantMatch {
				t.Errorf("MatchString(%q) = %v, want %v", tt.matches, got, tt.wantMatch)
			}
		})
	}
}

func TestCompileRegexInvalidPattern(t *testing.T) {
	if _, err := CompileRegex("([unclosed", false); err == nil {
		t.Fatal("CompileRegex with invalid pattern returned no error")
	}
}

func TestReadFilterFile(t *testing.T) {
	const xmlContent = `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<TextAnalysisTool.NET version="2023-04-25" showOnlyFilteredLines="False">
  <filters>
    <filter enabled="y" excluding="n" description="" backColor="87cefa" type="matches_text" case_sensitive="n" regex="y" text="^debug" />
    <filter enabled="n" excluding="n" description="" backColor="ff0000" type="matches_text" case_sensitive="y" regex="y" text="goodbye" />
  </filters>
</TextAnalysisTool.NET>`

	dir := t.TempDir()
	path := dir + "/filter.tat"
	if err := os.WriteFile(path, []byte(xmlContent), 0o644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}

	settings, err := ReadFilterFile(path)
	if err != nil {
		t.Fatalf("ReadFilterFile returned unexpected error: %v", err)
	}

	if len(settings.Filters) != 2 {
		t.Fatalf("got %d filters, want 2", len(settings.Filters))
	}
	if settings.Filters[0].Text != "^debug" {
		t.Errorf("filter[0].Text = %q, want %q", settings.Filters[0].Text, "^debug")
	}
	if settings.Filters[0].Enabled != "y" {
		t.Errorf("filter[0].Enabled = %q, want %q", settings.Filters[0].Enabled, "y")
	}
	if settings.Filters[1].CaseSensitive != "y" {
		t.Errorf("filter[1].CaseSensitive = %q, want %q", settings.Filters[1].CaseSensitive, "y")
	}
}

func TestReadFilterFileMissing(t *testing.T) {
	if _, err := ReadFilterFile("/nonexistent/path/to/filter.tat"); err == nil {
		t.Fatal("ReadFilterFile with missing file returned no error")
	}
}

func TestCompileFilterRegularExpressions(t *testing.T) {
	settings := TextAnalysisToolSettings{
		Filters: []FilterXML{
			{Enabled: "y", BackColor: "87cefa", Text: "^debug", CaseSensitive: "n"},
			{Enabled: "n", BackColor: "ff0000", Text: "goodbye", CaseSensitive: "y"},
			{Enabled: "y", BackColor: "90ee90", Text: "Hello", CaseSensitive: ""},
		},
	}

	filters, err := CompileFilterRegularExpressions(settings)
	if err != nil {
		t.Fatalf("CompileFilterRegularExpressions returned unexpected error: %v", err)
	}
	if len(filters) != 3 {
		t.Fatalf("got %d filters, want 3", len(filters))
	}

	if !filters[0].IsEnabled {
		t.Error("filters[0].IsEnabled = false, want true")
	}
	if filters[0].CaseSensitive {
		t.Error("filters[0].CaseSensitive = true, want false")
	}
	if filters[0].BackColor != "#87CEFA" {
		t.Errorf("filters[0].BackColor = %q, want %q", filters[0].BackColor, "#87CEFA")
	}

	if filters[1].IsEnabled {
		t.Error("filters[1].IsEnabled = true, want false")
	}
	if !filters[1].CaseSensitive {
		t.Error("filters[1].CaseSensitive = false, want true")
	}

	// Unset case_sensitive attribute ("") should default to case-insensitive.
	if filters[2].CaseSensitive {
		t.Error("filters[2].CaseSensitive = true, want false for unset attribute")
	}
	if !filters[2].Regex.MatchString("hello there") {
		t.Error("filters[2] regex should match case-insensitively by default")
	}
}

func TestCompileFilterRegularExpressionsInvalidRegex(t *testing.T) {
	settings := TextAnalysisToolSettings{
		Filters: []FilterXML{
			{Enabled: "y", Text: "([unclosed"},
		},
	}

	if _, err := CompileFilterRegularExpressions(settings); err == nil {
		t.Fatal("CompileFilterRegularExpressions with invalid regex returned no error")
	}
}

func mustFilter(t *testing.T, text string, caseSensitive bool, enabled bool, backColor string) Filter {
	t.Helper()
	re, err := CompileRegex(text, caseSensitive)
	if err != nil {
		t.Fatalf("CompileRegex(%q) failed: %v", text, err)
	}
	return Filter{
		XML:           FilterXML{Text: text},
		Regex:         re,
		IsEnabled:     enabled,
		CaseSensitive: caseSensitive,
		BackColor:     backColor,
	}
}

func TestGetMatchingFilter(t *testing.T) {
	filters := []Filter{
		mustFilter(t, "^debug", false, true, "#87CEFA"),
		mustFilter(t, "goodbye", false, true, "#FF0000"),
		mustFilter(t, "goodbye", false, false, "#000000"), // disabled duplicate, should never win
	}

	t.Run("returns first enabled matching filter", func(t *testing.T) {
		got, ok := GetMatchingFilter(filters, "debug: starting up")
		if !ok {
			t.Fatal("expected a match, got none")
		}
		if got.BackColor != "#87CEFA" {
			t.Errorf("got BackColor %q, want %q", got.BackColor, "#87CEFA")
		}
	})

	t.Run("skips disabled filters", func(t *testing.T) {
		got, ok := GetMatchingFilter(filters, "goodbye world")
		if !ok {
			t.Fatal("expected a match, got none")
		}
		if got.BackColor != "#FF0000" {
			t.Errorf("got BackColor %q, want %q (the enabled filter, not the disabled duplicate)", got.BackColor, "#FF0000")
		}
	})

	t.Run("no match", func(t *testing.T) {
		_, ok := GetMatchingFilter(filters, "nothing interesting here")
		if ok {
			t.Error("expected no match, got one")
		}
	})

	t.Run("empty filter list", func(t *testing.T) {
		_, ok := GetMatchingFilter(nil, "anything")
		if ok {
			t.Error("expected no match against an empty filter list")
		}
	})
}

func TestWriteFilterFileRoundTrips(t *testing.T) {
	filters := []Filter{
		mustFilter(t, "^debug", false, true, "#87CEFA"),
		mustFilter(t, "goodbye", true, false, "#FF0000"),
	}
	meta := TextAnalysisToolSettings{Version: "2023-04-25", ShowOnlyFilteredLines: "True"}

	path := t.TempDir() + "/out.tat"
	if err := WriteFilterFile(path, meta, filters); err != nil {
		t.Fatalf("WriteFilterFile returned unexpected error: %v", err)
	}

	settings, err := ReadFilterFile(path)
	if err != nil {
		t.Fatalf("ReadFilterFile on the written file returned unexpected error: %v", err)
	}
	if settings.Version != "2023-04-25" {
		t.Errorf("Version = %q, want %q (preserved from meta)", settings.Version, "2023-04-25")
	}
	if settings.ShowOnlyFilteredLines != "True" {
		t.Errorf("ShowOnlyFilteredLines = %q, want %q (preserved from meta)", settings.ShowOnlyFilteredLines, "True")
	}

	got, err := CompileFilterRegularExpressions(settings)
	if err != nil {
		t.Fatalf("CompileFilterRegularExpressions returned unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d filters, want 2", len(got))
	}
	if got[0].XML.Text != "^debug" || !got[0].IsEnabled || got[0].BackColor != "#87CEFA" {
		t.Errorf("filter[0] round-tripped incorrectly: %+v", got[0])
	}
	if got[1].XML.Text != "goodbye" || got[1].IsEnabled || !got[1].CaseSensitive {
		t.Errorf("filter[1] round-tripped incorrectly (want disabled, case-sensitive): %+v", got[1])
	}
}

func TestWriteFilterFileWritesLiveStateNotStaleXML(t *testing.T) {
	// Simulate a filter loaded from a file, then toggled in the UI: the
	// bool fields change but the original parsed XML strings do not.
	f := mustFilter(t, "ERROR", false, false, "#FF0000")
	f.XML.Enabled = "y" // stale: the original file had this filter enabled
	f.IsEnabled = false // live state: it was toggled off since

	path := t.TempDir() + "/out.tat"
	if err := WriteFilterFile(path, TextAnalysisToolSettings{}, []Filter{f}); err != nil {
		t.Fatalf("WriteFilterFile returned unexpected error: %v", err)
	}

	settings, err := ReadFilterFile(path)
	if err != nil {
		t.Fatalf("ReadFilterFile returned unexpected error: %v", err)
	}
	if settings.Filters[0].Enabled != "n" {
		t.Errorf("written enabled=%q, want %q (live state, not the stale XML)", settings.Filters[0].Enabled, "n")
	}
}

func TestWriteFilterFileFillsDefaultsWhenMetaIsEmpty(t *testing.T) {
	path := t.TempDir() + "/out.tat"
	if err := WriteFilterFile(path, TextAnalysisToolSettings{}, nil); err != nil {
		t.Fatalf("WriteFilterFile returned unexpected error: %v", err)
	}

	settings, err := ReadFilterFile(path)
	if err != nil {
		t.Fatalf("ReadFilterFile returned unexpected error: %v", err)
	}
	if settings.Version == "" {
		t.Error("Version is empty, want a default value filled in")
	}
	if settings.ShowOnlyFilteredLines == "" {
		t.Error("ShowOnlyFilteredLines is empty, want a default value filled in")
	}
}

func TestWriteFilterFileInvalidPath(t *testing.T) {
	if err := WriteFilterFile("/nonexistent/dir/out.tat", TextAnalysisToolSettings{}, nil); err == nil {
		t.Fatal("WriteFilterFile with an unwritable path returned no error")
	}
}

func TestCountMatches(t *testing.T) {
	filters := []Filter{
		mustFilter(t, "^debug", false, true, "#87CEFA"),
		mustFilter(t, "goodbye", false, true, "#FF0000"),
		mustFilter(t, "goodbye", false, false, "#000000"), // disabled duplicate, should never count
	}
	lines := []string{
		"debug: one",
		"goodbye world",
		"debug: two",
		"nothing interesting",
		"goodbye again",
	}

	counts := CountMatches(filters, lines)

	if len(counts) != 3 {
		t.Fatalf("got %d counts, want 3 (one per filter)", len(counts))
	}
	if counts[0] != 2 {
		t.Errorf("counts[0] (^debug) = %d, want 2", counts[0])
	}
	if counts[1] != 2 {
		t.Errorf("counts[1] (goodbye) = %d, want 2", counts[1])
	}
	if counts[2] != 0 {
		t.Errorf("counts[2] (disabled duplicate) = %d, want 0", counts[2])
	}
}

func TestCountMatchesFirstEnabledFilterWinsAttribution(t *testing.T) {
	// Both filters match "goodbye world", but CountMatches should attribute
	// it only to the first enabled filter, matching GetMatchingFilter's
	// highlighting semantics rather than counting every regex match.
	filters := []Filter{
		mustFilter(t, "goodbye", false, true, "#FF0000"),
		mustFilter(t, "world", false, true, "#00FF00"),
	}

	counts := CountMatches(filters, []string{"goodbye world"})

	if counts[0] != 1 {
		t.Errorf("counts[0] = %d, want 1", counts[0])
	}
	if counts[1] != 0 {
		t.Errorf("counts[1] = %d, want 0 (line already attributed to filter 0)", counts[1])
	}
}

func TestCountMatchesEmptyInputs(t *testing.T) {
	if got := CountMatches(nil, []string{"a", "b"}); len(got) != 0 {
		t.Errorf("CountMatches(nil, ...) = %v, want empty", got)
	}
	filters := []Filter{mustFilter(t, "a", false, true, "#000000")}
	if got := CountMatches(filters, nil); got[0] != 0 {
		t.Errorf("CountMatches(filters, nil) = %v, want [0]", got)
	}
}

func TestGetMatchingLines(t *testing.T) {
	filters := []Filter{
		mustFilter(t, "^debug", false, true, "#87CEFA"),
	}

	scanner := bufio.NewScanner(strings.NewReader("debug: one\nnothing\ndebug: two\n"))

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	GetMatchingLines(filters, scanner)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if strings.Count(output, "Found line matching pattern") != 2 {
		t.Errorf("expected 2 matches printed, got output: %q", output)
	}
	if !strings.Contains(output, "debug: one") || !strings.Contains(output, "debug: two") {
		t.Errorf("expected both matching lines in output, got: %q", output)
	}
}
