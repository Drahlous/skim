package main

import (
	"bufio"
	"example/user/skim/filterfiles"
	"fmt"
	"regexp"
)

// Path of xml filter file
const FILTER_FILE = "./examples/simple_filter.tat"

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
	filterSettings, err := filterfiles.ReadFilterFile(FILTER_FILE)
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
