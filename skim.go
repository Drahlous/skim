package main

import (
	"bufio"
	"example/user/skim/filterfiles"
	"fmt"
	"os"
	"regexp"
)

// Path of log file
const LOG_FILE = "./examples/simple.log"

// Path of xml filter file
const FILTER_FILE = "./examples/simple_filter_two.tat"

func GetMatchingLines(patterns []regexp.Regexp, scanner *bufio.Scanner) {

	// Read line-by-line
	for scanner.Scan() {
		line := scanner.Text()

		for _, pattern := range patterns {

			// Check whether the line matches our debug regex
			if pattern.MatchString(line) {
				fmt.Println("Found line matching pattern: ", line)
			}
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

	// Compile the extracted filters into regular expressions
	patterns, err := filterfiles.CompileFilterRegularExpressions(filterSettings)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Read the log file line-by-line
	logfile, err := os.Open(LOG_FILE)
	if err != nil {
		fmt.Println(err)
		return
	}
	scanner := bufio.NewScanner(logfile)

	GetMatchingLines(patterns, scanner)
}
