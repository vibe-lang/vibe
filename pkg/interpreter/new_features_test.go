package interpreter

// Tests for all new language features added in v0.2.0.

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// Logical operators && and ||
// ---------------------------------------------------------------------------

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"true && true", "true && true", true},
		{"true && false", "true && false", false},
		{"false && true", "false && true", false},
		{"false && false", "false && false", false},
		{"true || true", "true || true", true},
		{"true || false", "true || false", true},
		{"false || true", "false || true", true},
		{"false || false", "false || false", false},
		{"compound and", "5 > 3 && 10 > 7", true},
		{"compound or", "5 > 3 || 10 < 7", true},
		{"and with comparison", "x = 5\nx > 0 && x < 10", true},
		{"or short circuit", "true || false", true},
		{"and short circuit false", "false && true", false},
		{"chained conditions", "true && true && true", true},
		{"mixed and or", "true || false && false", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertBoolean(t, result, tt.expected)
		})
	}
}

func TestLogicalOperatorsInIf(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			"if with and",
			`x = 5
			if x > 0 && x < 10
				42
			else
				0
			end`,
			42,
		},
		{
			"if with or",
			`x = 15
			if x < 0 || x > 10
				42
			else
				0
			end`,
			42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertInteger(t, result, tt.expected)
		})
	}
}

// ---------------------------------------------------------------------------
// While loops
// ---------------------------------------------------------------------------

func TestWhileLoop(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			"basic while loop",
			`x = 0
			while x < 5
				x = x + 1
			end
			x`,
			5,
		},
		{
			"while with accumulator",
			`sum = 0
			i = 1
			while i <= 10
				sum = sum + i
				i = i + 1
			end
			sum`,
			55,
		},
		{
			"while never enters",
			`x = 10
			while x < 5
				x = x + 1
			end
			x`,
			10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertInteger(t, result, tt.expected)
		})
	}
}

// ---------------------------------------------------------------------------
// Break and Continue
// ---------------------------------------------------------------------------

func TestBreak(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			"break in while loop",
			`x = 0
			while true
				if x == 5
					break
				end
				x = x + 1
			end
			x`,
			5,
		},
		{
			"break in for loop",
			`result = 0
			for i in [1, 2, 3, 4, 5]
				if i == 3
					break
				end
				result = result + i
			end
			result`,
			3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertInteger(t, result, tt.expected)
		})
	}
}

func TestContinue(t *testing.T) {
	t.Run("continue in for loop skips odds", func(t *testing.T) {
		result := evalInput(t, `
		sum = 0
		for i in [1, 2, 3, 4, 5, 6]
			if i % 2 != 0
				continue
			end
			sum = sum + i
		end
		sum
		`)
		assertInteger(t, result, 12)
	})

	t.Run("continue in while loop", func(t *testing.T) {
		result := evalInput(t, `
		sum = 0
		i = 0
		while i < 10
			i = i + 1
			if i % 2 != 0
				continue
			end
			sum = sum + i
		end
		sum
		`)
		assertInteger(t, result, 30)
	})
}

// ---------------------------------------------------------------------------
// Hash maps
// ---------------------------------------------------------------------------

func TestHashLiteral(t *testing.T) {
	t.Run("create and access hash", func(t *testing.T) {
		result := evalInput(t, `
		h = {"name": "Alice", "age": 30}
		h["name"]
		`)
		assertString(t, result, "Alice")
	})

	t.Run("hash with integer values", func(t *testing.T) {
		result := evalInput(t, `
		h = {"x": 10, "y": 20}
		h["x"] + h["y"]
		`)
		assertInteger(t, result, 30)
	})

	t.Run("hash missing key returns nil", func(t *testing.T) {
		result := evalInput(t, `
		h = {"a": 1}
		h["missing"]
		`)
		assertNil(t, result)
	})

	t.Run("empty hash", func(t *testing.T) {
		result := evalInput(t, `
		h = {}
		len(h)
		`)
		assertInteger(t, result, 0)
	})

	t.Run("hash mutation", func(t *testing.T) {
		result := evalInput(t, `
		h = {"x": 1}
		h["x"] = 42
		h["x"]
		`)
		assertInteger(t, result, 42)
	})

	t.Run("hash add new key", func(t *testing.T) {
		result := evalInput(t, `
		h = {}
		h["name"] = "Bob"
		h["name"]
		`)
		assertString(t, result, "Bob")
	})
}

