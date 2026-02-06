package interpreter

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// stdlibModules maps import paths to module loader functions.
// Each loader injects structs, functions, and constants into the environment.
var stdlibModules map[string]func(env *Environment)

func init() {
	stdlibModules = map[string]func(env *Environment){
		"net/http":   loadNetHTTP,
		"math":       loadMath,
		"regex":      loadRegex,
		"concurrent": loadConcurrent,
		"os":         loadOS,
		"time":       loadTime,
	}
}

// LoadStdlib attempts to load a standard library module by name.
// Returns true if the module was found and loaded, false otherwise.
func LoadStdlib(name string, env *Environment) bool {
	loader, ok := stdlibModules[name]
	if !ok {
		return false
	}
	loader(env)
	return true
}

// ---------------------------------------------------------------------------
// net/http — HTTP client with fetch, Request, and Response
// ---------------------------------------------------------------------------

// Request struct definition (shared so fetch can create instances)
var requestStruct = &Struct{
	Name: "Request",
	Fields: map[string]Object{
		"url":     &String{Value: ""},
		"method":  &String{Value: "GET"},
		"headers": &Hash{Pairs: map[string]Object{}, Order: []string{}},
		"body":    &String{Value: ""},
	},
	DefaultValues: map[string]Object{
		"url":     &String{Value: ""},
		"method":  &String{Value: "GET"},
		"headers": &Hash{Pairs: map[string]Object{}, Order: []string{}},
		"body":    &String{Value: ""},
	},
}

// Response struct definition (shared so fetch can create instances)
var responseStruct = &Struct{
	Name: "Response",
	Fields: map[string]Object{
		"status":  &Integer{Value: 0},
		"body":    &String{Value: ""},
		"headers": &Hash{Pairs: map[string]Object{}, Order: []string{}},
		"ok":      &Boolean{Value: false},
	},
	DefaultValues: map[string]Object{
		"status":  &Integer{Value: 0},
		"body":    &String{Value: ""},
		"headers": &Hash{Pairs: map[string]Object{}, Order: []string{}},
		"ok":      &Boolean{Value: false},
	},
}

func loadNetHTTP(env *Environment) {
	// Register struct definitions so Request(...) and Response(...) constructors work
	env.Set("Request", requestStruct)
	env.Set("Response", responseStruct)

	// Register fetch function
	env.Set("fetch", &Builtin{Fn: stdlibFetch})

	// Override json to also handle Response structs: response.json() parses body
	env.Set("json", &Builtin{Fn: func(args ...Object) Object {
		if len(args) == 1 {
			// If it's a Response struct, extract the body and parse it
			if si, ok := args[0].(*StructInstance); ok && si.Struct.Name == "Response" {
				if body, ok := si.Fields["body"].(*String); ok {
					return builtinJSONParse(body)
				}
			}
		}
		// Fall back to normal json_parse behavior
		return builtinJSONParse(args...)
	}})
}

func stdlibFetch(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return newError("fetch: expected 1-2 arguments, got %d", len(args))
	}

	var url, method, body string
	var headers *Hash
	method = "GET"
	headers = &Hash{Pairs: map[string]Object{}, Order: []string{}}

	switch first := args[0].(type) {
	case *StructInstance:
		// fetch(Request(...))
		if first.Struct.Name != "Request" {
			return newError("fetch: expected a string or Request, got struct %s", first.Struct.Name)
		}
		if u, ok := first.Fields["url"]; ok {
			url = u.Inspect()
		}
		if m, ok := first.Fields["method"]; ok {
			method = strings.ToUpper(m.Inspect())
		}
		if b, ok := first.Fields["body"]; ok {
			body = b.Inspect()
		}
		if h, ok := first.Fields["headers"].(*Hash); ok {
			headers = h
		}
	case *String:
		// fetch("url") or fetch("url", {options})
		url = first.Value
		if len(args) == 2 {
			opts, ok := args[1].(*Hash)
			if !ok {
				return newError("fetch: second argument must be a hash, got %s", args[1].Type())
			}
			if m, exists := opts.Pairs["method"]; exists {
				method = strings.ToUpper(m.Inspect())
			}
			if b, exists := opts.Pairs["body"]; exists {
				body = b.Inspect()
			}
			if h, exists := opts.Pairs["headers"]; exists {
				if hh, ok := h.(*Hash); ok {
					headers = hh
				}
			}
		}
	default:
		return newError("fetch: first argument must be a string or Request, got %s", args[0].Type())
	}

	// Build Go HTTP request
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	httpReq, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return newError("fetch: %s", err.Error())
	}

	for k, v := range headers.Pairs {
		httpReq.Header.Set(k, v.Inspect())
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return newError("fetch: %s", err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return newError("fetch: error reading response: %s", err.Error())
	}

	// Build response headers hash
	respHeaders := &Hash{Pairs: map[string]Object{}, Order: []string{}}
	for k, vals := range resp.Header {
		lk := strings.ToLower(k)
		respHeaders.Pairs[lk] = &String{Value: strings.Join(vals, ", ")}
		respHeaders.Order = append(respHeaders.Order, lk)
	}

	// Return a Response struct instance
	return &StructInstance{
		Struct: responseStruct,
		Fields: map[string]Object{
			"status":  &Integer{Value: int64(resp.StatusCode)},
			"body":    &String{Value: string(respBody)},
			"headers": respHeaders,
			"ok":      &Boolean{Value: resp.StatusCode >= 200 && resp.StatusCode < 300},
		},
	}
}

