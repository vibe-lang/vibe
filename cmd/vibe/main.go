package main

import (
	"fmt"
	"os"

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
func main() {
	// Define the root command for the Vibe CLI
	var rootCmd = &cobra.Command{
		Use:   "vibe",
		Short: "Vibe is a Ruby-like programming language with type support",
		Long: `Vibe is an interpreted programming language similar to Ruby but with type support.
It aims to provide a pleasant and ergonomic developer experience while maintaining strong type safety.`,
	}

	// Define the 'run' subcommand for executing Vibe scripts
	var runCmd = &cobra.Command{
		Use:   "run [file]",
		Short: "Run a Vibe script",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runFile(args[0])
		},
	}

	// Define the 'repl' subcommand for interactive use
	var replCmd = &cobra.Command{
		Use:   "repl",
		Short: "Start Vibe REPL (interactive shell)",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Starting Vibe REPL...")
			fmt.Println("Not implemented yet. Coming soon!")
		},
	}

	// Add subcommands to the root command
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(replCmd)

	// Execute the command tree
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
func runFile(filename string) {
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