package interpreter

import (
	"strings"
	"testing"
)

// runInspect evaluates src and returns the Inspect form of the result.
func runInspect(src string) string {
	res := evalInputAllowErrors(src)
	if _, ok := res.(*Error); ok {
		return res.Inspect()
	}
	return New().inspectValue(res)
}

func TestSyntaxValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// operators
		{"floor division of negative ints", `-7 / 2`, "-4"},
		{"floor modulo of negative ints", `-7 % 3`, "2"},
		{"power operator", `2 ** 10`, "1024"},
		{"negative literal receiver", `-5.abs`, "5"},
		{"unary minus below power", `-2 ** 2`, "-4"},
		{"and returns operand", `nil && 1`, "nil"},
		{"or returns operand", `nil || "x"`, `"x"`},
		{"nil coalescing", `nil ?? 3`, "3"},
		{"ternary", `true ? 1 : 2`, "1"},
		{"spaceship", `1 <=> 2`, "-1"},
		{"string repetition", `"ab" * 3`, `"ababab"`},
		{"bit shift", `1 << 4`, "16"},
		{"compound assignment", "x = 2\nx += 3\nx *= 2\nx", "10"},
		{"or-assign keeps existing", "x = 1\nx ||= 5\nx", "1"},
		{"or-assign sets nil", "x = nil\nx ||= 5\nx", "5"},

		// strings
		{"interpolation", "n = 4\n\"n=${n + 1}\"", `"n=5"`},
		{"upcase", `"abc".upcase`, `"ABC"`},
		{"gsub with regex", `"a1b22".gsub(/[0-9]+/, "#")`, `"a#b#"`},
		{"gsub with hash", `"cat".gsub("c", "c" => "b")`, `"bat"`},
		{"split and join", `"a,b,c".split(",").join("-")`, `"a-b-c"`},
		{"center", `"ab".center(6, "*")`, `"**ab**"`},
		{"string slice by range", `"hello"[1..3]`, `"ell"`},
		{"negative string index", `"hello"[-1]`, `"o"`},
		{"tr", `"hello".tr("el", "ip")`, `"hippo"`},
		{"two-arg replace", `"hello world".replace("world", "vibe")`, `"hello vibe"`},
		{"string to_i prefix", `"42abc".to_i`, "42"},
		{"format specifiers", `format("%05.1f|%-3d|%s", 3.14159, 7, "x")`, `"003.1|7  |x"`},

		// collections
		{"array map with block", `[1, 2, 3].map { |x| x * 2 }`, "[2, 4, 6]"},
		{"array select", `[1, 2, 3, 4].select { |x| x.even? }`, "[2, 4]"},
		{"array reduce with init", `[1, 2, 3].reduce(10) { |a, b| a + b }`, "16"},
		{"array each_slice", `[1, 2, 3, 4, 5].each_slice(2).to_a`, "[[1, 2], [3, 4], [5]]"},
		{"array flatten", `[1, [2, [3]]].flatten`, "[1, 2, 3]"},
		{"array sort_by", `["bb", "a", "ccc"].sort_by { |s| s.length }`, `["a", "bb", "ccc"]`},
		{"array group_by", `[1, 2, 3, 4].group_by { |x| x % 2 }`, "{1: [1, 3], 0: [2, 4]}"},
		{"array zip", `[1, 2].zip([3, 4])`, "[[1, 3], [2, 4]]"},
		{"array negative index", `[1, 2, 3][-1]`, "3"},
		{"array slice range", `[1, 2, 3, 4][1..2]`, "[2, 3]"},
		{"array splat", "a = [2, 3]\n[1, *a, 4]", "[1, 2, 3, 4]"},
		{"hash insertion order", `{b: 1, a: 2}.keys`, `["b", "a"]`},
		{"hash default block", "h = Hash.new { |hash, k| k * 2 }\nh[4]", "8"},
		{"hash default value", "h = Hash.new(0)\nh[:x] += 1\nh[:x]", "1"},
		{"hash merge", `{a: 1}.merge({b: 2})`, "{a: 1, b: 2}"},
		{"hash map_values", `{a: 1}.transform_values { |v| v + 1 }`, "{a: 2}"},
		{"hash string-key rocket", `{"a" => 1}["a"]`, "1"},
		{"range to_a", `(1..4).to_a`, "[1, 2, 3, 4]"},
		{"exclusive range", `(1...4).to_a`, "[1, 2, 3]"},
		{"string range", `("a".."d").to_a`, `["a", "b", "c", "d"]`},
		{"range step", `(1..10).step(3).to_a`, "[1, 4, 7, 10]"},

		// blocks and procs
		{"lambda call", "f = lambda { |x| x + 1 }\nf.call(2)", "3"},
		{"proc curry", "add = lambda { |a, b| a + b }\nadd.curry[1][2]", "3"},
		{"proc composition", "f = lambda { |x| x + 1 }\ng = lambda { |x| x * 2 }\n(f >> g).call(3)", "8"},
		{"yield with block", "def twice\n yield\n yield\nend\nn = 0\ntwice { n += 1 }\nn", "2"},
		{"block_given? true", "def f\n block_given?\nend\nf() { 1 }", "true"},
		{"block_given? false", "def f\n block_given?\nend\nf()", "false"},
		{"break from block", "[1, 2, 3].each { |x| break x * 10 if x == 2 }", "20"},
		{"next in block", "[1, 2, 3].map { |x| next 0 if x == 2\n x }", "[1, 0, 3]"},
		{"arrow function", "f = -> (x) { x * 3 }\nf(2)", "6"},

		// methods
		{"default argument", "def f(a, b = 2)\n a + b\nend\nf(1)", "3"},
		{"keyword argument", "def f(a:, b: 5)\n a + b\nend\nf(a: 1)", "6"},
		{"variadic argument", "def f(*xs)\n xs.length\nend\nf(1, 2, 3)", "3"},
		{"implicit return", "def f\n 42\nend\nf()", "42"},
		{"early return", "def f(x)\n return 1 if x\n 2\nend\nf(true)", "1"},
		{"recursion", "def fact(n)\n n <= 1 ? 1 : n * fact(n - 1)\nend\nfact(10)", "3628800"},
		{"UFCS", "def double(x)\n x * 2\nend\n4.double", "8"},

		// control flow
		{"unless", "unless false\n 1\nelse\n 2\nend", "1"},
		{"while with break", "i = 0\nwhile true\n i += 1\n break if i > 4\nend\ni", "5"},
		{"until loop", "i = 0\nuntil i == 3\n i += 1\nend\ni", "3"},
		{"for in range", "s = 0\nfor i in 1..4\n s += i\nend\ns", "10"},
		{"case when range", "case 5\nwhen 1..3 then :low\nwhen 4..6 then :mid\nelse :high\nend", ":mid"},
		{"case when class", "case \"x\"\nwhen Integer then 1\nwhen String then 2\nend", "2"},
		{"postfix if", "x = 1\nx = 9 if x == 1\nx", "9"},
		{"loop with break value", "loop do\n break 7\nend", "7"},
		{"integer times", "n = 0\n3.times { n += 2 }\nn", "6"},
		{"upto", "r = []\n1.upto(3) { |i| r << i }\nr", "[1, 2, 3]"},

		// classes
		{"class with attr_accessor", "class P\n attr_accessor :x\n def initialize(x)\n @x = x\n end\nend\np1 = P.new(3)\np1.x = 4\np1.x", "4"},
		{"inheritance and super", "class A\n def hi\n \"a\"\n end\nend\nclass B < A\n def hi\n super + \"b\"\n end\nend\nB.new.hi", `"ab"`},
		{"super with args", "class A\n def initialize(x)\n @x = x\n end\n def x\n @x\n end\nend\nclass B < A\n def initialize(x)\n super(x * 2)\n end\nend\nB.new(2).x", "4"},
		{"module include", "module M\n def hi\n 1\n end\nend\nclass C\n include M\nend\nC.new.hi", "1"},
		{"singleton method", "class C\n def self.make\n 5\n end\nend\nC.make", "5"},
		{"is_a? with builtin class", `1.is_a?(Integer)`, "true"},
		{"is_a? on subclass", "class A\nend\nclass B < A\nend\nB.new.is_a?(A)", "true"},
		{"respond_to?", `"x".respond_to?(:upcase)`, "true"},
		{"method_missing", "class G\n def method_missing(n, *a)\n n.to_s\n end\nend\nG.new.foo", `"foo"`},
		{"comparable", "class V\n include Comparable\n attr_reader :n\n def initialize(n)\n @n = n\n end\n def <=>(o)\n n <=> o.n\n end\nend\nV.new(1) < V.new(2)", "true"},
		{"struct method", "struct P\n x: int\n def dbl()\n self.x * 2\n end\nend\nP(x: 4).dbl()", "8"},
		{"struct kwargs to positional", "def f(item)\n item\nend\nf(item: 3)", "3"},

		// exceptions
		{"rescue by class", "begin\n raise ArgumentError, \"bad\"\nrescue ArgumentError => e\n e.message\nend", `"bad"`},
		{"ensure runs", "x = 0\nbegin\n 1\nensure\n x = 5\nend\nx", "5"},
		{"retry", "n = 0\nbegin\n n += 1\n raise \"x\" if n < 3\nrescue\n retry\nend\nn", "3"},
		{"custom exception subclass of Error", "class MyErr < Error\nend\nbegin\n raise MyErr, \"m\"\nrescue MyErr => e\n e.message\nend", `"m"`},
		{"zero division is rescuable", "begin\n 1 / 0\nrescue ZeroDivisionError\n :z\nend", ":z"},
		{"structured Error", `Error("boom", {code: 1}).data[:code]`, "1"},

		// pattern matching
		{"case in array", "case [1, 2]\nin [a, b]\n a + b\nend", "3"},
		{"case in hash", "case {n: 3}\nin {n:}\n n\nend", "3"},
		{"case in guard", "case 5\nin x if x > 3\n :big\nelse\n :small\nend", ":big"},

		// math
		{"min with floats", `min(1.5, 2)`, "1.5"},
		{"max of array", `max([1, 9, 3])`, "9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runInspect(tt.input); got != tt.want {
				t.Errorf("input:\n%s\ngot  %s\nwant %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestSyntaxErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"integer division by zero", `1 / 0`, "division by zero"},
		{"modulo by zero", `1 % 0`, "division by zero"},
		{"string minus string", `"a" - "b"`, "unknown operator"},
		{"boolean addition", `true + true`, "unknown operator"},
		{"index on integer", `5[0]`, "index operator not supported"},
		{"undefined variable", `nope`, "identifier not found"},
		{"undefined method", `1.nope`, "undefined method"},
		{"too few arguments", "def f(a, b)\nend\nf(1)", "wrong number of arguments"},
		{"too many arguments", "def f(a)\nend\nf(1, 2)", "wrong number of arguments"},
		{"super outside class", `super`, "outside of a class"},
		{"super without parent method", "class A\n def t\n super.nope\n end\nend\nA.new.t", "no parent"},
		{"private method call", "class A\n private\n def s\n 1\n end\nend\nA.new.s", "private method"},
		{"frozen array push", "a = [1].freeze\na << 2", "frozen"},
		{"unhandled raise", `raise "boom"`, "boom"},
		{"non-class parent", "x = 1\nclass A < x\nend", "not a class"},
		{"is_a? needs a class", `1.is_a?(3)`, "class or module required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalInputAllowErrors(tt.input)
			msg := got.Inspect()
			if e, ok := got.(*Error); ok {
				msg = e.Message
			} else if ts, ok := got.(*ThrowSignal); ok {
				msg = ts.Inspect()
			} else {
				t.Fatalf("expected an error, got %T (%s)", got, msg)
			}
			if !strings.Contains(strings.ToLower(msg), strings.ToLower(tt.want)) {
				t.Errorf("expected error containing %q, got %q", tt.want, msg)
			}
		})
	}
}