// ---------------------------------------------------------------------------
// Array mutation
// ---------------------------------------------------------------------------

func TestArrayMutation(t *testing.T) {
	t.Run("set array element", func(t *testing.T) {
		result := evalInput(t, `
		arr = [1, 2, 3]
		arr[1] = 42
		arr[1]
		`)
		assertInteger(t, result, 42)
	})

	t.Run("mutate in loop", func(t *testing.T) {
		result := evalInput(t, `
		arr = [1, 2, 3, 4, 5]
		i = 0
		while i < len(arr)
			arr[i] = arr[i] * 2
			i = i + 1
		end
		arr[2]
		`)
		assertInteger(t, result, 6)
	})
}

// ---------------------------------------------------------------------------
// Struct mutation
// ---------------------------------------------------------------------------

func TestStructMutation(t *testing.T) {
	t.Run("set struct field", func(t *testing.T) {
		result := evalInput(t, `
		struct Point
			x: int
			y: int
		end

		p = Point(x: 1, y: 2)
		p.x = 10
		p.x
		`)
		assertInteger(t, result, 10)
	})

	t.Run("modify struct in place", func(t *testing.T) {
		result := evalInput(t, `
		struct Counter
			count: int
		end

		c = Counter(count: 0)
		c.count = c.count + 1
		c.count = c.count + 1
		c.count = c.count + 1
		c.count
		`)
		assertInteger(t, result, 3)
	})
}

// ---------------------------------------------------------------------------
// Builtins
// ---------------------------------------------------------------------------

func TestBuiltinLen(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"array length", "len([1, 2, 3])", 3},
		{"empty array", "len([])", 0},
		{"string length", `len("hello")`, 5},
		{"empty string", `len("")`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertInteger(t, result, tt.expected)
		})
	}
}

func TestBuiltinType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"type(5)", "INTEGER"},
		{"type(3.14)", "FLOAT"},
		{`type("hello")`, "STRING"},
		{"type(true)", "BOOLEAN"},
		{"type(nil)", "NIL"},
		{"type([1,2])", "ARRAY"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertString(t, result, tt.expected)
		})
	}
}

func TestBuiltinPushFirstLastRest(t *testing.T) {
	t.Run("push", func(t *testing.T) {
		result := evalInput(t, `
		arr = push([1, 2], 3)
		len(arr)
		`)
		assertInteger(t, result, 3)
	})

	t.Run("first", func(t *testing.T) {
		result := evalInput(t, `first([10, 20, 30])`)
		assertInteger(t, result, 10)
	})

	t.Run("first empty", func(t *testing.T) {
		result := evalInput(t, `first([])`)
		assertNil(t, result)
	})

	t.Run("last", func(t *testing.T) {
		result := evalInput(t, `last([10, 20, 30])`)
		assertInteger(t, result, 30)
	})

	t.Run("rest", func(t *testing.T) {
		result := evalInput(t, `len(rest([1, 2, 3]))`)
		assertInteger(t, result, 2)
	})
}

func TestBuiltinConversions(t *testing.T) {
	t.Run("to_s integer", func(t *testing.T) {
		result := evalInput(t, `to_s(42)`)
		assertString(t, result, "42")
	})

	t.Run("to_i string", func(t *testing.T) {
		result := evalInput(t, `to_i("123")`)
		assertInteger(t, result, 123)
	})

	t.Run("to_f string", func(t *testing.T) {
		result := evalInput(t, `to_f("3.14")`)
		assertFloat(t, result, 3.14)
	})
}

