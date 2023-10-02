package filterfiles

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
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
	XML   FilterXML
	Regex regexp.Regexp
}

func ReadFilterFile(filter_file_path string) (TextAnalysisToolSettings, error) {

	var textAnalysisToolSettings TextAnalysisToolSettings

	// Read from the filter file
	xmlFile, err := os.Open(filter_file_path)
	if err != nil {
		return textAnalysisToolSettings, err
	}

	fmt.Println("Successfully opened ", filter_file_path)

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

	regex, err := regexp.Compile(XML.Text)
	if err != nil {
		fmt.Println(err)
		return f, err
	}
	f.Regex = *regex
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
