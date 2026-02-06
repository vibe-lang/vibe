package lsp

// BuiltinInfo contains documentation for a builtin function.
type BuiltinInfo struct {
	Signature   string
	MethodStyle string
	Description string
}

var keywords = []string{
	"def", "end", "fn", "let", "const",
	"if", "elsif", "else", "unless", "until",
	"while", "for", "in", "case", "when",
	"break", "continue", "return",
	"class", "struct", "enum", "prop",
	"try", "catch", "throw", "finally",
	"import", "self", "super",
	"true", "false", "nil",
}

var typeNames = []string{
	"int", "float", "string", "boolean", "nil", "any",
}

var builtinNames []string

var builtinDocs = map[string]BuiltinInfo{
	// I/O
	"puts":  {Signature: "puts(value)", Description: "Prints value to stdout with a newline."},
	"print": {Signature: "print(value)", Description: "Prints value to stdout without a newline."},
	"input": {Signature: "input(prompt?)", Description: "Reads a line of input from stdin."},

	// Type conversion
	"to_int":    {Signature: "to_int(value)", MethodStyle: "value.to_int", Description: "Converts a value to an integer."},
	"to_float":  {Signature: "to_float(value)", MethodStyle: "value.to_float", Description: "Converts a value to a float."},
	"to_string": {Signature: "to_string(value)", MethodStyle: "value.to_string", Description: "Converts a value to a string."},
	"type":      {Signature: "type(value)", MethodStyle: "value.type", Description: "Returns the type name of a value."},
	"format":    {Signature: "format(template, args...)", Description: "Sprintf-style string formatting."},

	// JSON
	"to_json":   {Signature: "to_json(value)", MethodStyle: "value.to_json", Description: "Serializes a value to a JSON string."},
	"from_json": {Signature: "from_json(str)", Description: "Parses a JSON string into a Vibe value."},

	// Array functions
	"len":       {Signature: "len(array_or_string)", MethodStyle: "value.len", Description: "Returns the length of an array or string."},
	"push":      {Signature: "push(array, value)", MethodStyle: "array.push(value)", Description: "Appends a value to the end of an array."},
	"pop":       {Signature: "pop(array)", MethodStyle: "array.pop", Description: "Removes and returns the last element."},
	"shift":     {Signature: "shift(array)", MethodStyle: "array.shift", Description: "Removes and returns the first element."},
	"first":     {Signature: "first(array)", MethodStyle: "array.first", Description: "Returns the first element."},
	"last":      {Signature: "last(array)", MethodStyle: "array.last", Description: "Returns the last element."},
	"flatten":   {Signature: "flatten(array)", MethodStyle: "array.flatten", Description: "Flattens a nested array by one level."},
	"includes":  {Signature: "includes(array, value)", MethodStyle: "array.includes(value)", Description: "Returns true if the array contains the value."},
	"index_of":  {Signature: "index_of(array, value)", MethodStyle: "array.index_of(value)", Description: "Returns the index of value, or -1."},
	"remove_at": {Signature: "remove_at(array, index)", Description: "Removes and returns the element at index."},

	// Higher-order array functions
	"map":             {Signature: "map(array, fn)", MethodStyle: "array.map(fn)", Description: "Applies fn to each element, returns new array."},
	"filter":          {Signature: "filter(array, fn)", MethodStyle: "array.filter(fn)", Description: "Returns elements where fn returns true."},
	"each":            {Signature: "each(array, fn)", MethodStyle: "array.each(fn)", Description: "Calls fn for each element."},
	"reduce":          {Signature: "reduce(array, initial, fn)", MethodStyle: "array.reduce(initial, fn)", Description: "Reduces array to single value."},
	"find":            {Signature: "find(array, fn)", MethodStyle: "array.find(fn)", Description: "Returns first element where fn is true."},
	"any":             {Signature: "any(array, fn)", MethodStyle: "array.any(fn)", Description: "Returns true if fn is true for any element."},
	"all":             {Signature: "all(array, fn)", MethodStyle: "array.all(fn)", Description: "Returns true if fn is true for all elements."},
	"flat_map":        {Signature: "flat_map(array, fn)", MethodStyle: "array.flat_map(fn)", Description: "Maps and flattens result by one level."},
	"zip":             {Signature: "zip(a, b)", MethodStyle: "a.zip(b)", Description: "Combines two arrays into pairs."},
	"take":            {Signature: "take(array, n)", MethodStyle: "array.take(n)", Description: "Returns the first n elements."},
	"drop":            {Signature: "drop(array, n)", MethodStyle: "array.drop(n)", Description: "Returns elements after first n."},
	"chunk":           {Signature: "chunk(array, size)", MethodStyle: "array.chunk(size)", Description: "Splits array into chunks."},
	"group_by":        {Signature: "group_by(array, fn)", MethodStyle: "array.group_by(fn)", Description: "Groups elements by fn result."},
	"sort_by":         {Signature: "sort_by(array, fn)", MethodStyle: "array.sort_by(fn)", Description: "Sorts by fn result."},
	"min_by":          {Signature: "min_by(array, fn)", MethodStyle: "array.min_by(fn)", Description: "Element with minimum fn value."},
	"max_by":          {Signature: "max_by(array, fn)", MethodStyle: "array.max_by(fn)", Description: "Element with maximum fn value."},
	"each_with_index": {Signature: "each_with_index(array, fn)", MethodStyle: "array.each_with_index(fn)", Description: "Calls fn(element, index)."},
	"map_with_index":  {Signature: "map_with_index(array, fn)", MethodStyle: "array.map_with_index(fn)", Description: "Maps fn(element, index)."},
	"tally":           {Signature: "tally(array)", MethodStyle: "array.tally", Description: "Counts occurrences, returns hash."},
	"sort":            {Signature: "sort(array)", MethodStyle: "array.sort", Description: "Returns sorted copy."},
	"reverse":         {Signature: "reverse(array)", MethodStyle: "array.reverse", Description: "Returns reversed copy."},

	// String functions
	"trim":            {Signature: "trim(string)", MethodStyle: "string.trim", Description: "Removes leading/trailing whitespace."},
	"split":           {Signature: "split(string, delimiter)", MethodStyle: "string.split(delim)", Description: "Splits string into array."},
	"join":            {Signature: "join(array, separator)", MethodStyle: "array.join(sep)", Description: "Joins array into string."},
	"replace":         {Signature: "replace(string, old, new)", MethodStyle: "string.replace(old, new)", Description: "Replaces all occurrences."},
	"upcase":          {Signature: "upcase(string)", MethodStyle: "string.upcase", Description: "Converts to uppercase."},
	"downcase":        {Signature: "downcase(string)", MethodStyle: "string.downcase", Description: "Converts to lowercase."},
	"capitalize":      {Signature: "capitalize(string)", MethodStyle: "string.capitalize", Description: "Capitalizes first character."},
	"starts_with":     {Signature: "starts_with(string, prefix)", MethodStyle: "string.starts_with(prefix)", Description: "True if string starts with prefix."},
	"ends_with":       {Signature: "ends_with(string, suffix)", MethodStyle: "string.ends_with(suffix)", Description: "True if string ends with suffix."},
	"repeat":          {Signature: "repeat(string, count)", MethodStyle: "string.repeat(n)", Description: "Repeats string n times."},
	"chars":           {Signature: "chars(string)", MethodStyle: "string.chars", Description: "Splits into array of characters."},
	"pad_start":       {Signature: "pad_start(string, length, char?)", MethodStyle: "string.pad_start(n, ch)", Description: "Pads from the start."},
	"pad_end":         {Signature: "pad_end(string, length, char?)", MethodStyle: "string.pad_end(n, ch)", Description: "Pads from the end."},
	"string_reverse":  {Signature: "string_reverse(string)", MethodStyle: "string.string_reverse", Description: "Reverses a string."},
	"string_slice":    {Signature: "string_slice(string, start, end)", MethodStyle: "string.string_slice(s, e)", Description: "Returns substring."},
	"string_contains": {Signature: "string_contains(string, sub)", MethodStyle: "string.string_contains(sub)", Description: "True if string contains sub."},
	"index_of_string": {Signature: "index_of_string(string, sub)", MethodStyle: "string.index_of_string(sub)", Description: "Index of substring, or -1."},

	// Hash functions
	"keys":    {Signature: "keys(hash)", MethodStyle: "hash.keys", Description: "Returns array of all keys."},
	"values":  {Signature: "values(hash)", MethodStyle: "hash.values", Description: "Returns array of all values."},
	"has_key": {Signature: "has_key(hash, key)", MethodStyle: "hash.has_key(key)", Description: "True if hash contains key."},
	"merge":   {Signature: "merge(hash1, hash2)", MethodStyle: "hash1.merge(hash2)", Description: "Merges two hashes."},
	"delete":  {Signature: "delete(hash, key)", Description: "Removes key, returns value."},

	// File I/O
	"read_file":   {Signature: "read_file(path)", Description: "Reads file contents as string."},
	"write_file":  {Signature: "write_file(path, content)", Description: "Writes string to file."},
	"file_exists": {Signature: "file_exists(path)", Description: "True if file exists."},

	// Errors
	"Error": {Signature: "Error(message, data?)", Description: "Creates a structured error with .message, .data, .type fields."},
}