func TestBuiltinStringOps(t *testing.T) {
	t.Run("split", func(t *testing.T) {
		result := evalInput(t, `len(split("a,b,c", ","))`)
		assertInteger(t, result, 3)
	})

	t.Run("join", func(t *testing.T) {
		result := evalInput(t, `join(["a", "b", "c"], "-")`)
		assertString(t, result, "a-b-c")
	})

	t.Run("trim", func(t *testing.T) {
		result := evalInput(t, `trim("  hello  ")`)
		assertString(t, result, "hello")
	})

	t.Run("replace", func(t *testing.T) {
		result := evalInput(t, `replace("hello world", "world", "vibe")`)
		assertString(t, result, "hello vibe")
	})

	t.Run("contains string", func(t *testing.T) {
		result := evalInput(t, `contains("hello world", "world")`)
		assertBoolean(t, result, true)
	})

	t.Run("string_length", func(t *testing.T) {
		result := evalInput(t, `string_length("hello")`)
		assertInteger(t, result, 5)
	})
}

func TestBuiltinArrayOps(t *testing.T) {
	t.Run("sort", func(t *testing.T) {
		result := evalInput(t, `
		arr = sort([3, 1, 2])
		first(arr)
		`)
		assertInteger(t, result, 1)
	})

	t.Run("reverse", func(t *testing.T) {
		result := evalInput(t, `
		arr = reverse([1, 2, 3])
		first(arr)
		`)
		assertInteger(t, result, 3)
	})

	t.Run("contains array", func(t *testing.T) {
		result := evalInput(t, `contains([1, 2, 3], 2)`)
		assertBoolean(t, result, true)
	})

	t.Run("contains array false", func(t *testing.T) {
		result := evalInput(t, `contains([1, 2, 3], 5)`)
		assertBoolean(t, result, false)
	})

	t.Run("abs", func(t *testing.T) {
		result := evalInput(t, `abs(-42)`)
		assertInteger(t, result, 42)
	})

	t.Run("min", func(t *testing.T) {
		result := evalInput(t, `min(3, 7)`)
		assertInteger(t, result, 3)
	})

	t.Run("max", func(t *testing.T) {
		result := evalInput(t, `max(3, 7)`)
		assertInteger(t, result, 7)
	})
}

func TestBuiltinMapFilter(t *testing.T) {
	t.Run("map with function", func(t *testing.T) {
		result := evalInput(t, `
		def double(x: int): int
			x * 2
		end
		arr = map([1, 2, 3], double)
		arr[1]
		`)
		assertInteger(t, result, 4)
	})

	t.Run("filter with function", func(t *testing.T) {
		result := evalInput(t, `
		def is_even(x: int): int
			x % 2 == 0
		end
		arr = filter([1, 2, 3, 4, 5, 6], is_even)
		len(arr)
		`)
		assertInteger(t, result, 3)
	})
}

func TestBuiltinKeysValues(t *testing.T) {
	t.Run("keys", func(t *testing.T) {
		result := evalInput(t, `
		h = {"a": 1, "b": 2}
		len(keys(h))
		`)
		assertInteger(t, result, 2)
	})

	t.Run("values", func(t *testing.T) {
		result := evalInput(t, `
		h = {"a": 1, "b": 2}
		len(values(h))
		`)
		assertInteger(t, result, 2)
	})
}

// ---------------------------------------------------------------------------
// Try / Catch / Throw
// ---------------------------------------------------------------------------

func TestTryCatch(t *testing.T) {
	t.Run("catch thrown value", func(t *testing.T) {
		result := evalInput(t, `
		result = try
			throw "something went wrong"
		catch e
			e
		end
		result
		`)
		assertString(t, result, "something went wrong")
	})

	t.Run("no throw passes through", func(t *testing.T) {
		result := evalInput(t, `
		try
			42
		catch e
			0
		end
		`)
		assertInteger(t, result, 42)
	})

	t.Run("catch runtime error", func(t *testing.T) {
		result := evalInput(t, `
		try
			10 / 0
		catch e
			"caught"
		end
		`)
		assertString(t, result, "caught")
	})
}

// ---------------------------------------------------------------------------
// Import
// ---------------------------------------------------------------------------

