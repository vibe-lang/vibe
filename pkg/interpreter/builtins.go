package interpreter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// registerBuiltins adds all builtin functions to the given environment.
func registerBuiltins(env *Environment) {
	env.Set("puts", &Builtin{Fn: builtinPuts})
	env.Set("print", &Builtin{Fn: builtinPrint})
	env.Set("len", &Builtin{Fn: builtinLen})
	env.Set("type", &Builtin{Fn: builtinType})
	env.Set("push", &Builtin{Fn: builtinPush})
	env.Set("pop", &Builtin{Fn: builtinPop})
	env.Set("first", &Builtin{Fn: builtinFirst})
	env.Set("last", &Builtin{Fn: builtinLast})
	env.Set("rest", &Builtin{Fn: builtinRest})
	env.Set("append", &Builtin{Fn: builtinPush}) // alias for push
	env.Set("to_s", &Builtin{Fn: builtinToS})
	env.Set("to_i", &Builtin{Fn: builtinToI})
	env.Set("to_f", &Builtin{Fn: builtinToF})
	env.Set("input", &Builtin{Fn: builtinInput})
	env.Set("exit", &Builtin{Fn: builtinExit})
	env.Set("map", &Builtin{Fn: builtinMap})
	env.Set("filter", &Builtin{Fn: builtinFilter})
	env.Set("each", &Builtin{Fn: builtinEach})
	env.Set("sort", &Builtin{Fn: builtinSort})
	env.Set("reverse", &Builtin{Fn: builtinReverse})
	env.Set("contains", &Builtin{Fn: builtinContains})
	env.Set("keys", &Builtin{Fn: builtinKeys})
	env.Set("values", &Builtin{Fn: builtinValues})
	env.Set("split", &Builtin{Fn: builtinSplit})
	env.Set("join", &Builtin{Fn: builtinJoin})
	env.Set("replace", &Builtin{Fn: builtinReplace})
	env.Set("trim", &Builtin{Fn: builtinTrim})
	env.Set("abs", &Builtin{Fn: builtinAbs})
	env.Set("min", &Builtin{Fn: builtinMin})
	env.Set("max", &Builtin{Fn: builtinMax})
	env.Set("string_length", &Builtin{Fn: builtinStringLength})
	env.Set("read_file", &Builtin{Fn: builtinReadFile})
	env.Set("write_file", &Builtin{Fn: builtinWriteFile})
	env.Set("file_exists", &Builtin{Fn: builtinFileExists})
	env.Set("json_parse", &Builtin{Fn: builtinJSONParse})
	env.Set("json_encode", &Builtin{Fn: builtinJSONEncode})
	env.Set("json", &Builtin{Fn: builtinJSONParse}) // alias: "...".json() reads naturally

	// Array methods
	env.Set("reduce", &Builtin{Fn: builtinReduce})
	env.Set("find", &Builtin{Fn: builtinFind})
	env.Set("find_index", &Builtin{Fn: builtinFindIndex})
	env.Set("index_of", &Builtin{Fn: builtinIndexOf})
	env.Set("any", &Builtin{Fn: builtinAny})
	env.Set("all", &Builtin{Fn: builtinAll})
	env.Set("none", &Builtin{Fn: builtinNone})
	env.Set("empty", &Builtin{Fn: builtinEmpty})
	env.Set("count", &Builtin{Fn: builtinCount})
	env.Set("flatten", &Builtin{Fn: builtinFlatten})
	env.Set("compact", &Builtin{Fn: builtinCompact})
	env.Set("uniq", &Builtin{Fn: builtinUniq})
	env.Set("reject", &Builtin{Fn: builtinReject})
	env.Set("take", &Builtin{Fn: builtinTake})
	env.Set("drop", &Builtin{Fn: builtinDrop})
	env.Set("slice", &Builtin{Fn: builtinSlice})
	env.Set("flat_map", &Builtin{Fn: builtinFlatMap})
	env.Set("sort_by", &Builtin{Fn: builtinSortBy})
	env.Set("sum", &Builtin{Fn: builtinSum})
	env.Set("zip", &Builtin{Fn: builtinZip})
	env.Set("concat", &Builtin{Fn: builtinConcat})
	env.Set("each_with_index", &Builtin{Fn: builtinEachWithIndex})
	env.Set("map_with_index", &Builtin{Fn: builtinMapWithIndex})
	env.Set("partition", &Builtin{Fn: builtinPartition})
	env.Set("group_by", &Builtin{Fn: builtinGroupBy})
	env.Set("select", &Builtin{Fn: builtinFilter}) // alias for filter
	env.Set("shift", &Builtin{Fn: builtinRest})    // alias for rest (returns all but first)
	env.Set("unshift", &Builtin{Fn: builtinUnshift})

	// String methods
	env.Set("upcase", &Builtin{Fn: builtinUpcase})
	env.Set("downcase", &Builtin{Fn: builtinDowncase})
	env.Set("capitalize", &Builtin{Fn: builtinCapitalize})
	env.Set("starts_with", &Builtin{Fn: builtinStartsWith})
	env.Set("ends_with", &Builtin{Fn: builtinEndsWith})
	env.Set("repeat", &Builtin{Fn: builtinRepeat})
	env.Set("chars", &Builtin{Fn: builtinChars})
	env.Set("pad_start", &Builtin{Fn: builtinPadStart})
	env.Set("pad_end", &Builtin{Fn: builtinPadEnd})
	env.Set("string_reverse", &Builtin{Fn: builtinStringReverse})
	env.Set("string_slice", &Builtin{Fn: builtinStringSlice})
	env.Set("string_contains", &Builtin{Fn: builtinStringContains})
	env.Set("index_of_string", &Builtin{Fn: builtinIndexOfString})
}

