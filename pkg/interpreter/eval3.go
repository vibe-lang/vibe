package interpreter

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/vibe-lang/vibe/pkg/ast"
)

// Range is an integer range (1..5, 1...5, 1.., ..5) or a single-character string range.
type Range struct {
	Start     int64
	End       int64
	Exclusive bool
	Endless   bool
	BeginLess bool
	StrStart  string
	StrEnd    string
	IsStr     bool
}

func (r *Range) Type() ObjectType { return "RANGE" }
func (r *Range) Inspect() string {
	op := ".."
	if r.Exclusive {
		op = "..."
	}
	switch {
	case r.IsStr:
		return fmt.Sprintf("%q%s%q", r.StrStart, op, r.StrEnd)
	case r.Endless:
		return fmt.Sprintf("%d%s", r.Start, op)
	case r.BeginLess:
		return fmt.Sprintf("%s%d", op, r.End)
	}
	return fmt.Sprintf("%d%s%d", r.Start, op, r.End)
}

// maxRangeLen bounds materialised ranges so an endless range cannot exhaust memory.
const maxRangeLen = 50_000_000

// ToArray converts a Range to an Array. Endless ranges yield an empty array;
// callers that can iterate lazily should check Endless first.
func (r *Range) ToArray() *Array {
	var elements []Object
	if r.IsStr {
		for _, s := range strRange(r.StrStart, r.StrEnd, r.Exclusive) {
			elements = append(elements, &String{Value: s})
		}
		return &Array{Elements: elements}
	}
	if r.Endless || r.BeginLess {
		return &Array{}
	}
	if r.Start <= r.End {
		end := r.End
		if r.Exclusive {
			end--
		}
		for val := r.Start; val <= end && len(elements) < maxRangeLen; val++ {
			elements = append(elements, &Integer{Value: val})
		}
	} else {
		end := r.End
		if r.Exclusive {
			end++
		}
		for val := r.Start; val >= end && len(elements) < maxRangeLen; val-- {
			elements = append(elements, &Integer{Value: val})
		}
	}
	return &Array{Elements: elements}
}

// strRange expands 'a'..'e' style ranges using Ruby's string successor.
func strRange(a, b string, exclusive bool) []string {
	var out []string
	if len(a) > len(b) {
		return out
	}
	cur := a
	for n := 0; n < 1_000_000; n++ {
		if cur == b {
			if !exclusive {
				out = append(out, cur)
			}
			return out
		}
		if len(cur) > len(b) {
			return out
		}
		out = append(out, cur)
		cur = strSucc(cur)
	}
	return out
}

func strSucc(s string) string {
	if s == "" {
		return ""
	}
	b := []byte(s)
	for idx := len(b) - 1; idx >= 0; idx-- {
		c := b[idx]
		switch {
		case c == 'z':
			b[idx] = 'a'
		case c == 'Z':
			b[idx] = 'A'
		case c == '9':
			b[idx] = '0'
		default:
			b[idx]++
			return string(b)
		}
		if idx == 0 {
			prefix := byte('a')
			switch c {
			case 'Z':
				prefix = 'A'
			case '9':
				prefix = '1'
			}
			return string(prefix) + string(b)
		}
	}
	return string(b)
}

// evalRangeExpression evaluates a range expression like 1..5 or 1...5.
func (i *Interpreter) evalRangeExpression(node *ast.RangeExpression) Object {
	var start, end Object
	if node.Start != nil {
		start = i.eval(node.Start)
		if isError(start) {
			return start
		}
	}
	if node.End != nil {
		end = i.eval(node.End)
		if isError(end) {
			return end
		}
	}
	return i.makeRange(start, end, node.Exclusive)
}

