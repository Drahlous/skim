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

func makeFilter(XML FilterXML) (Filter, error) {

	var f Filter

	f.XML = XML

	// Translate individual fields for ease of use
	if f.XML.Enabled == "y" {
		f.IsEnabled = true
	} else {
		f.IsEnabled = false
	}
	f.CaseSensitive = f.XML.CaseSensitive == "y"
	f.Excluding = f.XML.Excluding == "y"

	regex, err := CompileRegex(XML.Text, f.CaseSensitive)
	if err != nil {
		fmt.Println(err)
		return f, err
	}
	f.Regex = regex

	f.BackColor = fmt.Sprintf("#%s", strings.ToUpper(f.XML.BackColor))

	return f, nil
}

func CompileFilterRegularExpressions(filterSettings TextAnalysisToolSettings) ([]Filter, error) {
	var filters []Filter

	for _, XMLFilter := range filterSettings.Filters {
		f, err := makeFilter(XMLFilter)
		if err != nil {
			fmt.Println(err)
			return filters, err
		}
		filters = append(filters, f)
	}

	return filters, nil
}

// filterToXML converts a Filter's live state back into a FilterXML for
// serialization. It reads from the bool/BackColor fields rather than f.XML
// directly, since in-session edits (toggling enabled/case-sensitive, regex
// text changes) update those fields but leave the original parsed f.XML
// strings untouched. There's no UI action to toggle Excluding, but it's
// still read from the live bool field for the same reason and for
// consistency with Enabled/CaseSensitive, rather than the stale XML string.
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

// GetMatchingFilter returns the first enabled, non-excluding filter whose
// regex matches line, for highlighting purposes. Excluding filters are never
// returned here: they mean "hide this line" rather than "color this line"
// and are handled separately by IsExcluded, which callers should check
// first (an excluded line should never be shown, regardless of whether it
// would also match a highlighting filter).
func GetMatchingFilter(filters []Filter, line string) (Filter, bool) {
	var filter Filter
	for _, filter := range filters {

		// Only continue if this filter is enabled and not an exclusion filter
		if !filter.IsEnabled || filter.Excluding {
			continue
		}

		// Check whether the line matches the filter's regex
		re := filter.Regex
		if re.MatchString(line) {
			return filter, true
		}
	}
	return filter, false
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
