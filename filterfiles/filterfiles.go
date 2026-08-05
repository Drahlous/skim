package filterfiles

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

/*
Example Filter File:
<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<TextAnalysisTool.NET version="2023-04-25" showOnlyFilteredLines="False">
  <filters>
    <filter enabled="y" excluding="n" description="" backColor="87cefa" type="matches_text" case_sensitive="n" regex="y" text="^debug" />
  </filters>
</TextAnalysisTool.NET>
*/

// Structs for unmarshaling the XML filter file
type TextAnalysisToolSettings struct {
	XMLName               xml.Name `xml:"TextAnalysisTool.NET"`
	Version               string   `xml:"version,attr"`
	ShowOnlyFilteredLines string   `xml:"showOnlyFilteredLines,attr"`
	// Array of all Filters in the file
	Filters []FilterXML `xml:"filters>filter"`
}

type FilterXML struct {
	XMLName       xml.Name `xml:"filter"`
	Enabled       string   `xml:"enabled,attr"`
	Excluding     string   `xml:"excluding,attr"`
	Description   string   `xml:"description,attr"`
	BackColor     string   `xml:"backColor,attr"`
	Type          string   `xml:"type,attr"`
	CaseSensitive string   `xml:"case_sensitive,attr"`
	Regex         string   `xml:"regex,attr"`
	Text          string   `xml:"text,attr"`
}

type Filter struct {
	XML           FilterXML
	Regex         regexp.Regexp
	IsEnabled     bool
	CaseSensitive bool
	Excluding     bool
	BackColor     string
}

// neverMatchRegex never matches anything, including the empty string (no
// single rune can satisfy a negated class spanning the entire Unicode
// range). It stands in for a filter's Regex when its configured pattern
// fails to compile, so a disabled invalid filter -- see makeFilter -- stays
// safe to evaluate (e.g. if a caller later flips IsEnabled without first
// fixing the regex) instead of panicking on a zero-value regexp.Regexp.
var neverMatchRegex = regexp.MustCompile(`[^\x00-\x{10FFFF}]`)

// CompileRegex compiles text into a regexp, applying a case-insensitive
// flag unless caseSensitive is set.
func CompileRegex(text string, caseSensitive bool) (regexp.Regexp, error) {
	pattern := text
	if !caseSensitive {
		pattern = "(?i)" + pattern
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return regexp.Regexp{}, err
	}

	return *regex, nil
}

func ReadFilterFile(filter_file_path string) (TextAnalysisToolSettings, error) {

	var textAnalysisToolSettings TextAnalysisToolSettings

	// Read from the filter file
	xmlFile, err := os.Open(filter_file_path)
	if err != nil {
		return textAnalysisToolSettings, err
	}

	// defer the closing of our xml file so we can parse it later
	defer xmlFile.Close()

	// Parse the XML settings
	byteValue, _ := io.ReadAll(xmlFile)
	err = xml.Unmarshal(byteValue, &textAnalysisToolSettings)
	if err != nil {
		return textAnalysisToolSettings, err
	}

	return textAnalysisToolSettings, nil
}

// makeFilter converts a single parsed FilterXML into a compiled Filter.
// index is this filter's position in the file's <filters> list (0-based),
// used only to identify it in the error message if its regex fails to
// compile. On a regex compile failure, makeFilter does not fail outright:
// it returns a Filter that is forced disabled (regardless of the file's own
// enabled="y/n") with neverMatchRegex standing in for the unusable pattern,
// plus a descriptive error identifying the offending filter -- so one bad
// filter in a hand-edited or exported .tat file can't take down the whole
// load (see CompileFilterRegularExpressions and the issue this fixed).
// f.XML.Text keeps the original, still-invalid pattern text so the filter
// editor shows it as-is for the user to fix.
func makeFilter(index int, XML FilterXML) (Filter, error) {

	var f Filter

	f.XML = XML
	f.IsEnabled = f.XML.Enabled == "y"
	f.CaseSensitive = f.XML.CaseSensitive == "y"
	f.Excluding = f.XML.Excluding == "y"
	f.BackColor = fmt.Sprintf("#%s", strings.ToUpper(f.XML.BackColor))

	regex, err := CompileRegex(XML.Text, f.CaseSensitive)
	if err != nil {
		f.IsEnabled = false
		f.Regex = *neverMatchRegex
		desc := XML.Description
		if desc == "" {
			desc = fmt.Sprintf("filter #%d", index+1)
		} else {
			desc = fmt.Sprintf("filter #%d (%q)", index+1, desc)
		}
		return f, fmt.Errorf("%s: disabled, invalid regex %q: %w", desc, XML.Text, err)
	}
	f.Regex = regex

	return f, nil
}

// CompileFilterRegularExpressions compiles every filter in filterSettings.
// A filter whose regex fails to compile is disabled and kept in the
// returned slice (see makeFilter) rather than aborting the whole load, so a
// single bad filter in an otherwise-valid file doesn't prevent using the
// rest; its error is instead collected into the returned warnings slice for
// the caller to log/surface. A nil warnings slice means every filter
// compiled cleanly.
func CompileFilterRegularExpressions(filterSettings TextAnalysisToolSettings) ([]Filter, []error) {
	var filters []Filter
	var warnings []error

	for i, XMLFilter := range filterSettings.Filters {
		f, err := makeFilter(i, XMLFilter)
		if err != nil {
			warnings = append(warnings, err)
		}
		filters = append(filters, f)
	}

	return filters, warnings
}

