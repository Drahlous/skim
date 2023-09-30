package main

import (
	"fmt"
	"os"
    "bufio"
)

func main() {
    fmt.Println("Hello world.")
    
    // Read line-by-line
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        line := scanner.Text()
        fmt.Println("read line: ", line)
    }
}