func (i *Interpreter) makeRange(start, end Object, exclusive bool) Object {
	if ss, ok := start.(*String); ok {
		se, ok := end.(*String)
		if !ok {
			return newError("bad value for range")
		}
		return &Range{IsStr: true, StrStart: ss.Value, StrEnd: se.Value, Exclusive: exclusive}
	}
	r := &Range{Exclusive: exclusive}
	if start == nil {
		r.BeginLess = true
	} else {
		n, ok := toInt64Strict(start)
		if !ok {
			return newError("range start must be an integer, got %s", start.Type())
		}
		r.Start = n
	}
	if end == nil {
		r.Endless = true
	} else if _, isNil := end.(*Nil); isNil {
		r.Endless = true
	} else {
		n, ok := toInt64Strict(end)
		if !ok {
			return newError("range end must be an integer, got %s", end.Type())
		}
		r.End = n
	}
	return r
}

func toInt64Strict(o Object) (int64, bool) {
	switch v := o.(type) {
	case *Integer:
		return v.Value, true
	case *Float:
		return int64(v.Value), true
	}
	return 0, false
}

// evalRangeCallExpression evaluates a Range() constructor call.
func (i *Interpreter) evalRangeCallExpression(node *ast.RangeCallExpression) Object {
	start := i.eval(node.Start)
	if isError(start) {
		return start
	}
	end := i.eval(node.End)
	if isError(end) {
		return end
	}
	return i.makeRange(start, end, node.Exclusive)
}

// iterItems returns the elements a for-loop or each-style iteration visits.
func (i *Interpreter) iterItems(c Object) ([]Object, bool) {
	switch v := c.(type) {
	case *Array:
		return v.Elements, true
	case *Range:
		return v.ToArray().Elements, true
	case *Hash:
		out := make([]Object, 0, len(v.Order))
		for _, k := range v.Order {
			out = append(out, &Array{Elements: []Object{v.keyObj(k), v.Pairs[k]}})
		}
		return out, true
	case *String:
		var out []Object
		for _, r := range v.Value {
			out = append(out, &String{Value: string(r)})
		}
		return out, true
	case *Nil:
		return nil, true
	case *StructInstance:
		var out []Object
		for _, f := range v.Struct.fieldOrder() {
			out = append(out, &Array{Elements: []Object{&String{Value: f}, v.Fields[f]}})
		}
		return out, true
	case *ClassInstance:
		if m := v.Class.findMethod("each"); m != nil {
			var items []Object
			collector := &Builtin{Fn: func(args ...Object) Object {
				if len(args) == 1 {
					items = append(items, args[0])
				} else {
					items = append(items, &Array{Elements: args})
				}
				return &Nil{}
			}}
			if r := i.callMethodWithBlock(v, m.owner, m.fn, nil, collector); isError(r) {
				return nil, false
			}
			return items, true
		}
	}
	return nil, false
}

// evalForLoop evaluates `for x in collection`, `for k, v in hash`, and `for a, b in pairs`.
func (i *Interpreter) evalForLoop(node *ast.ForLoop) Object {
	collection := i.eval(node.Collection)
	if isError(collection) {
		return collection
	}

	if r, ok := collection.(*Range); ok && r.Endless && !r.IsStr {
		return i.forEndless(node, r)
	}

	items, ok := i.iterItems(collection)
	if !ok {
		return newError("for loop collection must be an array, range, or hash, got %s", collection.Type())
	}
	_, isHash := collection.(*Hash)
	_, isStruct := collection.(*StructInstance)

	for idx, element := range items {
		var res Object
		if node.ValueIterator != nil {
			if pair, ok := element.(*Array); ok && (isHash || isStruct || len(pair.Elements) == 2) && (isHash || isStruct) {
				i.setVar(node.Iterator.Value, pair.Elements[0])
				i.setVar(node.ValueIterator.Value, pair.Elements[1])
			} else if pair, ok := element.(*Array); ok && len(pair.Elements) >= 2 {
				i.setVar(node.Iterator.Value, pair.Elements[0])
				i.setVar(node.ValueIterator.Value, pair.Elements[1])
			} else {
				i.setVar(node.Iterator.Value, element)
				i.setVar(node.ValueIterator.Value, &Integer{Value: int64(idx)})
			}
		} else {
			i.setVar(node.Iterator.Value, element)
		}
		res = i.evalBlockStatement(node.Body)
		if stop, out := loopControl(res); stop {
			return out
		}
	}
	return collection
}