var keywordDocs = map[string]string{
	"def":      "`def name(params)` -- Define a function. Close with `end`.",
	"fn":       "`fn(params) { body }` -- Anonymous function expression.",
	"class":    "`class Name` or `class Name < Parent` -- Define a class. Close with `end`.",
	"struct":   "`struct Name` -- Define a struct type. Close with `end`.",
	"enum":     "`enum Name` -- Define an enumeration. Close with `end`.",
	"if":       "`if condition` -- Conditional block. Close with `end`.",
	"unless":   "`unless condition` -- Inverse conditional. Close with `end`.",
	"for":      "`for item in collection` -- Iterate over array, range, or hash. Close with `end`.",
	"while":    "`while condition` -- Loop while true. Close with `end`.",
	"until":    "`until condition` -- Loop until true. Close with `end`.",
	"case":     "`case subject` / `when value` -- Pattern matching. Close with `end`.",
	"try":      "`try` / `catch e` / `finally` -- Error handling. Close with `end`.",
	"import":   "`import \"module\"` -- Import a stdlib module or file.",
	"return":   "`return value` -- Return from current function.",
	"throw":    "`throw value` -- Throw an error (string or Error object).",
	"const":    "`const NAME = value` -- Declare an immutable constant.",
	"let":      "`let name = value` -- Declare a variable (optional keyword).",
	"self":     "Reference to the current class instance.",
	"super":    "Reference to the parent class. Use `super(args)` or `super.method()`.",
	"nil":      "The absence of a value.",
	"true":     "Boolean true.",
	"false":    "Boolean false.",
	"in":       "Used in `for x in collection` and `value in collection` (containment test).",
	"end":      "Closes a block (`def`, `class`, `if`, `for`, `while`, etc.).",
	"break":    "Exits the current loop.",
	"continue": "Skips to the next iteration of the current loop.",
	"prop":     "`prop name: type` -- Declare a property inside a class.",
}

func init() {
	builtinNames = make([]string, 0, len(builtinDocs))
	for name := range builtinDocs {
		builtinNames = append(builtinNames, name)
	}
}
