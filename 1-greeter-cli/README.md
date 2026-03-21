A CLI (Command Line Interface) is:
👉 A program you run by typing commands in the terminal.


# run it without compiling first (just to see it work)
go run main.go --name Alice

# compile it into a real binary , Same output. But now it's a standalone binary.
go build -o greet .
./greet --name Alice 





You typed:   ./greet --name Alice        # typing a command
               ↓       ↓      ↓
os.Args:     [0]      [1]    [2]
           "./greet" "--name" "Alice"    # passing arguments (--name Alice)

Hello, Alice.                            # getting output in terminal
