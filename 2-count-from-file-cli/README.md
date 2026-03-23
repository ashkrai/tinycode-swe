# 2. count-from-file-cli

In Go (and Linux), **everything is a file** — an actual file on disk, a pipe from another program, keyboard input — they all look identical to your code. `bufio.Scanner` reads any of them the same way. You just decide *which* one to point it at.
```
user gives a filename  →  open that file  →  point Scanner at it
user gives nothing     →  point Scanner at os.Stdin (the pipe)



./wc myfile.txt        # reads a file
cat myfile.txt | ./wc  # reads from a pipe (no file given)



Run it — exactly these commands
# compile
go build -o wc .

# test 1: read a file
./wc sample.txt

# test 2: file doesn't exist — should print an error and exit
./wc ghost.txt

# test 3: pipe input — no filename at all
cat sample.txt | ./wc

# test 4: also works with echo
echo "hello world" | ./wc
```

Expected output for `sample.txt`:
```
lines: 5
words: 14
chars: 70


What each new thing does
bufio.Scanner — reads a file one line at a time. Without it you'd have to manually detect newlines in raw bytes. It handles that for you.

strings.Fields(line) — splits "hello   world" into ["hello", "world"]. Handles multiple spaces, tabs, anything. Returns a slice so len() of it is the word count.

fmt.Fprintln(os.Stderr, ...) — errors go to stderr, not stdout. This matters because when you pipe output to another program (./wc file.txt | grep lines), error messages won't mix in with the real data.

os.Exit(1) — exit code 1 means something went wrong. Exit code 0 (the default) means success. Shell scripts and CI pipelines check this number to know if your program failed.

defer f.Close() — runs when main() ends, no matter what. Closes the file and frees the OS resource. Always do this right after os.Open.