package main

import (
	"bufio"
	"example/user/skim/filterfiles"
	"example/user/skim/ui"
	"flag"
	"fmt"
	"os"
)

func run(filter_file string, log_file string) {

	// Read filter settings from the XML file
	filterSettings, err := filterfiles.ReadFilterFile(filter_file)
	if err != nil {
		fmt.Println(err)
		return
	}

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

	ui.RunUI(filters, scanner)
}

func main() {

	// Parse Command Line Options
	filter_file := flag.String("filter", "./examples/simple_filter_two.tat", "supply the path to a TAT filter file")
	log_file := flag.String("log", "./examples/simple_longer.log", "supply the path to the input log file")
	flag.Parse()

	// Run the program
	run(*filter_file, *log_file)
}