func (i *Interpreter) forEndless(node *ast.ForLoop, r *Range) Object {
	for n := r.Start; ; n++ {
		i.setVar(node.Iterator.Value, &Integer{Value: n})
		res := i.evalBlockStatement(node.Body)
		if stop, out := loopControl(res); stop {
			return out
		}
	}
}

// evalIndexExpression evaluates obj[index].
func (i *Interpreter) evalIndexExpression(node *ast.IndexExpression) Object {
	left := i.eval(node.Left)
	if isError(left) {
		return left
	}
	index := i.eval(node.Index)
	if isError(index) {
		return index
	}
	return i.indexGet(left, index)
}

// indexGet implements obj[index] for builtin collections and user objects.
func (i *Interpreter) indexGet(left, index Object) Object {
	switch obj := left.(type) {
	case *Array:
		switch ix := index.(type) {
		case *Integer:
			n := ix.Value
			if n < 0 {
				n += int64(len(obj.Elements))
			}
			if n < 0 || n >= int64(len(obj.Elements)) {
				return newError("array index out of bounds: %d", ix.Value)
			}
			return obj.Elements[n]
		case *Range:
			start, length, ok := rangeBounds(ix, len(obj.Elements))
			if !ok {
				return &Nil{}
			}
			end := start + length
			if end > len(obj.Elements) {
				end = len(obj.Elements)
			}
			return &Array{Elements: append([]Object{}, obj.Elements[start:end]...)}
		case *Float:
			return i.indexGet(left, &Integer{Value: int64(ix.Value)})
		}
		return newError("array index must be an integer, got %s", index.Type())
	case *String:
		runes := []rune(obj.Value)
		switch ix := index.(type) {
		case *Integer:
			n := ix.Value
			if n < 0 {
				n += int64(len(runes))
			}
			if n < 0 || n >= int64(len(runes)) {
				return newError("string index out of bounds: %d", ix.Value)
			}
			return &String{Value: string(runes[n])}
		case *Range:
			start, length, ok := rangeBounds(ix, len(runes))
			if !ok {
				return &Nil{}
			}
			end := start + length
			if end > len(runes) {
				end = len(runes)
			}
			return &String{Value: string(runes[start:end])}
		case *String:
			if strings.Contains(obj.Value, ix.Value) {
				return &String{Value: ix.Value}
			}
			return &Nil{}
		case *RegexObject:
			m := ix.Re.FindString(obj.Value)
			if m == "" && !ix.Re.MatchString(obj.Value) {
				return &Nil{}
			}
			return &String{Value: m}
		}
		return newError("string index must be an integer, got %s", index.Type())
	case *Hash:
		if v, ok := obj.Pairs[hashKey(index)]; ok {
			return v
		}
		if obj.Default != nil {
			if fn, ok := obj.Default.(*Function); ok {
				return i.callFunction(fn, []Object{obj, index}, nil)
			}
			return obj.Default
		}
		return &Nil{}
	case *StructInstance:
		switch ix := index.(type) {
		case *String:
			if v, ok := obj.Fields[ix.Value]; ok {
				return v
			}
		case *Symbol:
			if v, ok := obj.Fields[ix.Name]; ok {
				return v
			}
		case *Integer:
			order := obj.Struct.fieldOrder()
			if int(ix.Value) < len(order) && ix.Value >= 0 {
				return obj.Fields[order[ix.Value]]
			}
		}
		return &Nil{}
	case *ClassInstance:
		if m := obj.Class.findMethod("[]"); m != nil {
			return i.callMethodWithBlock(obj, m.owner, m.fn, []Object{index}, nil)
		}
	case *Compound:
		if ix, ok := index.(*Integer); ok && ix.Value >= 0 && int(ix.Value) < len(obj.Elements) {
			return obj.Elements[ix.Value]
		}
		return newError("compound index out of bounds")
	case *ClassObject:
		if m := obj.findStatic("[]"); m != nil {
			return i.callStatic(obj, m, []Object{index}, nil)
		}
	case *Function, *Builtin, *BoundMethod:
		return i.callFunction(left, []Object{index}, nil)
	case *RegexObject:
		return &Nil{}
	case *Nil:
		return newError("undefined method '[]' for nil")
	}
	return newError("index operator not supported for %s", left.Type())
}

