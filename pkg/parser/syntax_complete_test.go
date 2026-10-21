package parser

import (
	"testing"

	"github.com/vibe-lang/vibe/pkg/lexer"
)

func parseErrors(src string) []string {
	p := New(lexer.New(src))
	p.ParseProgram()
	return p.Errors()
}

func TestParsesSyntaxForms(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"class with parent and attrs", "class A < B\n attr_accessor :x, :y\n def initialize(x)\n @x = x\n end\nend"},
		{"module with module_function", "module M\n def self.f\n 1\n end\nend"},
		{"block with params", "[1, 2].each { |x| puts(x) }"},
		{"do block", "[1, 2].each do |x|\n puts(x)\nend"},
		{"keyword and default params", "def f(a, b = 1, *rest, k:, j: 2, **opts, &blk)\nend"},
		{"safe navigation", "a&.b&.c"},
		{"rocket hash", `{"a" => 1, :b => 2}`},
		{"splat in array and call", "f(*xs)\n[1, *xs]"},
		{"case in with guard", "case x\nin [a, b] if a > b\n 1\nend"},
		{"rescue ensure else", "begin\n 1\nrescue A, B => e\n 2\nelse\n 3\nensure\n 4\nend"},
		{"postfix modifiers", "x = 1 if y\nx = 2 unless y\nx += 1 while x < 5"},
		{"ternary and nil coalesce", "a ? b : c\na ?? b"},
		{"ranges", "(1..5)\n(1...5)\n(1..)"},
		{"arrow function", "f = -> (x) { x + 1 }"},
		{"string interpolation", `"a ${b + 1} c"`},
		{"negative literal receiver", "-5.abs"},
		{"op assign variants", "a ||= 1\nb &&= 2\nc **= 3\nd <<= 1"},
		{"struct with method", "struct P\n x: int\n def dbl()\n x\n end\nend"},
		{"enum", "enum Color\n Red\n Green\nend"},
		{"multiple assignment", "a, b = 1, 2\na, *b = [1, 2, 3]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if errs := parseErrors(tt.input); len(errs) > 0 {
				t.Errorf("unexpected parse errors for %q: %v", tt.input, errs)
			}
		})
	}
}

func TestRejectsMalformedSyntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unclosed paren", "f(1, 2"},
		{"unclosed block", "[1].each { |x| x"},
		{"missing end", "def f\n 1"},
		{"dangling operator", "1 +"},
		{"unclosed array", "[1, 2"},
		{"unclosed hash", "{a: 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if errs := parseErrors(tt.input); len(errs) == 0 {
				t.Errorf("expected parse errors for %q", tt.input)
			}
		})
	}
}