func TestImport(t *testing.T) {
	// Create a temp file to import
	tmpDir := t.TempDir()
	libPath := filepath.Join(tmpDir, "lib.vb")
	err := os.WriteFile(libPath, []byte(`
def add(a: int, b: int): int
	a + b
end
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("import file and use function", func(t *testing.T) {
		input := `import "` + libPath + `"
		add(3, 4)
		`
		result := evalInput(t, input)
		assertInteger(t, result, 7)
	})
}

// ---------------------------------------------------------------------------
// Integration: combine new features
// ---------------------------------------------------------------------------

func TestNewFeaturesIntegration(t *testing.T) {
	t.Run("fizzbuzz with while and logical ops", func(t *testing.T) {
		result := evalInput(t, `
		count = 0
		i = 1
		while i <= 30
			if i % 3 == 0 && i % 5 == 0
				count = count + 1
			end
			i = i + 1
		end
		count
		`)
		assertInteger(t, result, 2) // 15 and 30
	})

	t.Run("hash as accumulator", func(t *testing.T) {
		result := evalInput(t, `
		counts = {}
		words = ["hello", "world", "hello", "vibe", "world", "hello"]
		for word in words
			if contains(counts, word)
				counts[word] = counts[word] + 1
			else
				counts[word] = 1
			end
		end
		counts["hello"]
		`)
		assertInteger(t, result, 3)
	})

	t.Run("bubble sort with while and mutation", func(t *testing.T) {
		result := evalInput(t, `
		arr = [5, 3, 1, 4, 2]
		n = len(arr)
		i = 0
		while i < n
			j = 0
			while j < n - 1 - i
				if arr[j] > arr[j + 1]
					temp = arr[j]
					arr[j] = arr[j + 1]
					arr[j + 1] = temp
				end
				j = j + 1
			end
			i = i + 1
		end
		arr[0]
		`)
		assertInteger(t, result, 1)
	})
}

// ---------------------------------------------------------------------------
// Method-style calls: obj.method(args) -> method(obj, args)
// ---------------------------------------------------------------------------

func TestMethodStyleCalls(t *testing.T) {
	// String methods
	t.Run("string.split", func(t *testing.T) {
		result := evalInput(t, `
		parts = "hello world".split(" ")
		parts[0]
		`)
		assertString(t, result, "hello")
	})

	t.Run("string.replace", func(t *testing.T) {
		result := evalInput(t, `"hello world".replace("world", "vibe")`)
		assertString(t, result, "hello vibe")
	})

	t.Run("string.trim", func(t *testing.T) {
		result := evalInput(t, `"  hello  ".trim()`)
		assertString(t, result, "hello")
	})

	t.Run("string.contains", func(t *testing.T) {
		result := evalInput(t, `"hello world".contains("world")`)
		assertBoolean(t, result, true)
	})

	t.Run("string.len", func(t *testing.T) {
		result := evalInput(t, `"hello".len()`)
		assertInteger(t, result, 5)
	})

	// Array methods
	t.Run("array.sort", func(t *testing.T) {
		result := evalInput(t, `
		arr = [3, 1, 2].sort()
		arr[0]
		`)
		assertInteger(t, result, 1)
	})

	t.Run("array.reverse", func(t *testing.T) {
		result := evalInput(t, `
		arr = [1, 2, 3].reverse()
		arr[0]
		`)
		assertInteger(t, result, 3)
	})

	t.Run("array.contains", func(t *testing.T) {
		result := evalInput(t, `[1, 2, 3].contains(2)`)
		assertBoolean(t, result, true)
	})

	t.Run("array.first", func(t *testing.T) {
		result := evalInput(t, `[10, 20, 30].first()`)
		assertInteger(t, result, 10)
	})

	t.Run("array.last", func(t *testing.T) {
		result := evalInput(t, `[10, 20, 30].last()`)
		assertInteger(t, result, 30)
	})

	t.Run("array.len", func(t *testing.T) {
		result := evalInput(t, `[1, 2, 3, 4, 5].len()`)
		assertInteger(t, result, 5)
	})

	t.Run("array.map", func(t *testing.T) {
		result := evalInput(t, `
		def double(x: int): int
			x * 2
		end
		arr = [1, 2, 3]
		doubled = arr.map(double)
		doubled[1]
		`)
		assertInteger(t, result, 4)
	})

	t.Run("array.filter", func(t *testing.T) {
		result := evalInput(t, `
		def gt2(x: int): boolean
			x > 2
		end
		arr = [1, 2, 3, 4, 5]
		arr.filter(gt2).len()
		`)
		assertInteger(t, result, 3)
	})

	// Hash methods
	t.Run("hash.keys", func(t *testing.T) {
		result := evalInput(t, `
		h = {"a": 1, "b": 2}
		h.keys().len()
		`)
		assertInteger(t, result, 2)
	})

	t.Run("hash.values", func(t *testing.T) {
		result := evalInput(t, `
		h = {"a": 1, "b": 2}
		h.values().len()
		`)
		assertInteger(t, result, 2)
	})

	// Chaining
	t.Run("method chaining", func(t *testing.T) {
		result := evalInput(t, `
		parts = "  Hello World  ".trim().split(" ")
		parts[0]
		`)
		assertString(t, result, "Hello")
	})

	// Struct field access still works
	t.Run("struct field access preserved", func(t *testing.T) {
		result := evalInput(t, `
		struct Point
			x: int
			y: int
		end
		p = Point(x: 10, y: 20)
		p.x + p.y
		`)
		assertInteger(t, result, 30)
	})

	// Type conversion method style
	t.Run("to_s method style", func(t *testing.T) {
		result := evalInput(t, `42.to_s()`)
		assertString(t, result, "42")
	})
}

// ---------------------------------------------------------------------------
// JSON parse and encode
// ---------------------------------------------------------------------------

func TestJSONParse(t *testing.T) {
	t.Run("parse object", func(t *testing.T) {
		result := evalInput(t, `
		data = json_parse('{"name": "Alice", "age": 30}')
		data["name"]
		`)
		assertString(t, result, "Alice")
	})

	t.Run("parse object age as int", func(t *testing.T) {
		result := evalInput(t, `
		data = json_parse('{"age": 30}')
		data["age"]
		`)
		assertInteger(t, result, 30)
	})

	t.Run("parse array", func(t *testing.T) {
		result := evalInput(t, `
		data = json_parse('[1, 2, 3]')
		data[1]
		`)
		assertInteger(t, result, 2)
	})

	t.Run("parse nested", func(t *testing.T) {
		result := evalInput(t, `
		data = json_parse('{"user": {"name": "Bob"}}')
		data["user"]["name"]
		`)
		assertString(t, result, "Bob")
	})

	t.Run("parse boolean", func(t *testing.T) {
		result := evalInput(t, `
		data = json_parse('{"active": true}')
		data["active"]
		`)
		assertBoolean(t, result, true)
	})

	t.Run("parse null", func(t *testing.T) {
		result := evalInput(t, `
		data = json_parse('{"value": null}')
		data["value"].type()
		`)
		assertString(t, result, "NIL")
	})

	t.Run("parse float", func(t *testing.T) {
		result := evalInput(t, `
		data = json_parse('{"pi": 3.14}')
		data["pi"].to_s()
		`)
		assertString(t, result, "3.14")
	})
}

func TestJSONEncode(t *testing.T) {
	t.Run("encode hash", func(t *testing.T) {
		result := evalInput(t, `
		json_encode({"name": "Alice"})
		`)
		assertString(t, result, `{"name":"Alice"}`)
	})

	t.Run("encode array", func(t *testing.T) {
		result := evalInput(t, `
		json_encode([1, 2, 3])
		`)
		assertString(t, result, `[1,2,3]`)
	})

	t.Run("encode string", func(t *testing.T) {
		result := evalInput(t, `
		json_encode("hello")
		`)
		assertString(t, result, `"hello"`)
	})

	t.Run("encode roundtrip", func(t *testing.T) {
		result := evalInput(t, `
		original = {"name": "Alice", "age": 30}
		encoded = json_encode(original)
		decoded = json_parse(encoded)
		decoded["name"]
		`)
		assertString(t, result, "Alice")
	})
}

func TestJSONMethodStyle(t *testing.T) {
	t.Run("string.json_parse()", func(t *testing.T) {
		result := evalInput(t, `
		data = '{"x": 42}'.json_parse()
		data["x"]
		`)
		assertInteger(t, result, 42)
	})

	t.Run("string.json() alias", func(t *testing.T) {
		result := evalInput(t, `
		data = '{"x": 42}'.json()
		data["x"]
		`)
		assertInteger(t, result, 42)
	})
}

// ---------------------------------------------------------------------------
// Hash dot access
// ---------------------------------------------------------------------------

func TestHashDotAccess(t *testing.T) {
	t.Run("read key via dot", func(t *testing.T) {
		result := evalInput(t, `
		h = {"name": "Alice", "age": 30}
		h.name
		`)
		assertString(t, result, "Alice")
	})

	t.Run("read nested hash via dot", func(t *testing.T) {
		result := evalInput(t, `
		h = {"user": {"name": "Bob"}}
		h.user.name
		`)
		assertString(t, result, "Bob")
	})

	t.Run("dot assignment on hash", func(t *testing.T) {
		result := evalInput(t, `
		h = {"name": "Alice"}
		h.name = "Bob"
		h.name
		`)
		assertString(t, result, "Bob")
	})

	t.Run("dot assignment adds new key", func(t *testing.T) {
		result := evalInput(t, `
		h = {"name": "Alice"}
		h.age = 30
		h.age
		`)
		assertInteger(t, result, 30)
	})

	t.Run("bare key hash with dot access", func(t *testing.T) {
		result := evalInput(t, `
		config = {host: "localhost", port: 8080}
		config.host
		`)
		assertString(t, result, "localhost")
	})

	t.Run("hash dot access with method fallback", func(t *testing.T) {
		result := evalInput(t, `
		h = {"a": 1, "b": 2}
		h.len
		`)
		assertInteger(t, result, 2)
	})
}

// ---------------------------------------------------------------------------
// Stdlib imports
// ---------------------------------------------------------------------------

func TestStdlibImportRequired(t *testing.T) {
	t.Run("fetch not available without import", func(t *testing.T) {
		result := evalInput(t, `fetch`)
		if result == nil || result.Type() != ERROR_OBJ {
			t.Fatalf("expected error when using fetch without import, got %v", result)
		}
	})

	t.Run("Request not available without import", func(t *testing.T) {
		result := evalInput(t, `Request`)
		if result == nil || result.Type() != ERROR_OBJ {
			t.Fatalf("expected error when using Request without import, got %v", result)
		}
	})

	t.Run("import net/http makes fetch available", func(t *testing.T) {
		result := evalInput(t, `
		import "net/http"
		type(fetch)
		`)
		assertString(t, result, "BUILTIN")
	})

	t.Run("import net/http makes Request available", func(t *testing.T) {
		result := evalInput(t, `
		import "net/http"
		type(Request)
		`)
		assertString(t, result, "STRUCT")
	})

	t.Run("import net/http makes Response available", func(t *testing.T) {
		result := evalInput(t, `
		import "net/http"
		type(Response)
		`)
		assertString(t, result, "STRUCT")
	})
}

func TestStdlibRequestStruct(t *testing.T) {
	t.Run("create Request with defaults", func(t *testing.T) {
		result := evalInput(t, `
		import "net/http"
		req = Request(url: "https://example.com")
		req.method
		`)
		assertString(t, result, "GET")
	})

	t.Run("create Request with method", func(t *testing.T) {
		result := evalInput(t, `
		import "net/http"
		req = Request(url: "https://example.com", method: "POST")
		req.method
		`)
		assertString(t, result, "POST")
	})

	t.Run("mutate Request fields", func(t *testing.T) {
		result := evalInput(t, `
		import "net/http"
		req = Request(url: "https://example.com")
		req.body = "hello"
		req.body
		`)
		assertString(t, result, "hello")
	})
}
