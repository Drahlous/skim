package main

import (
	"bufio"
	"example/user/skim/filterfiles"
	"example/user/skim/ui"
	"fmt"
	"os"
)

// Path of log file
const LOG_FILE = "./examples/simple.log"

// Path of xml filter file
const FILTER_FILE = "./examples/simple_filter_two.tat"

func main() {

	//var filter_file = *flag.String("filter", "./examples/simple_filter_two.tat", "supply the path to a TAT filter file")
	//var log_file = *flag.String("log", "./examples/simple.log", "supply the path to the input log file")

	//flag.Parse()

	filter_file := FILTER_FILE
	log_file := LOG_FILE

	// Read filter settings from the XML file
	filterSettings, err := filterfiles.ReadFilterFile(filter_file)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print the parsed TAT filter settings
	fmt.Println("TextAnalysisTool Version: " + filterSettings.Version)
	fmt.Println("TextAnalysisTool showOnlyFilteredLines: " + filterSettings.ShowOnlyFilteredLines)

	// Compile the extracted filters into regular expressions
	filters, err := filterfiles.CompileFilterRegularExpressions(filterSettings)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Read the log file line-by-line
	logfile, err := os.Open(log_file)
	if err != nil {
		fmt.Println(err)
		return
	}
	scanner := bufio.NewScanner(logfile)

	filterfiles.GetMatchingLines(filters, scanner)

	ui.RunUI(filters)
}