// ---------------------------------------------------------------------------
// math — Math functions and constants
// ---------------------------------------------------------------------------

func loadMath(env *Environment) {
	// Create a Math "namespace" as a hash with function values
	mathNs := &Hash{
		Pairs: map[string]Object{
			"PI":  &Float{Value: math.Pi},
			"E":   &Float{Value: math.E},
			"INF": &Float{Value: math.Inf(1)},
		},
		Order: []string{"PI", "E", "INF"},
	}
	env.Set("Math", mathNs)

	// Register math functions as builtins that can also be called via Math.sqrt() etc.
	env.Set("sqrt", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("sqrt: expected 1 argument, got %d", len(args))
		}
		val := toFloat64(args[0])
		if val == nil {
			return newError("sqrt: argument must be a number, got %s", args[0].Type())
		}
		return &Float{Value: math.Sqrt(*val)}
	}})

	env.Set("floor", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("floor: expected 1 argument, got %d", len(args))
		}
		val := toFloat64(args[0])
		if val == nil {
			return newError("floor: argument must be a number, got %s", args[0].Type())
		}
		return &Integer{Value: int64(math.Floor(*val))}
	}})

	env.Set("ceil", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("ceil: expected 1 argument, got %d", len(args))
		}
		val := toFloat64(args[0])
		if val == nil {
			return newError("ceil: argument must be a number, got %s", args[0].Type())
		}
		return &Integer{Value: int64(math.Ceil(*val))}
	}})

	env.Set("round", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("round: expected 1 argument, got %d", len(args))
		}
		val := toFloat64(args[0])
		if val == nil {
			return newError("round: argument must be a number, got %s", args[0].Type())
		}
		return &Integer{Value: int64(math.Round(*val))}
	}})

	env.Set("pow", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return newError("pow: expected 2 arguments (base, exponent), got %d", len(args))
		}
		base := toFloat64(args[0])
		exp := toFloat64(args[1])
		if base == nil || exp == nil {
			return newError("pow: arguments must be numbers")
		}
		result := math.Pow(*base, *exp)
		// Return integer if result has no fractional part
		if result == float64(int64(result)) {
			return &Integer{Value: int64(result)}
		}
		return &Float{Value: result}
	}})

	env.Set("log", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("log: expected 1 argument, got %d", len(args))
		}
		val := toFloat64(args[0])
		if val == nil {
			return newError("log: argument must be a number, got %s", args[0].Type())
		}
		return &Float{Value: math.Log(*val)}
	}})

	env.Set("sin", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("sin: expected 1 argument, got %d", len(args))
		}
		val := toFloat64(args[0])
		if val == nil {
			return newError("sin: argument must be a number, got %s", args[0].Type())
		}
		return &Float{Value: math.Sin(*val)}
	}})

	env.Set("cos", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("cos: expected 1 argument, got %d", len(args))
		}
		val := toFloat64(args[0])
		if val == nil {
			return newError("cos: argument must be a number, got %s", args[0].Type())
		}
		return &Float{Value: math.Cos(*val)}
	}})

	env.Set("tan", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("tan: expected 1 argument, got %d", len(args))
		}
		val := toFloat64(args[0])
		if val == nil {
			return newError("tan: argument must be a number, got %s", args[0].Type())
		}
		return &Float{Value: math.Tan(*val)}
	}})

	env.Set("random", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 0 {
			return newError("random: expected 0 arguments, got %d", len(args))
		}
		return &Float{Value: rand.Float64()}
	}})
}