// filterToXML converts a Filter's live state back into a FilterXML for
// serialization. It reads from the bool/BackColor fields rather than f.XML
// directly, since in-session edits (toggling enabled/case-sensitive/excluding,
// regex text and description changes, picking a color) update those fields
// but leave the original parsed f.XML strings untouched.
func filterToXML(f Filter) FilterXML {
	enabled := "n"
	if f.IsEnabled {
		enabled = "y"
	}
	caseSensitive := "n"
	if f.CaseSensitive {
		caseSensitive = "y"
	}
	excluding := "n"
	if f.Excluding {
		excluding = "y"
	}

	regexAttr := f.XML.Regex
	if regexAttr == "" {
		regexAttr = "y"
	}
	filterType := f.XML.Type
	if filterType == "" {
		filterType = "matches_text"
	}

	return FilterXML{
		Enabled:       enabled,
		Excluding:     excluding,
		Description:   f.XML.Description,
		BackColor:     strings.ToLower(strings.TrimPrefix(f.BackColor, "#")),
		Type:          filterType,
		CaseSensitive: caseSensitive,
		Regex:         regexAttr,
		Text:          f.XML.Text,
	}
}

// HideUnmatchedByDefault reports whether meta's showOnlyFilteredLines
// attribute requests that non-matching log lines start out hidden. TAT's own
// default when the attribute is absent (or unrecognized) is "False", i.e.
// show everything, matching WriteFilterFile's default.
func HideUnmatchedByDefault(meta TextAnalysisToolSettings) bool {
	return strings.EqualFold(meta.ShowOnlyFilteredLines, "True")
}

// WriteFilterFile serializes filters back to a .tat file at path, the
// inverse of ReadFilterFile + CompileFilterRegularExpressions. meta supplies
// the root element's version/showOnlyFilteredLines attributes (normally the
// TextAnalysisToolSettings the filters were originally loaded from, so a
// save preserves them); if either is empty, a reasonable default is used so
// a filter set built entirely from scratch in the UI still produces a valid
// file.
func WriteFilterFile(path string, meta TextAnalysisToolSettings, filters []Filter) error {
	version := meta.Version
	if version == "" {
		version = "2023-04-25"
	}
	showOnlyFilteredLines := meta.ShowOnlyFilteredLines
	if showOnlyFilteredLines == "" {
		showOnlyFilteredLines = "False"
	}

	settings := TextAnalysisToolSettings{
		Version:               version,
		ShowOnlyFilteredLines: showOnlyFilteredLines,
	}
	for _, f := range filters {
		settings.Filters = append(settings.Filters, filterToXML(f))
	}

	body, err := xml.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	out := []byte(`<?xml version="1.0" encoding="utf-8" standalone="yes"?>` + "\n")
	out = append(out, body...)
	out = append(out, '\n')

	return os.WriteFile(path, out, 0o644)
}

// GetMatchingFilterIndex is GetMatchingFilter, but returns the index into
// filters of the matching filter instead of a copy of it, for callers (like
// a per-line match cache) that need to attribute a match back to its
// position rather than its value.
func GetMatchingFilterIndex(filters []Filter, line string) (int, bool) {
	for i, filter := range filters {
		// Only continue if this filter is enabled and not an exclusion filter
		if !filter.IsEnabled || filter.Excluding {
			continue
		}

		// Check whether the line matches the filter's regex
		if filter.Regex.MatchString(line) {
			return i, true
		}
	}
	return -1, false
}

// GetMatchingFilter returns the first enabled, non-excluding filter whose
// regex matches line, for highlighting purposes. Excluding filters are never
// returned here: they mean "hide this line" rather than "color this line"
// and are handled separately by IsExcluded, which callers should check
// first (an excluded line should never be shown, regardless of whether it
// would also match a highlighting filter).
func GetMatchingFilter(filters []Filter, line string) (Filter, bool) {
	idx, ok := GetMatchingFilterIndex(filters, line)
	if !ok {
		var filter Filter
		return filter, false
	}
	return filters[idx], true
}

// IsExcluded reports whether line matches any enabled excluding filter. An
// excluded line should be hidden unconditionally, regardless of hideUnmatched
// or whether it would otherwise match a highlighting filter, and regardless
// of filter order: unlike GetMatchingFilter's first-match-wins highlighting,
// exclusion is checked against every enabled excluding filter.
func IsExcluded(filters []Filter, line string) bool {
	for _, filter := range filters {
		if !filter.IsEnabled || !filter.Excluding {
			continue
		}
		if filter.Regex.MatchString(line) {
			return true
		}
	}
	return false
}

// CountMatches returns, for each filter (by index, matching filters'
// order), how many lines it is the highlighting match for. This follows the
// same first-enabled-filter-wins attribution as GetMatchingFilter, so a
// count reflects exactly the lines that filter is shown coloring, not
// simply every line its regex happens to match. Excluding filters are
// skipped, same as GetMatchingFilter: they hide lines, they don't "win"
// highlighting attribution for them.
func CountMatches(filters []Filter, lines []string) []int {
	counts := make([]int, len(filters))
	for _, line := range lines {
		for i, filter := range filters {
			if !filter.IsEnabled || filter.Excluding {
				continue
			}
			if filter.Regex.MatchString(line) {
				counts[i]++
				break
			}
		}
	}
	return counts
}

func GetMatchingLines(filters []Filter, scanner *bufio.Scanner) {

	// Read line-by-line
	for scanner.Scan() {
		line := scanner.Text()

		for _, filter := range filters {

			re := filter.Regex
			// Check whether the line matches our debug regex
			if re.MatchString(line) {
				fmt.Println("Found line matching pattern: ", line)
			}
		}
	}
}
