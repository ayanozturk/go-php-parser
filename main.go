package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide a file or folder as the first argument.")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	fmt.Printf("Processing: %s\n", inputPath)
}