// evalStringInterpolation evaluates a string with interpolated expressions.
func (i *Interpreter) evalStringInterpolation(node *ast.StringInterpolationLiteral) Object {
	var out bytes.Buffer
	for idx, part := range node.Parts {
		out.WriteString(part)
		if idx < len(node.Expressions) {
			val := i.eval(node.Expressions[idx])
			if isError(val) {
				return val
			}
			out.WriteString(i.display(val))
		}
	}
	return &String{Value: out.String()}
}

// evalStructLiteral evaluates Name(field: value, ...) when Name is a struct or class.
func (i *Interpreter) evalStructLiteral(node *ast.StructLiteral) Object {
	typeObj, ok := i.env.Get(node.Type)
	if !ok {
		if c := i.lookupConstant(node.Type); c != nil {
			typeObj = c
		} else {
			return newError("undefined type: %s", node.Type)
		}
	}

	order := node.FieldOrder
	if len(order) == 0 {
		for k := range node.Fields {
			order = append(order, k)
		}
	}
	args := []Object{}
	kw := &Hash{Pairs: map[string]Object{}, IsKw: true}
	for _, name := range order {
		val := i.eval(node.Fields[name])
		if isError(val) {
			return val
		}
		kw.set(name, &Symbol{Name: name}, val)
	}
	if len(order) > 0 {
		args = append(args, kw)
	}

	switch t := typeObj.(type) {
	case *Struct:
		if len(node.TypeArgs) > 0 && len(t.TypeParams) > 0 && len(node.TypeArgs) != len(t.TypeParams) {
			return newError("wrong number of type arguments for %s: expected %d, got %d",
				node.Type, len(t.TypeParams), len(node.TypeArgs))
		}
		res := i.constructStruct(t, args)
		if si, ok := res.(*StructInstance); ok {
			si.TypeArgs = node.TypeArgs
		}
		return res
	case *ClassObject:
		if len(node.TypeArgs) > 0 && len(t.TypeParams) > 0 && len(node.TypeArgs) != len(t.TypeParams) {
			return newError("wrong number of type arguments for %s: expected %d, got %d",
				node.Type, len(t.TypeParams), len(node.TypeArgs))
		}
		res := i.callClassConstructor(t, args, nil)
		if ci, ok := res.(*ClassInstance); ok && len(node.TypeArgs) > 0 {
			ci.TypeArgs = node.TypeArgs
		}
		return res
	case *Builtin:
		return t.Fn(positionalValues(args)...)
	}
	return newError("not a struct type: %s", node.Type)
}

func positionalValues(args []Object) []Object {
	if n := len(args); n == 1 {
		if h, ok := args[0].(*Hash); ok && h.IsKw {
			out := make([]Object, 0, len(h.Order))
			for _, k := range h.Order {
				out = append(out, h.Pairs[k])
			}
			return out
		}
	}
	return args
}

// constructStruct builds a struct instance from keyword or positional arguments.
func (i *Interpreter) constructStruct(s *Struct, args []Object) Object {
	inst := &StructInstance{Struct: s, Fields: map[string]Object{}}
	for k, v := range s.DefaultValues {
		inst.Fields[k] = v
	}
	order := s.fieldOrder()
	if len(args) == 1 {
		if h, ok := args[0].(*Hash); ok && (h.IsKw || len(order) != 1) {
			for _, k := range h.Order {
				if _, exists := s.Fields[k]; !exists {
					return newError("undefined field '%s' in struct '%s'", k, s.Name)
				}
				inst.Fields[k] = h.Pairs[k]
			}
			return inst
		}
	}
	if len(args) > len(order) {
		return newError("wrong number of arguments for struct %s (given %d, expected 0..%d)", s.Name, len(args), len(order))
	}
	for idx, a := range args {
		inst.Fields[order[idx]] = a
	}
	return inst
}
