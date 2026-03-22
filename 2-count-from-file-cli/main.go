package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	// ── Step 1: decide where to read from ────────────────────────────
	//
	// os.Args[1] would be the filename if the user typed one.
	// If they didn't, we fall back to os.Stdin (the pipe).
	//
	//   ./wc myfile.txt        → os.Args = ["./wc", "myfile.txt"]
	//   cat myfile.txt | ./wc  → os.Args = ["./wc"]   (no filename)

	var input *os.File // *os.File is the type for any open file

	if len(os.Args) >= 2 {
		// A filename was given — try to open it.
		filename := os.Args[1]

		f, err := os.Open(filename)
		if err != nil {
			// os.Open returns a descriptive error automatically:
			//   "open myfile.txt: no such file or directory"
			//   "open myfile.txt: permission denied"
			// We just print it and exit with code 1 (means "error").
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer f.Close() // close the file when main() finishes
		input = f

	} else {
		// No filename given — read from the pipe / keyboard.
		input = os.Stdin
	}

	// ── Step 2: count lines, words, and characters ───────────────────
	//
	// bufio.Scanner reads one line at a time.
	// Each call to scanner.Scan() loads the next line.
	// scanner.Text() gives you that line as a string.

	var lines, words, chars int

	scanner := bufio.NewScanner(input)

	for scanner.Scan() {        // moves to next line; returns false at end-of-file
		line := scanner.Text() // the current line, without the newline character

		lines++
		words += len(strings.Fields(line)) // Fields splits on any whitespace
		chars += len(line) + 1             // +1 for the newline we stripped
	}

	// scanner.Err() is nil if everything went fine.
	// It is non-nil if something went wrong mid-read (e.g. disk error).
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error reading:", err)
		os.Exit(1)
	}

	// ── Step 3: print the result ─────────────────────────────────────
	fmt.Println("lines:", lines)
	fmt.Println("words:", words)
	fmt.Println("chars:", chars)
}