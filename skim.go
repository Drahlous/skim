package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
)

// Path of xml filter file
const FILTER_FILE = "./examples/simple_filter.tat"

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
	Filters []Filter `xml:"filters>filter"`
}

type Filter struct {
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

func ReadFilterFile(filter_file_path string) (TextAnalysisToolSettings, error) {

	var textAnalysisToolSettings TextAnalysisToolSettings

	// Read from the filter file
	xmlFile, err := os.Open(filter_file_path)
	if err != nil {
		return textAnalysisToolSettings, err
	}

	fmt.Println("Successfully opened ", FILTER_FILE)

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

func GetMatchingLines(pattern string, scanner bufio.Scanner) {
	// Compile a single debug pattern.
	// TODO: Read a list of patterns from a file
	debug_pattern, err := regexp.Compile("debug")
	if err != nil {
		return
	}

	// Read line-by-line
	// TODO: Allow the user to specify a logfile
	for scanner.Scan() {
		line := scanner.Text()

		// Check whether the line matches our debug regex
		if debug_pattern.MatchString(line) {
			fmt.Println("Found line matching pattern: ", line)
		}
	}
}

func main() {
	// Read filter settings from the XML file
	filterSettings, err := ReadFilterFile(FILTER_FILE)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print the parsed TAT filter settings
	fmt.Println("TextAnalysisTool Version: " + filterSettings.Version)
	fmt.Println("TextAnalysisTool showOnlyFilteredLines: " + filterSettings.ShowOnlyFilteredLines)
	for _, filter := range filterSettings.Filters {
		fmt.Println("filter text: " + filter.Text)
	}
}
