package main

import (
	"fmt"
	"os"
)

func main() {
	// When you type:  ./greet --name Alice
	// Go sees:        os.Args = ["./greet", "--name", "Alice"]
	//
	// os.Args is just a slice of strings — the words you typed, split by spaces.
	// os.Args[0] is always the program name itself.
	// os.Args[1] is the first word after it.  etc.

	// If the user didn't type a name, print a helpful message and stop.
	if len(os.Args) < 3 {
		fmt.Println("Usage: ./greet --name Alice")
		os.Exit(1)
	}

	// os.Args[1] is "--name"
	// os.Args[2] is whatever the user typed after it — "Alice", "Bob", etc.
	name := os.Args[2]

	fmt.Println("Hello,", name)
}