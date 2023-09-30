package main

import (
	"fmt"
	"os"
    "bufio"
    "regexp"
)


func main() {
    
    // Compile a single debug pattern.
    // TODO: Read a list of patterns from a file
    debug_pattern, err := regexp.Compile("debug")
    if err != nil {
        return
    }

    // Read line-by-line
    // TODO: Allow the user to specify a logfile
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        line := scanner.Text()

        // Check whether the line matches our debug regex
        if debug_pattern.MatchString(line) {
            fmt.Println("Found line matching pattern: ", line)
        }
    }
}
