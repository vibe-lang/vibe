package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
	"github.com/vibe-lang/vibe/pkg/interpreter"
	"github.com/vibe-lang/vibe/pkg/lexer"
	"github.com/vibe-lang/vibe/pkg/parser"
)

// main is the entry point for the Vibe language interpreter.
// It sets up the command-line interface using the Cobra library and
// handles the execution of subcommands.
//
// Usage examples:
//   - Run a Vibe script: vibe run script.vb
//   - Start the Vibe REPL: vibe repl
//
// Version is set at build time via -ldflags
// x-release-please-start-version
var Version = "0.2.3"

// x-release-please-end

func main() {
	var rootCmd = &cobra.Command{
		Use:     "vibe",
		Short:   "Vibe is a Ruby-like programming language with type support",
		Version: Version,
		Long: `Vibe is an interpreted programming language similar to Ruby but with type support.
It aims to provide a pleasant and ergonomic developer experience while maintaining strong type safety.`,
	}

	var runCmd = &cobra.Command{
		Use:   "run [file] [args...]",
		Short: "Run a Vibe script",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runFile(args[0], args[1:])
		},
	}

	var replCmd = &cobra.Command{
		Use:   "repl",
		Short: "Start Vibe REPL (interactive shell)",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Starting Vibe REPL...")
			startRepl()
		},
	}

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(replCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// runFile executes a Vibe script file.
// It performs the following steps:
// 1. Read the file content
// 2. Create a lexer to tokenize the input
// 3. Create a parser to build an AST
// 4. Check for parsing errors
// 5. Create an interpreter to evaluate the AST
// 6. Execute the program
//
// If any errors occur during this process, they will be printed to stdout
// and the program will exit with a non-zero status code.
//
// Parameters:
//   - filename: The path to the Vibe script file to execute
func runFile(filename string, scriptArgs ...[]string) {
	// Read the file
	input, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		os.Exit(1)
	}

	// Tokenize the input with the lexer
	l := lexer.New(string(input))

	// Parse the tokens into an AST
	p := parser.New(l)
	program := p.ParseProgram()

	// Check for parsing errors
	if len(p.Errors()) != 0 {
		printParserErrors(p.Errors())
		os.Exit(1)
	}

	// Create an interpreter and evaluate the program
	i := interpreter.New()

	// Set CLI arguments if provided
	var args []string
	if len(scriptArgs) > 0 {
		args = scriptArgs[0]
	}
	i.SetArgs(filename, args)

	result := i.Eval(program)

	// Check for runtime errors
	if result != nil && result.Type() == interpreter.ERROR_OBJ {
		fmt.Println(result.Inspect())
		os.Exit(1)
	}
}

// printParserErrors prints parsing errors to stdout.
// This function formats the error messages in a user-friendly way
// to help the programmer identify and fix syntax errors in their code.
//
// Parameters:
//   - errors: A slice of error message strings
func printParserErrors(errors []string) {
	fmt.Println("Parser errors:")
	for _, msg := range errors {
		fmt.Printf("\t%s\n", msg)
	}
}

// evaluateAndPrint evaluates a Vibe expression and prints the result.
func evaluateAndPrint(i *interpreter.Interpreter, input string) {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		printParserErrors(p.Errors())
		return
	}

	result := i.Eval(program)
	if result != nil && result.Type() != interpreter.NIL_OBJ {
		fmt.Println("Result:", result.Inspect())
	}
}

// startRepl starts the interactive REPL (Read-Eval-Print Loop).
// It reads user input with readline support (history, arrow keys).
func startRepl() {
	i := interpreter.New()

	// Store history in ~/.vibe_history
	homeDir, _ := os.UserHomeDir()
	historyFile := filepath.Join(homeDir, ".vibe_history")

	rl, err := readline.NewEx(&readline.Config{
		Prompt:            "vibe> ",
		HistoryFile:       historyFile,
		HistoryLimit:      1000,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		fmt.Printf("Error initializing REPL: %s\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	fmt.Println("Vibe REPL (interactive shell)")
	fmt.Println("Type expressions to evaluate them")
	fmt.Println("Press Ctrl+D to exit")

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			continue
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		if line == "" {
			continue
		}

		evaluateAndPrint(i, line)
	}

	fmt.Println("\nGoodbye!")
}