func TestMathModule(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Math.sqrt", "import \"math\"\nMath.sqrt(16)", "4"},
		{"Math.floor", "import \"math\"\nMath.floor(2.7)", "2"},
		{"Math::PI", "import \"math\"\nMath::PI.floor", "3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runInspect(tt.input); got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestRuntimeTypeChecks(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string // Inspect of the result, or error text when wantErr
		wantErr bool
	}{
		{"typed attr_accessor ok", "class P\n attr_accessor age: int\nend\np = P.new\np.age = 5\np.age", "5", false},
		{"typed attr_accessor rejects string", "class P\n attr_accessor age: int\nend\np = P.new\np.age = \"x\"", "type mismatch for property 'age': expected int, got string", true},
		{"typed attr_reader mixed with plain", "class P\n attr_reader :a, b: string\n def initialize\n @a = 1\n @b = \"s\"\n end\nend\nP.new.b", `"s"`, false},
		{"implements satisfied", "interface S\n def area(): float\nend\nclass Q implements S\n def area(): float\n 1.0\n end\nend\nQ.new.area", "1", false},
		{"implements missing method", "interface S\n def area(): float\nend\nclass Q implements S\nend", "class Q does not implement S: missing method 'area'", true},
		{"implements non-interface", "class Q implements Nope\nend", "'Nope' is not an interface", true},
		{"typed block param rejects", "[1, \"a\"].each { |x: int| x }", "expected int, got string", true},
		{"int param ok", "def f(a: int): int\n a + 1\nend\nf(1)", "2", false},
		{"int param rejects string", "def f(a: int): int\n a\nend\nf(\"x\")", "expected int, got string", true},
		{"float param accepts int", "def f(a: float): float\n a\nend\nf(2)", "2", false},
		{"return type mismatch", "def f(): int\n \"s\"\nend\nf()", "must return int, got string", true},
		{"union param", "def f(a: int | string): string\n \"${a}\"\nend\nf(1)", `"1"`, false},
		{"union param rejects float", "def f(a: int | string)\n a\nend\nf(1.5)", "expected int | string", true},
		{"optional param accepts nil", "def f(a: string?)\n a\nend\nf(nil)", "nil", false},
		{"optional param rejects int", "def f(a: string?)\n a\nend\nf(1)", "expected", true},
		{"array element types", "def f(a: int[])\n a.length\nend\nf([1, 2])", "2", false},
		{"array element mismatch", "def f(a: int[])\n a\nend\nf([1, \"x\"])", "expected int[]", true},
		{"keyword param type", "def f(k: int = 1)\n k\nend\nf(k: \"a\")", "expected int", true},
		{"class typed param", "class A\nend\ndef f(a: A)\n 1\nend\nf(A.new)", "1", false},
		{"class typed param mismatch", "class A\nend\ndef f(a: A)\n 1\nend\nf(3)", "expected A", true},
		{"generic param accepts anything", "def f<T>(a: T): T\n a\nend\nf(\"s\")", `"s"`, false},
		{"type alias", "type ID = int | string\ndef f(a: ID)\n a\nend\nf(1)", "1", false},
		{"type alias rejects", "type ID = int | string\ndef f(a: ID)\n a\nend\nf(nil)", "expected ID", true},
		{"interface satisfied", "interface Named\n def name(): string\nend\nclass P\n def name()\n \"p\"\n end\nend\ndef f(a: Named)\n a.name\nend\nf(P.new)", `"p"`, false},
		{"interface violated", "interface Named\n def name(): string\nend\ndef f(a: Named)\n 1\nend\nf(3)", "expected Named", true},
		{"explicit generic constructor", "class Box<T>\n def initialize(v: T)\n @v = v\n end\n def get(): T\n @v\n end\nend\nBox<int>.new(3).get()", "3", false},
		{"typed variable", "x: int = 5\nx", "5", false},
		{"typed variable mismatch", "x: string = 5", "type mismatch", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := evalInputAllowErrors(tt.input)
			if tt.wantErr {
				e, ok := res.(*Error)
				if !ok {
					t.Fatalf("expected error containing %q, got %s", tt.want, res.Inspect())
				}
				if !strings.Contains(e.Message, tt.want) {
					t.Errorf("expected %q in %q", tt.want, e.Message)
				}
				return
			}
			if got := runInspect(tt.input); got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}