// toFloat64 converts an Object to float64 pointer, or nil if not numeric.
func toFloat64(obj Object) *float64 {
	switch v := obj.(type) {
	case *Integer:
		f := float64(v.Value)
		return &f
	case *Float:
		return &v.Value
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// regex — Regular expression support
// ---------------------------------------------------------------------------

// RegexObject wraps a compiled regular expression.
type RegexObject struct {
	Pattern string
	Re      *regexp.Regexp
}

func (r *RegexObject) Type() ObjectType { return "REGEX" }
func (r *RegexObject) Inspect() string  { return "Regex(" + r.Pattern + ")" }

func loadRegex(env *Environment) {
	// Regex(pattern) constructor
	env.Set("Regex", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("Regex: expected 1 argument (pattern), got %d", len(args))
		}
		pattern, ok := args[0].(*String)
		if !ok {
			return newError("Regex: argument must be a string, got %s", args[0].Type())
		}
		re, err := regexp.Compile(pattern.Value)
		if err != nil {
			return newError("Regex: invalid pattern: %s", err.Error())
		}
		return &RegexObject{Pattern: pattern.Value, Re: re}
	}})

	// match(regex_or_string, input) — returns first match or nil
	env.Set("match", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return newError("match: expected 2 arguments (pattern, string), got %d", len(args))
		}

		var re *regexp.Regexp

		switch p := args[0].(type) {
		case *RegexObject:
			re = p.Re
		case *String:
			var err error
			re, err = regexp.Compile(p.Value)
			if err != nil {
				return newError("match: invalid pattern: %s", err.Error())
			}
		default:
			return newError("match: first argument must be a Regex or string, got %s", args[0].Type())
		}

		input, ok := args[1].(*String)
		if !ok {
			return newError("match: second argument must be a string, got %s", args[1].Type())
		}

		result := re.FindString(input.Value)
		if result == "" {
			return &Nil{}
		}
		return &String{Value: result}
	}})

	// match_all(regex_or_string, input) — returns all matches
	env.Set("match_all", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return newError("match_all: expected 2 arguments, got %d", len(args))
		}

		var re *regexp.Regexp

		switch p := args[0].(type) {
		case *RegexObject:
			re = p.Re
		case *String:
			var err error
			re, err = regexp.Compile(p.Value)
			if err != nil {
				return newError("match_all: invalid pattern: %s", err.Error())
			}
		default:
			return newError("match_all: first argument must be a Regex or string, got %s", args[0].Type())
		}

		input, ok := args[1].(*String)
		if !ok {
			return newError("match_all: second argument must be a string, got %s", args[1].Type())
		}

		matches := re.FindAllString(input.Value, -1)
		elements := make([]Object, len(matches))
		for i, m := range matches {
			elements[i] = &String{Value: m}
		}
		return &Array{Elements: elements}
	}})

	// replace_regex(pattern, input, replacement) — replace matches
	env.Set("replace_regex", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 3 {
			return newError("replace_regex: expected 3 arguments, got %d", len(args))
		}

		var re *regexp.Regexp

		switch p := args[0].(type) {
		case *RegexObject:
			re = p.Re
		case *String:
			var err error
			re, err = regexp.Compile(p.Value)
			if err != nil {
				return newError("replace_regex: invalid pattern: %s", err.Error())
			}
		default:
			return newError("replace_regex: first argument must be a Regex or string, got %s", args[0].Type())
		}

		input, ok := args[1].(*String)
		if !ok {
			return newError("replace_regex: second argument must be a string, got %s", args[1].Type())
		}

		replacement, ok := args[2].(*String)
		if !ok {
			return newError("replace_regex: third argument must be a string, got %s", args[2].Type())
		}

		return &String{Value: re.ReplaceAllString(input.Value, replacement.Value)}
	}})

	// match_groups(pattern, input) — returns array of capture groups (index 0 = full match)
	env.Set("match_groups", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return newError("match_groups: expected 2 arguments (pattern, string), got %d", len(args))
		}

		var re *regexp.Regexp
		switch p := args[0].(type) {
		case *RegexObject:
			re = p.Re
		case *String:
			var err error
			re, err = regexp.Compile(p.Value)
			if err != nil {
				return newError("match_groups: invalid pattern: %s", err.Error())
			}
		default:
			return newError("match_groups: first argument must be a Regex or string, got %s", args[0].Type())
		}

		input, ok := args[1].(*String)
		if !ok {
			return newError("match_groups: second argument must be a string, got %s", args[1].Type())
		}

		groups := re.FindStringSubmatch(input.Value)
		if groups == nil {
			return &Nil{}
		}
		elements := make([]Object, len(groups))
		for i, g := range groups {
			elements[i] = &String{Value: g}
		}
		return &Array{Elements: elements}
	}})

	// test(pattern, input) — returns true if pattern matches anywhere in input
	env.Set("test", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return newError("test: expected 2 arguments (pattern, string), got %d", len(args))
		}

		var re *regexp.Regexp
		switch p := args[0].(type) {
		case *RegexObject:
			re = p.Re
		case *String:
			var err error
			re, err = regexp.Compile(p.Value)
			if err != nil {
				return newError("test: invalid pattern: %s", err.Error())
			}
		default:
			return newError("test: first argument must be a Regex or string, got %s", args[0].Type())
		}

		input, ok := args[1].(*String)
		if !ok {
			return newError("test: second argument must be a string, got %s", args[1].Type())
		}

		return &Boolean{Value: re.MatchString(input.Value)}
	}})

	// split_regex(pattern, input) — split string by regex pattern
	env.Set("split_regex", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return newError("split_regex: expected 2 arguments (pattern, string), got %d", len(args))
		}

		var re *regexp.Regexp
		switch p := args[0].(type) {
		case *RegexObject:
			re = p.Re
		case *String:
			var err error
			re, err = regexp.Compile(p.Value)
			if err != nil {
				return newError("split_regex: invalid pattern: %s", err.Error())
			}
		default:
			return newError("split_regex: first argument must be a Regex or string, got %s", args[0].Type())
		}

		input, ok := args[1].(*String)
		if !ok {
			return newError("split_regex: second argument must be a string, got %s", args[1].Type())
		}

		parts := re.Split(input.Value, -1)
		elements := make([]Object, len(parts))
		for i, p := range parts {
			elements[i] = &String{Value: p}
		}
		return &Array{Elements: elements}
	}})
}