// builtinInterpreter is stored so map/filter/each can call user functions.
// This is set by the interpreter during evaluation.
var builtinInterpreter *Interpreter

func builtinPuts(args ...Object) Object {
	for _, arg := range args {
		fmt.Println(arg.Inspect())
	}
	return &Nil{}
}

func builtinPrint(args ...Object) Object {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = arg.Inspect()
	}
	fmt.Print(strings.Join(parts, " "))
	return &Nil{}
}

func builtinLen(args ...Object) Object {
	if len(args) != 1 {
		return newError("len: expected 1 argument, got %d", len(args))
	}
	switch arg := args[0].(type) {
	case *Array:
		return &Integer{Value: int64(len(arg.Elements))}
	case *String:
		return &Integer{Value: int64(len(arg.Value))}
	case *Hash:
		return &Integer{Value: int64(len(arg.Pairs))}
	default:
		return newError("len: unsupported type %s", args[0].Type())
	}
}

func builtinType(args ...Object) Object {
	if len(args) != 1 {
		return newError("type: expected 1 argument, got %d", len(args))
	}
	return &String{Value: string(args[0].Type())}
}

func builtinPush(args ...Object) Object {
	if len(args) != 2 {
		return newError("push: expected 2 arguments (array, element), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("push: first argument must be an array, got %s", args[0].Type())
	}
	newElements := make([]Object, len(arr.Elements)+1)
	copy(newElements, arr.Elements)
	newElements[len(arr.Elements)] = args[1]
	return &Array{Elements: newElements}
}

func builtinPop(args ...Object) Object {
	if len(args) != 1 {
		return newError("pop: expected 1 argument, got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("pop: argument must be an array, got %s", args[0].Type())
	}
	if len(arr.Elements) == 0 {
		return newError("pop: array is empty")
	}
	return arr.Elements[len(arr.Elements)-1]
}

func builtinFirst(args ...Object) Object {
	if len(args) != 1 {
		return newError("first: expected 1 argument, got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("first: argument must be an array, got %s", args[0].Type())
	}
	if len(arr.Elements) == 0 {
		return &Nil{}
	}
	return arr.Elements[0]
}

func builtinLast(args ...Object) Object {
	if len(args) != 1 {
		return newError("last: expected 1 argument, got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("last: argument must be an array, got %s", args[0].Type())
	}
	if len(arr.Elements) == 0 {
		return &Nil{}
	}
	return arr.Elements[len(arr.Elements)-1]
}

func builtinRest(args ...Object) Object {
	if len(args) != 1 {
		return newError("rest: expected 1 argument, got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("rest: argument must be an array, got %s", args[0].Type())
	}
	if len(arr.Elements) == 0 {
		return &Array{Elements: []Object{}}
	}
	newElements := make([]Object, len(arr.Elements)-1)
	copy(newElements, arr.Elements[1:])
	return &Array{Elements: newElements}
}

func builtinToS(args ...Object) Object {
	if len(args) != 1 {
		return newError("to_s: expected 1 argument, got %d", len(args))
	}
	return &String{Value: args[0].Inspect()}
}

func builtinToI(args ...Object) Object {
	if len(args) != 1 {
		return newError("to_i: expected 1 argument, got %d", len(args))
	}
	switch arg := args[0].(type) {
	case *Integer:
		return arg
	case *Float:
		return &Integer{Value: int64(arg.Value)}
	case *String:
		val, err := strconv.ParseInt(arg.Value, 10, 64)
		if err != nil {
			return newError("to_i: cannot convert %q to integer", arg.Value)
		}
		return &Integer{Value: val}
	case *Boolean:
		if arg.Value {
			return &Integer{Value: 1}
		}
		return &Integer{Value: 0}
	default:
		return newError("to_i: unsupported type %s", args[0].Type())
	}
}

func builtinToF(args ...Object) Object {
	if len(args) != 1 {
		return newError("to_f: expected 1 argument, got %d", len(args))
	}
	switch arg := args[0].(type) {
	case *Float:
		return arg
	case *Integer:
		return &Float{Value: float64(arg.Value)}
	case *String:
		val, err := strconv.ParseFloat(arg.Value, 64)
		if err != nil {
			return newError("to_f: cannot convert %q to float", arg.Value)
		}
		return &Float{Value: val}
	default:
		return newError("to_f: unsupported type %s", args[0].Type())
	}
}

func builtinInput(args ...Object) Object {
	if len(args) > 1 {
		return newError("input: expected 0 or 1 arguments, got %d", len(args))
	}
	if len(args) == 1 {
		fmt.Print(args[0].Inspect())
	}
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return &String{Value: scanner.Text()}
	}
	return &String{Value: ""}
}

func builtinExit(args ...Object) Object {
	code := 0
	if len(args) == 1 {
		if intObj, ok := args[0].(*Integer); ok {
			code = int(intObj.Value)
		}
	}
	os.Exit(code)
	return &Nil{}
}

func builtinMap(args ...Object) Object {
	if len(args) != 2 {
		return newError("map: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("map: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	results := make([]Object, len(arr.Elements))
	for idx, elem := range arr.Elements {
		result := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(result) {
			return result
		}
		results[idx] = result
	}
	return &Array{Elements: results}
}

func builtinFilter(args ...Object) Object {
	if len(args) != 2 {
		return newError("filter: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("filter: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	var results []Object
	for _, elem := range arr.Elements {
		result := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(result) {
			return result
		}
		if isTruthy(result) {
			results = append(results, elem)
		}
	}
	if results == nil {
		results = []Object{}
	}
	return &Array{Elements: results}
}

func builtinEach(args ...Object) Object {
	if len(args) != 2 {
		return newError("each: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("each: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	for _, elem := range arr.Elements {
		result := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(result) {
			return result
		}
	}
	return &Nil{}
}

func builtinSort(args ...Object) Object {
	if len(args) != 1 {
		return newError("sort: expected 1 argument, got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("sort: argument must be an array, got %s", args[0].Type())
	}
	newElements := make([]Object, len(arr.Elements))
	copy(newElements, arr.Elements)
	sort.Slice(newElements, func(a, b int) bool {
		aInt, aOk := newElements[a].(*Integer)
		bInt, bOk := newElements[b].(*Integer)
		if aOk && bOk {
			return aInt.Value < bInt.Value
		}
		aStr, aOk := newElements[a].(*String)
		bStr, bOk := newElements[b].(*String)
		if aOk && bOk {
			return aStr.Value < bStr.Value
		}
		return false
	})
	return &Array{Elements: newElements}
}

func builtinReverse(args ...Object) Object {
	if len(args) != 1 {
		return newError("reverse: expected 1 argument, got %d", len(args))
	}
	switch arg := args[0].(type) {
	case *Array:
		newElements := make([]Object, len(arg.Elements))
		for idx, elem := range arg.Elements {
			newElements[len(arg.Elements)-1-idx] = elem
		}
		return &Array{Elements: newElements}
	case *String:
		runes := []rune(arg.Value)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return &String{Value: string(runes)}
	default:
		return newError("reverse: argument must be an array or string, got %s", args[0].Type())
	}
}

func builtinContains(args ...Object) Object {
	if len(args) != 2 {
		return newError("contains: expected 2 arguments, got %d", len(args))
	}
	switch container := args[0].(type) {
	case *Array:
		for _, elem := range container.Elements {
			if elem.Inspect() == args[1].Inspect() {
				return &Boolean{Value: true}
			}
		}
		return &Boolean{Value: false}
	case *String:
		substr, ok := args[1].(*String)
		if !ok {
			return newError("contains: second argument must be a string for string search")
		}
		return &Boolean{Value: strings.Contains(container.Value, substr.Value)}
	case *Hash:
		key := args[1].Inspect()
		for k := range container.Pairs {
			if k == key {
				return &Boolean{Value: true}
			}
		}
		return &Boolean{Value: false}
	default:
		return newError("contains: unsupported type %s", args[0].Type())
	}
}

func builtinKeys(args ...Object) Object {
	if len(args) != 1 {
		return newError("keys: expected 1 argument, got %d", len(args))
	}
	hash, ok := args[0].(*Hash)
	if !ok {
		return newError("keys: argument must be a hash, got %s", args[0].Type())
	}
	keys := make([]Object, 0, len(hash.Pairs))
	for k := range hash.Pairs {
		keys = append(keys, &String{Value: k})
	}
	return &Array{Elements: keys}
}

func builtinValues(args ...Object) Object {
	if len(args) != 1 {
		return newError("values: expected 1 argument, got %d", len(args))
	}
	hash, ok := args[0].(*Hash)
	if !ok {
		return newError("values: argument must be a hash, got %s", args[0].Type())
	}
	vals := make([]Object, 0, len(hash.Pairs))
	for _, v := range hash.Pairs {
		vals = append(vals, v)
	}
	return &Array{Elements: vals}
}

func builtinSplit(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return newError("split: expected 1-2 arguments (string, delimiter?), got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("split: first argument must be a string, got %s", args[0].Type())
	}

	// No delimiter: split into individual characters
	if len(args) == 1 {
		chars := []rune(str.Value)
		elements := make([]Object, len(chars))
		for i, c := range chars {
			elements[i] = &String{Value: string(c)}
		}
		return &Array{Elements: elements}
	}

	delim, ok := args[1].(*String)
	if !ok {
		return newError("split: second argument must be a string, got %s", args[1].Type())
	}
	parts := strings.Split(str.Value, delim.Value)
	elements := make([]Object, len(parts))
	for i, p := range parts {
		elements[i] = &String{Value: p}
	}
	return &Array{Elements: elements}
}

func builtinJoin(args ...Object) Object {
	if len(args) != 2 {
		return newError("join: expected 2 arguments (array, separator), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("join: first argument must be an array, got %s", args[0].Type())
	}
	sep, ok := args[1].(*String)
	if !ok {
		return newError("join: second argument must be a string, got %s", args[1].Type())
	}
	parts := make([]string, len(arr.Elements))
	for i, elem := range arr.Elements {
		parts[i] = elem.Inspect()
	}
	return &String{Value: strings.Join(parts, sep.Value)}
}

func builtinReplace(args ...Object) Object {
	if len(args) != 3 {
		return newError("replace: expected 3 arguments (string, old, new), got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("replace: first argument must be a string")
	}
	old, ok := args[1].(*String)
	if !ok {
		return newError("replace: second argument must be a string")
	}
	new_, ok := args[2].(*String)
	if !ok {
		return newError("replace: third argument must be a string")
	}
	return &String{Value: strings.ReplaceAll(str.Value, old.Value, new_.Value)}
}

func builtinTrim(args ...Object) Object {
	if len(args) != 1 {
		return newError("trim: expected 1 argument, got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("trim: argument must be a string, got %s", args[0].Type())
	}
	return &String{Value: strings.TrimSpace(str.Value)}
}

func builtinAbs(args ...Object) Object {
	if len(args) != 1 {
		return newError("abs: expected 1 argument, got %d", len(args))
	}
	switch arg := args[0].(type) {
	case *Integer:
		if arg.Value < 0 {
			return &Integer{Value: -arg.Value}
		}
		return arg
	case *Float:
		if arg.Value < 0 {
			return &Float{Value: -arg.Value}
		}
		return arg
	default:
		return newError("abs: argument must be a number, got %s", args[0].Type())
	}
}

func builtinMin(args ...Object) Object {
	if len(args) != 2 {
		return newError("min: expected 2 arguments, got %d", len(args))
	}
	a, aOk := args[0].(*Integer)
	b, bOk := args[1].(*Integer)
	if aOk && bOk {
		if a.Value < b.Value {
			return a
		}
		return b
	}
	return newError("min: arguments must be integers")
}

func builtinMax(args ...Object) Object {
	if len(args) != 2 {
		return newError("max: expected 2 arguments, got %d", len(args))
	}
	a, aOk := args[0].(*Integer)
	b, bOk := args[1].(*Integer)
	if aOk && bOk {
		if a.Value > b.Value {
			return a
		}
		return b
	}
	return newError("max: arguments must be integers")
}

func builtinStringLength(args ...Object) Object {
	if len(args) != 1 {
		return newError("string_length: expected 1 argument, got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("string_length: argument must be a string, got %s", args[0].Type())
	}
	return &Integer{Value: int64(len(str.Value))}
}

func builtinReadFile(args ...Object) Object {
	if len(args) != 1 {
		return newError("read_file: expected 1 argument, got %d", len(args))
	}
	path, ok := args[0].(*String)
	if !ok {
		return newError("read_file: argument must be a string, got %s", args[0].Type())
	}
	data, err := os.ReadFile(path.Value)
	if err != nil {
		return newError("read_file: %s", err.Error())
	}
	return &String{Value: string(data)}
}

func builtinWriteFile(args ...Object) Object {
	if len(args) != 2 {
		return newError("write_file: expected 2 arguments (path, content), got %d", len(args))
	}
	path, ok := args[0].(*String)
	if !ok {
		return newError("write_file: first argument must be a string")
	}
	content, ok := args[1].(*String)
	if !ok {
		return newError("write_file: second argument must be a string")
	}
	err := os.WriteFile(path.Value, []byte(content.Value), 0644)
	if err != nil {
		return newError("write_file: %s", err.Error())
	}
	return &Nil{}
}

func builtinFileExists(args ...Object) Object {
	if len(args) != 1 {
		return newError("file_exists: expected 1 argument, got %d", len(args))
	}
	path, ok := args[0].(*String)
	if !ok {
		return newError("file_exists: argument must be a string")
	}
	_, err := os.Stat(path.Value)
	return &Boolean{Value: err == nil}
}

// ---------------------------------------------------------------------------
// Array methods
// ---------------------------------------------------------------------------

func builtinReduce(args ...Object) Object {
	if len(args) < 2 || len(args) > 3 {
		return newError("reduce: expected 2-3 arguments (array, function, initial?), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("reduce: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]

	var acc Object
	startIdx := 0
	if len(args) == 3 {
		acc = args[2]
	} else {
		if len(arr.Elements) == 0 {
			return newError("reduce: cannot reduce empty array without initial value")
		}
		acc = arr.Elements[0]
		startIdx = 1
	}

	for idx := startIdx; idx < len(arr.Elements); idx++ {
		result := builtinInterpreter.applyFunction(fn, []Object{acc, arr.Elements[idx]})
		if isError(result) {
			return result
		}
		acc = result
	}
	return acc
}

func builtinFind(args ...Object) Object {
	if len(args) != 2 {
		return newError("find: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("find: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	for _, elem := range arr.Elements {
		result := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(result) {
			return result
		}
		if isTruthy(result) {
			return elem
		}
	}
	return &Nil{}
}

func builtinFindIndex(args ...Object) Object {
	if len(args) != 2 {
		return newError("find_index: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("find_index: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	for idx, elem := range arr.Elements {
		result := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(result) {
			return result
		}
		if isTruthy(result) {
			return &Integer{Value: int64(idx)}
		}
	}
	return &Integer{Value: -1}
}

func builtinIndexOf(args ...Object) Object {
	if len(args) != 2 {
		return newError("index_of: expected 2 arguments (array, value), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("index_of: first argument must be an array, got %s", args[0].Type())
	}
	target := args[1].Inspect()
	for idx, elem := range arr.Elements {
		if elem.Inspect() == target {
			return &Integer{Value: int64(idx)}
		}
	}
	return &Integer{Value: -1}
}

func builtinAny(args ...Object) Object {
	if len(args) != 2 {
		return newError("any: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("any: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	for _, elem := range arr.Elements {
		result := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(result) {
			return result
		}
		if isTruthy(result) {
			return &Boolean{Value: true}
		}
	}
	return &Boolean{Value: false}
}

func builtinAll(args ...Object) Object {
	if len(args) != 2 {
		return newError("all: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("all: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	for _, elem := range arr.Elements {
		result := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(result) {
			return result
		}
		if !isTruthy(result) {
			return &Boolean{Value: false}
		}
	}
	return &Boolean{Value: true}
}

func builtinNone(args ...Object) Object {
	if len(args) != 2 {
		return newError("none: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("none: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	for _, elem := range arr.Elements {
		result := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(result) {
			return result
		}
		if isTruthy(result) {
			return &Boolean{Value: false}
		}
	}
	return &Boolean{Value: true}
}

func builtinEmpty(args ...Object) Object {
	if len(args) != 1 {
		return newError("empty: expected 1 argument, got %d", len(args))
	}
	switch arg := args[0].(type) {
	case *Array:
		return &Boolean{Value: len(arg.Elements) == 0}
	case *String:
		return &Boolean{Value: len(arg.Value) == 0}
	case *Hash:
		return &Boolean{Value: len(arg.Pairs) == 0}
	default:
		return newError("empty: unsupported type %s", args[0].Type())
	}
}

func builtinCount(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return newError("count: expected 1-2 arguments (array, function?), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("count: first argument must be an array, got %s", args[0].Type())
	}
	if len(args) == 1 {
		return &Integer{Value: int64(len(arr.Elements))}
	}
	fn := args[1]
	count := int64(0)
	for _, elem := range arr.Elements {
		result := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(result) {
			return result
		}
		if isTruthy(result) {
			count++
		}
	}
	return &Integer{Value: count}
}

func builtinFlatten(args ...Object) Object {
	if len(args) != 1 {
		return newError("flatten: expected 1 argument, got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("flatten: argument must be an array, got %s", args[0].Type())
	}
	return &Array{Elements: flattenArray(arr.Elements)}
}

func flattenArray(elements []Object) []Object {
	var result []Object
	for _, elem := range elements {
		if inner, ok := elem.(*Array); ok {
			result = append(result, flattenArray(inner.Elements)...)
		} else {
			result = append(result, elem)
		}
	}
	if result == nil {
		result = []Object{}
	}
	return result
}

func builtinCompact(args ...Object) Object {
	if len(args) != 1 {
		return newError("compact: expected 1 argument, got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("compact: argument must be an array, got %s", args[0].Type())
	}
	var result []Object
	for _, elem := range arr.Elements {
		if _, isNil := elem.(*Nil); !isNil {
			result = append(result, elem)
		}
	}
	if result == nil {
		result = []Object{}
	}
	return &Array{Elements: result}
}

func builtinUniq(args ...Object) Object {
	if len(args) != 1 {
		return newError("uniq: expected 1 argument, got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("uniq: argument must be an array, got %s", args[0].Type())
	}
	seen := map[string]bool{}
	var result []Object
	for _, elem := range arr.Elements {
		key := elem.Inspect()
		if !seen[key] {
			seen[key] = true
			result = append(result, elem)
		}
	}
	if result == nil {
		result = []Object{}
	}
	return &Array{Elements: result}
}

func builtinReject(args ...Object) Object {
	if len(args) != 2 {
		return newError("reject: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("reject: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	var result []Object
	for _, elem := range arr.Elements {
		r := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(r) {
			return r
		}
		if !isTruthy(r) {
			result = append(result, elem)
		}
	}
	if result == nil {
		result = []Object{}
	}
	return &Array{Elements: result}
}

func builtinTake(args ...Object) Object {
	if len(args) != 2 {
		return newError("take: expected 2 arguments (array, count), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("take: first argument must be an array, got %s", args[0].Type())
	}
	n, ok := args[1].(*Integer)
	if !ok {
		return newError("take: second argument must be an integer, got %s", args[1].Type())
	}
	count := int(n.Value)
	if count > len(arr.Elements) {
		count = len(arr.Elements)
	}
	if count < 0 {
		count = 0
	}
	result := make([]Object, count)
	copy(result, arr.Elements[:count])
	return &Array{Elements: result}
}

func builtinDrop(args ...Object) Object {
	if len(args) != 2 {
		return newError("drop: expected 2 arguments (array, count), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("drop: first argument must be an array, got %s", args[0].Type())
	}
	n, ok := args[1].(*Integer)
	if !ok {
		return newError("drop: second argument must be an integer, got %s", args[1].Type())
	}
	count := int(n.Value)
	if count > len(arr.Elements) {
		count = len(arr.Elements)
	}
	if count < 0 {
		count = 0
	}
	result := make([]Object, len(arr.Elements)-count)
	copy(result, arr.Elements[count:])
	return &Array{Elements: result}
}

func builtinSlice(args ...Object) Object {
	if len(args) < 2 || len(args) > 3 {
		return newError("slice: expected 2-3 arguments (array, start, end?), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("slice: first argument must be an array, got %s", args[0].Type())
	}
	startObj, ok := args[1].(*Integer)
	if !ok {
		return newError("slice: second argument must be an integer, got %s", args[1].Type())
	}
	start := int(startObj.Value)
	if start < 0 {
		start = len(arr.Elements) + start
	}
	if start < 0 {
		start = 0
	}
	if start > len(arr.Elements) {
		return &Array{Elements: []Object{}}
	}

	end := len(arr.Elements)
	if len(args) == 3 {
		endObj, ok := args[2].(*Integer)
		if !ok {
			return newError("slice: third argument must be an integer, got %s", args[2].Type())
		}
		end = int(endObj.Value)
		if end < 0 {
			end = len(arr.Elements) + end
		}
		if end > len(arr.Elements) {
			end = len(arr.Elements)
		}
	}
	if end < start {
		return &Array{Elements: []Object{}}
	}

	result := make([]Object, end-start)
	copy(result, arr.Elements[start:end])
	return &Array{Elements: result}
}

func builtinFlatMap(args ...Object) Object {
	if len(args) != 2 {
		return newError("flat_map: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("flat_map: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	var result []Object
	for _, elem := range arr.Elements {
		r := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(r) {
			return r
		}
		if inner, ok := r.(*Array); ok {
			result = append(result, inner.Elements...)
		} else {
			result = append(result, r)
		}
	}
	if result == nil {
		result = []Object{}
	}
	return &Array{Elements: result}
}

func builtinSortBy(args ...Object) Object {
	if len(args) != 2 {
		return newError("sort_by: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("sort_by: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]

	// Compute keys for all elements
	type elemKey struct {
		elem Object
		key  Object
	}
	pairs := make([]elemKey, len(arr.Elements))
	for idx, elem := range arr.Elements {
		k := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(k) {
			return k
		}
		pairs[idx] = elemKey{elem, k}
	}

	sort.SliceStable(pairs, func(a, b int) bool {
		aKey, bKey := pairs[a].key, pairs[b].key
		aInt, aOk := aKey.(*Integer)
		bInt, bOk := bKey.(*Integer)
		if aOk && bOk {
			return aInt.Value < bInt.Value
		}
		aStr, aOk := aKey.(*String)
		bStr, bOk := bKey.(*String)
		if aOk && bOk {
			return aStr.Value < bStr.Value
		}
		aFloat, aOk := aKey.(*Float)
		bFloat, bOk := bKey.(*Float)
		if aOk && bOk {
			return aFloat.Value < bFloat.Value
		}
		return false
	})

	result := make([]Object, len(pairs))
	for i, p := range pairs {
		result[i] = p.elem
	}
	return &Array{Elements: result}
}

func builtinSum(args ...Object) Object {
	if len(args) != 1 {
		return newError("sum: expected 1 argument, got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("sum: argument must be an array, got %s", args[0].Type())
	}
	if len(arr.Elements) == 0 {
		return &Integer{Value: 0}
	}

	hasFloat := false
	sumInt := int64(0)
	sumFloat := float64(0)

	for _, elem := range arr.Elements {
		switch v := elem.(type) {
		case *Integer:
			sumInt += v.Value
			sumFloat += float64(v.Value)
		case *Float:
			hasFloat = true
			sumFloat += v.Value
		default:
			return newError("sum: array contains non-numeric element of type %s", elem.Type())
		}
	}
	if hasFloat {
		return &Float{Value: sumFloat}
	}
	return &Integer{Value: sumInt}
}

func builtinZip(args ...Object) Object {
	if len(args) != 2 {
		return newError("zip: expected 2 arguments (array, array), got %d", len(args))
	}
	a, aOk := args[0].(*Array)
	b, bOk := args[1].(*Array)
	if !aOk || !bOk {
		return newError("zip: both arguments must be arrays")
	}
	length := len(a.Elements)
	if len(b.Elements) < length {
		length = len(b.Elements)
	}
	result := make([]Object, length)
	for i := 0; i < length; i++ {
		result[i] = &Array{Elements: []Object{a.Elements[i], b.Elements[i]}}
	}
	return &Array{Elements: result}
}

func builtinConcat(args ...Object) Object {
	if len(args) < 2 {
		return newError("concat: expected at least 2 arguments, got %d", len(args))
	}
	first, ok := args[0].(*Array)
	if !ok {
		return newError("concat: first argument must be an array, got %s", args[0].Type())
	}
	var result []Object
	result = append(result, first.Elements...)
	for i := 1; i < len(args); i++ {
		other, ok := args[i].(*Array)
		if !ok {
			return newError("concat: argument %d must be an array, got %s", i+1, args[i].Type())
		}
		result = append(result, other.Elements...)
	}
	return &Array{Elements: result}
}

func builtinEachWithIndex(args ...Object) Object {
	if len(args) != 2 {
		return newError("each_with_index: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("each_with_index: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	for idx, elem := range arr.Elements {
		result := builtinInterpreter.applyFunction(fn, []Object{elem, &Integer{Value: int64(idx)}})
		if isError(result) {
			return result
		}
	}
	return &Nil{}
}

func builtinMapWithIndex(args ...Object) Object {
	if len(args) != 2 {
		return newError("map_with_index: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("map_with_index: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	results := make([]Object, len(arr.Elements))
	for idx, elem := range arr.Elements {
		result := builtinInterpreter.applyFunction(fn, []Object{elem, &Integer{Value: int64(idx)}})
		if isError(result) {
			return result
		}
		results[idx] = result
	}
	return &Array{Elements: results}
}

func builtinPartition(args ...Object) Object {
	if len(args) != 2 {
		return newError("partition: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("partition: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	var yes, no []Object
	for _, elem := range arr.Elements {
		result := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(result) {
			return result
		}
		if isTruthy(result) {
			yes = append(yes, elem)
		} else {
			no = append(no, elem)
		}
	}
	if yes == nil {
		yes = []Object{}
	}
	if no == nil {
		no = []Object{}
	}
	return &Array{Elements: []Object{
		&Array{Elements: yes},
		&Array{Elements: no},
	}}
}

func builtinGroupBy(args ...Object) Object {
	if len(args) != 2 {
		return newError("group_by: expected 2 arguments (array, function), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("group_by: first argument must be an array, got %s", args[0].Type())
	}
	fn := args[1]
	groups := map[string][]Object{}
	order := []string{}
	for _, elem := range arr.Elements {
		key := builtinInterpreter.applyFunction(fn, []Object{elem})
		if isError(key) {
			return key
		}
		k := key.Inspect()
		if _, exists := groups[k]; !exists {
			order = append(order, k)
		}
		groups[k] = append(groups[k], elem)
	}
	pairs := map[string]Object{}
	for k, elems := range groups {
		pairs[k] = &Array{Elements: elems}
	}
	return &Hash{Pairs: pairs, Order: order}
}

func builtinUnshift(args ...Object) Object {
	if len(args) != 2 {
		return newError("unshift: expected 2 arguments (array, element), got %d", len(args))
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newError("unshift: first argument must be an array, got %s", args[0].Type())
	}
	newElements := make([]Object, len(arr.Elements)+1)
	newElements[0] = args[1]
	copy(newElements[1:], arr.Elements)
	return &Array{Elements: newElements}
}

// ---------------------------------------------------------------------------
// String methods
// ---------------------------------------------------------------------------

func builtinUpcase(args ...Object) Object {
	if len(args) != 1 {
		return newError("upcase: expected 1 argument, got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("upcase: argument must be a string, got %s", args[0].Type())
	}
	return &String{Value: strings.ToUpper(str.Value)}
}

func builtinDowncase(args ...Object) Object {
	if len(args) != 1 {
		return newError("downcase: expected 1 argument, got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("downcase: argument must be a string, got %s", args[0].Type())
	}
	return &String{Value: strings.ToLower(str.Value)}
}

func builtinCapitalize(args ...Object) Object {
	if len(args) != 1 {
		return newError("capitalize: expected 1 argument, got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("capitalize: argument must be a string, got %s", args[0].Type())
	}
	if len(str.Value) == 0 {
		return str
	}
	return &String{Value: strings.ToUpper(str.Value[:1]) + strings.ToLower(str.Value[1:])}
}

func builtinStartsWith(args ...Object) Object {
	if len(args) != 2 {
		return newError("starts_with: expected 2 arguments (string, prefix), got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("starts_with: first argument must be a string, got %s", args[0].Type())
	}
	prefix, ok := args[1].(*String)
	if !ok {
		return newError("starts_with: second argument must be a string, got %s", args[1].Type())
	}
	return &Boolean{Value: strings.HasPrefix(str.Value, prefix.Value)}
}

func builtinEndsWith(args ...Object) Object {
	if len(args) != 2 {
		return newError("ends_with: expected 2 arguments (string, suffix), got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("ends_with: first argument must be a string, got %s", args[0].Type())
	}
	suffix, ok := args[1].(*String)
	if !ok {
		return newError("ends_with: second argument must be a string, got %s", args[1].Type())
	}
	return &Boolean{Value: strings.HasSuffix(str.Value, suffix.Value)}
}

func builtinRepeat(args ...Object) Object {
	if len(args) != 2 {
		return newError("repeat: expected 2 arguments (string, count), got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("repeat: first argument must be a string, got %s", args[0].Type())
	}
	count, ok := args[1].(*Integer)
	if !ok {
		return newError("repeat: second argument must be an integer, got %s", args[1].Type())
	}
	if count.Value < 0 {
		return newError("repeat: count must be non-negative")
	}
	return &String{Value: strings.Repeat(str.Value, int(count.Value))}
}

func builtinChars(args ...Object) Object {
	if len(args) != 1 {
		return newError("chars: expected 1 argument, got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("chars: argument must be a string, got %s", args[0].Type())
	}
	runes := []rune(str.Value)
	elements := make([]Object, len(runes))
	for i, r := range runes {
		elements[i] = &String{Value: string(r)}
	}
	return &Array{Elements: elements}
}

func builtinPadStart(args ...Object) Object {
	if len(args) < 2 || len(args) > 3 {
		return newError("pad_start: expected 2-3 arguments (string, length, padChar?), got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("pad_start: first argument must be a string, got %s", args[0].Type())
	}
	length, ok := args[1].(*Integer)
	if !ok {
		return newError("pad_start: second argument must be an integer, got %s", args[1].Type())
	}
	padChar := " "
	if len(args) == 3 {
		pc, ok := args[2].(*String)
		if !ok {
			return newError("pad_start: third argument must be a string, got %s", args[2].Type())
		}
		if len(pc.Value) > 0 {
			padChar = string([]rune(pc.Value)[0])
		}
	}
	result := str.Value
	for int64(len(result)) < length.Value {
		result = padChar + result
	}
	return &String{Value: result}
}

func builtinPadEnd(args ...Object) Object {
	if len(args) < 2 || len(args) > 3 {
		return newError("pad_end: expected 2-3 arguments (string, length, padChar?), got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("pad_end: first argument must be a string, got %s", args[0].Type())
	}
	length, ok := args[1].(*Integer)
	if !ok {
		return newError("pad_end: second argument must be an integer, got %s", args[1].Type())
	}
	padChar := " "
	if len(args) == 3 {
		pc, ok := args[2].(*String)
		if !ok {
			return newError("pad_end: third argument must be a string, got %s", args[2].Type())
		}
		if len(pc.Value) > 0 {
			padChar = string([]rune(pc.Value)[0])
		}
	}
	result := str.Value
	for int64(len(result)) < length.Value {
		result = result + padChar
	}
	return &String{Value: result}
}

func builtinStringReverse(args ...Object) Object {
	if len(args) != 1 {
		return newError("string_reverse: expected 1 argument, got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("string_reverse: argument must be a string, got %s", args[0].Type())
	}
	runes := []rune(str.Value)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return &String{Value: string(runes)}
}

func builtinStringSlice(args ...Object) Object {
	if len(args) < 2 || len(args) > 3 {
		return newError("string_slice: expected 2-3 arguments (string, start, end?), got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("string_slice: first argument must be a string, got %s", args[0].Type())
	}
	startObj, ok := args[1].(*Integer)
	if !ok {
		return newError("string_slice: second argument must be an integer, got %s", args[1].Type())
	}
	runes := []rune(str.Value)
	start := int(startObj.Value)
	if start < 0 {
		start = len(runes) + start
	}
	if start < 0 {
		start = 0
	}
	if start > len(runes) {
		return &String{Value: ""}
	}
	end := len(runes)
	if len(args) == 3 {
		endObj, ok := args[2].(*Integer)
		if !ok {
			return newError("string_slice: third argument must be an integer, got %s", args[2].Type())
		}
		end = int(endObj.Value)
		if end < 0 {
			end = len(runes) + end
		}
		if end > len(runes) {
			end = len(runes)
		}
	}
	if end < start {
		return &String{Value: ""}
	}
	return &String{Value: string(runes[start:end])}
}

func builtinStringContains(args ...Object) Object {
	if len(args) != 2 {
		return newError("string_contains: expected 2 arguments, got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("string_contains: first argument must be a string, got %s", args[0].Type())
	}
	substr, ok := args[1].(*String)
	if !ok {
		return newError("string_contains: second argument must be a string, got %s", args[1].Type())
	}
	return &Boolean{Value: strings.Contains(str.Value, substr.Value)}
}

func builtinIndexOfString(args ...Object) Object {
	if len(args) != 2 {
		return newError("index_of_string: expected 2 arguments, got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("index_of_string: first argument must be a string, got %s", args[0].Type())
	}
	substr, ok := args[1].(*String)
	if !ok {
		return newError("index_of_string: second argument must be a string, got %s", args[1].Type())
	}
	return &Integer{Value: int64(strings.Index(str.Value, substr.Value))}
}

// ---------------------------------------------------------------------------
// json_parse(string) — parse JSON string into Vibe objects
// json_encode(object) — encode Vibe objects into JSON string
// ---------------------------------------------------------------------------

func builtinJSONParse(args ...Object) Object {
	if len(args) != 1 {
		return newError("json_parse: expected 1 argument, got %d", len(args))
	}
	str, ok := args[0].(*String)
	if !ok {
		return newError("json_parse: argument must be a string, got %s", args[0].Type())
	}

	var raw interface{}
	if err := json.Unmarshal([]byte(str.Value), &raw); err != nil {
		return newError("json_parse: %s", err.Error())
	}

	return goValueToObject(raw)
}

func builtinJSONEncode(args ...Object) Object {
	if len(args) != 1 {
		return newError("json_encode: expected 1 argument, got %d", len(args))
	}

	goVal := objectToGoValue(args[0])
	data, err := json.Marshal(goVal)
	if err != nil {
		return newError("json_encode: %s", err.Error())
	}
	return &String{Value: string(data)}
}

// goValueToObject converts a Go interface{} (from JSON) to a Vibe Object.
func goValueToObject(val interface{}) Object {
	switch v := val.(type) {
	case nil:
		return &Nil{}
	case bool:
		return &Boolean{Value: v}
	case float64:
		// JSON numbers are always float64; convert to int if no fractional part
		if v == float64(int64(v)) {
			return &Integer{Value: int64(v)}
		}
		return &Float{Value: v}
	case string:
		return &String{Value: v}
	case []interface{}:
		elements := make([]Object, len(v))
		for i, elem := range v {
			elements[i] = goValueToObject(elem)
		}
		return &Array{Elements: elements}
	case map[string]interface{}:
		pairs := make(map[string]Object, len(v))
		order := make([]string, 0, len(v))
		for k, val := range v {
			pairs[k] = goValueToObject(val)
			order = append(order, k)
		}
		return &Hash{Pairs: pairs, Order: order}
	default:
		return &String{Value: fmt.Sprintf("%v", v)}
	}
}

// objectToGoValue converts a Vibe Object to a Go interface{} for JSON encoding.
func objectToGoValue(obj Object) interface{} {
	switch o := obj.(type) {
	case *Nil:
		return nil
	case *Boolean:
		return o.Value
	case *Integer:
		return o.Value
	case *Float:
		return o.Value
	case *String:
		return o.Value
	case *Array:
		result := make([]interface{}, len(o.Elements))
		for i, elem := range o.Elements {
			result[i] = objectToGoValue(elem)
		}
		return result
	case *Hash:
		result := make(map[string]interface{}, len(o.Pairs))
		for k, v := range o.Pairs {
			result[k] = objectToGoValue(v)
		}
		return result
	default:
		return o.Inspect()
	}
}