// ---------------------------------------------------------------------------
// concurrent — Goroutine-based concurrency
// ---------------------------------------------------------------------------

// TaskObject wraps a goroutine result.
type TaskObject struct {
	done   chan struct{}
	result Object
	mu     sync.Mutex
}

func (t *TaskObject) Type() ObjectType { return "TASK" }
func (t *TaskObject) Inspect() string  { return "<Task>" }

// ChannelObject wraps a Go channel.
type ChannelObject struct {
	ch chan Object
}

func (c *ChannelObject) Type() ObjectType { return "CHANNEL" }
func (c *ChannelObject) Inspect() string  { return "<Channel>" }

func loadConcurrent(env *Environment) {
	// spawn(fn) — run function in a goroutine, return a Task
	env.Set("spawn", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("spawn: expected 1 argument (function), got %d", len(args))
		}

		task := &TaskObject{
			done:   make(chan struct{}),
			result: &Nil{},
		}

		// Create a new interpreter with a child environment for the goroutine
		// so it doesn't share mutable state with the main goroutine.
		parentInterp := getBuiltinInterpreter()
		spawnedInterp := &Interpreter{
			env: NewEnclosedEnvironment(parentInterp.env),
		}
		registerBuiltins(spawnedInterp.env)

		go func() {
			defer close(task.done)
			result := spawnedInterp.applyFunction(args[0], []Object{})
			task.mu.Lock()
			task.result = result
			task.mu.Unlock()
		}()

		return task
	}})

	// await(task) — wait for task to complete, return result
	env.Set("await", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("await: expected 1 argument (task), got %d", len(args))
		}
		task, ok := args[0].(*TaskObject)
		if !ok {
			return newError("await: argument must be a Task, got %s", args[0].Type())
		}
		<-task.done
		task.mu.Lock()
		result := task.result
		task.mu.Unlock()
		return result
	}})

	// Channel() — create a new channel
	env.Set("Channel", &Builtin{Fn: func(args ...Object) Object {
		return &ChannelObject{ch: make(chan Object, 1)}
	}})

	// send(channel, value) — send a value to the channel
	env.Set("send", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return newError("send: expected 2 arguments (channel, value), got %d", len(args))
		}
		ch, ok := args[0].(*ChannelObject)
		if !ok {
			return newError("send: first argument must be a Channel, got %s", args[0].Type())
		}
		ch.ch <- args[1]
		return &Nil{}
	}})

	// receive(channel) — receive a value from the channel
	env.Set("receive", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("receive: expected 1 argument (channel), got %d", len(args))
		}
		ch, ok := args[0].(*ChannelObject)
		if !ok {
			return newError("receive: argument must be a Channel, got %s", args[0].Type())
		}
		val := <-ch.ch
		return val
	}})

	// sleep(ms) — sleep for specified milliseconds
	env.Set("sleep", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("sleep: expected 1 argument (milliseconds), got %d", len(args))
		}
		ms, ok := args[0].(*Integer)
		if !ok {
			return newError("sleep: argument must be an integer, got %s", args[0].Type())
		}
		time.Sleep(time.Duration(ms.Value) * time.Millisecond)
		return &Nil{}
	}})
}

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

// EnumObject represents an enum type definition.
type EnumObject struct {
	Name   string
	Values map[string]*Integer
}

func (e *EnumObject) Type() ObjectType { return "ENUM" }
func (e *EnumObject) Inspect() string  { return "<enum " + e.Name + ">" }

// ListStdlibModules returns the names of all available stdlib modules.
func ListStdlibModules() []string {
	modules := make([]string, 0, len(stdlibModules))
	for name := range stdlibModules {
		modules = append(modules, fmt.Sprintf("\"%s\"", name))
	}
	return modules
}
