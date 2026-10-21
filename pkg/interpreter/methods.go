package interpreter

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

var hashMethodNames = map[string]bool{
	"keys": true, "values": true, "length": true, "size": true, "count": true, "each": true, "map": true,
	"select": true, "filter": true, "reject": true, "merge": true, "key?": true, "has_key?": true,
	"include?": true, "member?": true, "fetch": true, "dig": true, "delete": true, "to_a": true,
	"empty?": true, "any?": true, "all?": true, "none?": true, "sum": true, "min_by": true, "max_by": true,
	"sort_by": true, "group_by": true, "partition": true, "find": true, "first": true, "invert": true,
	"transform_values": true, "transform_keys": true, "each_pair": true, "to_h": true, "freeze": true,
	"merge!": true, "update": true, "store": true, "clear": true, "key": true, "values_at": true,
	"slice": true, "except": true, "compact": true, "sort": true, "reduce": true, "inject": true,
	"filter_map": true, "flat_map": true, "each_with_object": true, "each_with_index": true,
	"default": true, "default=": true, "class": true, "inspect": true, "to_s": true, "dup": true,
	"value?": true, "has_value?": true, "find_all": true, "tally": true, "zip": true, "entries": true,
	"is_a?": true, "respond_to?": true, "nil?": true, "frozen?": true, "send": true, "tap": true, "then": true,
	"delete_if": true, "keep_if": true, "select!": true, "reject!": true, "min": true, "max": true,
	"uniq": true, "take": true, "drop": true, "each_key": true, "each_value": true,
	"fetch_values": true, "assoc": true, "to_json": true, "hash": true, "equal?": true,
	"eql?": true, "itself": true, "instance_of?": true, "kind_of?": true,
}

func hasBuiltinMethod(recv Object, name string) bool {
	if _, ok := recv.(*Hash); ok {
		return hashMethodNames[name]
	}
	return false
}

func (i *Interpreter) callBuiltinMethod(recv Object, name string, args []Object, blk Object) (Object, bool) {
	var res Object
	var ok bool
	if blk == nil && len(args) > 0 && blockTakers[name] {
		switch recv.(type) {
		case *Array, *Hash, *Range:
			switch args[len(args)-1].(type) {
			case *Function, *Builtin, *BoundMethod:
				blk = args[len(args)-1]
				args = args[:len(args)-1]
			}
		}
	}
	switch r := recv.(type) {
	case *Integer:
		res, ok = i.intMethod(r, name, args, blk)
	case *Float:
		res, ok = i.floatMethod(r, name, args, blk)
	case *String:
		res, ok = i.stringMethod(r, name, args, blk)
	case *Symbol:
		res, ok = i.symbolMethod(r, name, args, blk)
	case *Array:
		res, ok = i.arrayMethod(r, name, args, blk)
	case *Hash:
		res, ok = i.hashMethod(r, name, args, blk)
	case *Range:
		res, ok = i.rangeMethod(r, name, args, blk)
	case *Function, *Builtin, *BoundMethod:
		res, ok = i.procMethod(recv, name, args, blk)
	case *Nil:
		res, ok = i.nilMethod(name, args)
	case *Boolean:
		res, ok = i.boolMethod(r, name, args)
	case *RegexObject:
		res, ok = i.regexMethod(r, name, args, blk)
	case *StructInstance:
		res, ok = i.callStructMethod(r, name, args, blk)
	case *Struct:
		res, ok = i.structClassMethod(r, name, args, blk)
	case *VibeError:
		res, ok = i.vibeErrorMethod(r, name)
	case *Compound:
		res, ok = i.arrayMethod(&Array{Elements: r.Elements}, name, args, blk)
	}
	if ok {
		return res, true
	}
	return i.objectMethod(recv, name, args, blk)
}

func (i *Interpreter) yield(blk Object, args ...Object) Object {
	return i.callFunction(blk, args, nil)
}

func argInt(args []Object, idx int, def int64) (int64, bool) {
	if idx >= len(args) {
		return def, true
	}
	n, ok := toInt64Strict(args[idx])
	return n, ok
}

func (i *Interpreter) frozenErr(o Object) Object {
	return newError("can't modify frozen %s: %s", i.className(o), i.inspectValue(o))
}

// objectMethod provides the methods every value responds to.
func (i *Interpreter) objectMethod(recv Object, name string, args []Object, blk Object) (Object, bool) {
	switch name {
	case "class":
		return i.classOf(recv), true
	case "is_a?", "kind_of?", "instance_of?":
		if len(args) != 1 {
			return newError("wrong number of arguments (given %d, expected 1)", len(args)), true
		}
		c, ok := args[0].(*ClassObject)
		if !ok {
			if s, ok := args[0].(*Struct); ok {
				si, isS := recv.(*StructInstance)
				return boolObj(isS && si.Struct == s), true
			}
			return newError("class or module required"), true
		}
		if name == "instance_of?" {
			return boolObj(i.classOf(recv) == Object(c)), true
		}
		return boolObj(i.isInstanceOf(recv, c)), true
	case "nil?":
		_, isNil := recv.(*Nil)
		return boolObj(isNil), true
	case "respond_to?":
		if len(args) < 1 {
			return newError("wrong number of arguments (given 0, expected 1)"), true
		}
		return boolObj(i.respondTo(recv, symOrString(args[0]), len(args) > 1 && isTruthy(args[1]))), true
	case "send", "public_send", "__send__":
		if len(args) < 1 {
			return newError("no method name given"), true
		}
		mode := callMode{isCall: true, selfCall: name != "public_send"}
		return i.callMember(recv, symOrString(args[0]), args[1:], blk, mode), true
	case "method":
		if len(args) != 1 {
			return newError("wrong number of arguments (given %d, expected 1)", len(args)), true
		}
		return &BoundMethod{Recv: recv, Name: symOrString(args[0])}, true
	case "methods", "public_methods":
		return &Array{Elements: i.methodNames(recv)}, true
	case "freeze":
		i.freeze(recv)
		return recv, true
	case "frozen?":
		return boolObj(i.isFrozen(recv)), true
	case "dup", "clone":
		d := i.dup(recv)
		if name == "clone" && i.isFrozen(recv) {
			frozenOpt := true
			if len(args) > 0 {
				if h, ok := args[len(args)-1].(*Hash); ok {
					if v, ok := h.Pairs["freeze"]; ok && !isTruthy(v) {
						frozenOpt = false
					}
				}
			}
			if frozenOpt {
				i.freeze(d)
			}
		}
		return d, true
	case "inspect":
		return &String{Value: i.inspectValue(recv)}, true
	case "to_s":
		return &String{Value: i.display(recv)}, true
	case "itself":
		return recv, true
	case "tap":
		if blk == nil {
			return newError("no block given (tap)"), true
		}
		if r := i.yield(blk, recv); isError(r) {
			return r, true
		}
		return recv, true
	case "then", "yield_self":
		if blk == nil {
			return recv, true
		}
		return i.yield(blk, recv), true
	case "==", "eql?", "equal?":
		if name == "equal?" {
			return boolObj(recv == args[0]), true
		}
		return boolObj(i.equal(recv, args[0])), true
	case "!=":
		return boolObj(!i.equal(recv, args[0])), true
	case "!":
		return boolObj(!isTruthy(recv)), true
	case "===":
		return i.caseEqual(recv, args[0]), true
	case "=~":
		return &Nil{}, true
	case "<=>":
		if c, ok := i.compare(recv, args[0]); ok {
			return &Integer{Value: int64(c)}, true
		}
		return &Nil{}, true
	case "hash":
		return &Integer{Value: int64(hashString(i.keyString(recv)))}, true
	case "object_id", "__id__":
		return &Integer{Value: int64(objectID(recv))}, true
	case "instance_variables":
		var out []Object
		if ci, ok := recv.(*ClassInstance); ok {
			for _, k := range sortedKeys(ci.Fields) {
				if !strings.HasPrefix(k, "__") {
					out = append(out, &Symbol{Name: "@" + k})
				}
			}
		}
		return &Array{Elements: out}, true
	case "instance_variable_get":
		if ci, ok := recv.(*ClassInstance); ok && len(args) == 1 {
			if v, ok := ci.Fields[strings.TrimPrefix(symOrString(args[0]), "@")]; ok {
				return v, true
			}
			return &Nil{}, true
		}
	case "instance_variable_set":
		if ci, ok := recv.(*ClassInstance); ok && len(args) == 2 {
			if ci.Frozen {
				return i.frozenErr(recv), true
			}
			ci.Fields[strings.TrimPrefix(symOrString(args[0]), "@")] = args[1]
			return args[1], true
		}
	case "instance_variable_defined?":
		if ci, ok := recv.(*ClassInstance); ok && len(args) == 1 {
			_, has := ci.Fields[strings.TrimPrefix(symOrString(args[0]), "@")]
			return boolObj(has), true
		}
	case "extend":
		if ci, ok := recv.(*ClassInstance); ok {
			sing := newClassObject(ci.Class.Name)
			sing.Parent = ci.Class
			for _, a := range args {
				if m, ok := a.(*ClassObject); ok {
					sing.Includes = append(sing.Includes, m)
				}
			}
			ci.Class = sing
			return recv, true
		}
	case "display":
		fmt.Print(i.display(recv))
		return &Nil{}, true
	case "instance_eval", "instance_exec":
		if fn, ok := blk.(*Function); ok {
			env := NewEnclosedEnvironment(fn.Env)
			env.Set("self", recv)
			cp := *fn
			return i.invoke(&cp, []Object{recv}, env, nil), true
		}
	case "to_enum", "each_entry", "enum_for":
		if items, ok := i.iterItems(recv); ok {
			return &Array{Elements: items}, true
		}
	}
	return nil, false
}

func hashString(s string) uint32 {
	var h uint32 = 2166136261
	for k := 0; k < len(s); k++ {
		h ^= uint32(s[k])
		h *= 16777619
	}
	return h
}

var objectIDs = map[Object]int{}

func objectID(o Object) int {
	switch v := o.(type) {
	case *Integer:
		return int(v.Value)*2 + 1
	case *Nil:
		return 8
	case *Boolean:
		if v.Value {
			return 20
		}
		return 0
	}
	if id, ok := objectIDs[o]; ok {
		return id
	}
	id := (len(objectIDs) + 1) * 8
	objectIDs[o] = id
	return id
}

func sortedKeys(m map[string]Object) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// keyString yields a string that is identical for values that are == (used by uniq, tally, group_by).
func (i *Interpreter) keyString(o Object) string {
	switch v := o.(type) {
	case *Integer:
		return "n:" + strconv.FormatInt(v.Value, 10)
	case *Float:
		if v.Value == math.Trunc(v.Value) && math.Abs(v.Value) < 1e15 {
			return "n:" + strconv.FormatInt(int64(v.Value), 10)
		}
		return "f:" + strconv.FormatFloat(v.Value, 'g', -1, 64)
	case *String:
		return "s:" + v.Value
	case *Symbol:
		return "y:" + v.Name
	case *Array:
		parts := make([]string, len(v.Elements))
		for idx, e := range v.Elements {
			parts[idx] = i.keyString(e)
		}
		return "a:[" + strings.Join(parts, ",") + "]"
	case *Hash:
		parts := make([]string, 0, len(v.Order))
		for _, k := range v.Order {
			parts = append(parts, k+"=>"+i.keyString(v.Pairs[k]))
		}
		sort.Strings(parts)
		return "h:{" + strings.Join(parts, ",") + "}"
	case *ClassInstance:
		if m := v.Class.findMethod("hash"); m != nil {
			return "c:" + i.display(i.callMethodWithBlock(v, m.owner, m.fn, nil, nil))
		}
		return fmt.Sprintf("c:%p", v)
	case *StructInstance:
		parts := []string{}
		for _, f := range v.Struct.fieldOrder() {
			parts = append(parts, i.keyString(v.Fields[f]))
		}
		return "t:" + v.Struct.Name + "(" + strings.Join(parts, ",") + ")"
	case *Nil:
		return "nil"
	case *Boolean:
		return "b:" + strconv.FormatBool(v.Value)
	}
	return string(o.Type()) + ":" + o.Inspect()
}

// classOf returns the class object representing the value's class.
func (i *Interpreter) classOf(o Object) Object {
	switch v := o.(type) {
	case *ClassInstance:
		c := v.Class
		for c != nil && c.Singleton {
			c = c.Parent
		}
		return c
	case *StructInstance:
		return v.Struct
	}
	name := i.className(o)
	if c := i.globalClass(name); c != nil {
		return c
	}
	return &String{Value: name}
}

func (i *Interpreter) respondTo(recv Object, name string, includePrivate bool) bool {
	switch r := recv.(type) {
	case *ClassInstance:
		if m := r.Class.findMethod(name); m != nil {
			return includePrivate || m.fn.Visibility == ""
		}
		if r.Class.findNative(name) != nil {
			return true
		}
		if _, ok := r.Fields[name]; ok {
			return false
		}
		if mm := r.Class.findMethod("respond_to_missing?"); mm != nil {
			return isTruthy(i.callMethodWithBlock(r, mm.owner, mm.fn, []Object{&Symbol{Name: name}, boolObj(includePrivate)}, nil))
		}
	case *ClassObject:
		if r.findStatic(name) != nil || r.findNativeStatic(name) != nil {
			return true
		}
		switch name {
		case "new", "name", "ancestors", "superclass", "class", "instance_methods":
			return true
		}
		return false
	case *StructInstance:
		if _, ok := r.Fields[name]; ok {
			return true
		}
	case *Hash:
		if _, ok := r.Pairs[name]; ok {
			return true
		}
	}
	// A builtin method answers if a probe-free lookup says so.
	if builtinResponds(recv, name) {
		return true
	}
	if fn, ok := i.env.Get(name); ok && isCallable(fn) {
		switch recv.(type) {
		case *ClassInstance, *ClassObject:
			return false
		}
		return true
	}
	return false
}

func (i *Interpreter) methodNames(recv Object) []Object {
	var out []Object
	if ci, ok := recv.(*ClassInstance); ok {
		seen := map[string]bool{}
		for _, a := range ci.Class.ancestors() {
			for _, n := range sortedMethodNames(a) {
				if !seen[n] && a.Methods[n].Visibility == "" {
					seen[n] = true
					out = append(out, &Symbol{Name: n})
				}
			}
		}
	}
	return out
}

func sortedMethodNames(c *ClassObject) []string {
	names := make([]string, 0, len(c.Methods))
	for k := range c.Methods {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func (i *Interpreter) freeze(o Object) {
	switch v := o.(type) {
	case *Array:
		v.Frozen = true
	case *Hash:
		v.Frozen = true
	case *String:
		v.Frozen = true
	case *ClassInstance:
		v.Frozen = true
	case *StructInstance:
		v.Frozen = true
	case *ClassObject:
		v.Frozen = true
	}
}

func (i *Interpreter) isFrozen(o Object) bool {
	switch v := o.(type) {
	case *Array:
		return v.Frozen
	case *Hash:
		return v.Frozen
	case *String:
		return v.Frozen
	case *ClassInstance:
		return v.Frozen
	case *StructInstance:
		return v.Frozen
	case *ClassObject:
		return v.Frozen
	case *Integer, *Float, *Symbol, *Nil, *Boolean, *Range:
		return true
	}
	return false
}

func (i *Interpreter) dup(o Object) Object {
	switch v := o.(type) {
	case *Array:
		return &Array{Elements: append([]Object{}, v.Elements...)}
	case *Hash:
		h := &Hash{Pairs: map[string]Object{}, Default: v.Default}
		for _, k := range v.Order {
			h.set(k, v.keyObj(k), v.Pairs[k])
		}
		return h
	case *String:
		return &String{Value: v.Value}
	case *ClassInstance:
		c := &ClassInstance{Class: v.Class, Fields: map[string]Object{}, TypeArgs: v.TypeArgs}
		for k, x := range v.Fields {
			c.Fields[k] = x
		}
		if m := v.Class.findMethod("initialize_copy"); m != nil {
			i.callMethodWithBlock(c, m.owner, m.fn, []Object{v}, nil)
		}
		return c
	case *StructInstance:
		c := &StructInstance{Struct: v.Struct, TypeArgs: v.TypeArgs, Fields: map[string]Object{}}
		for k, x := range v.Fields {
			c.Fields[k] = x
		}
		return c
	}
	return o
}

// ---------------------------------------------------------------------------
// Numbers
// ---------------------------------------------------------------------------

func (i *Interpreter) intMethod(n *Integer, name string, args []Object, blk Object) (Object, bool) {
	v := n.Value
	switch name {
	case "times":
		if blk == nil {
			var out []Object
			for k := int64(0); k < v; k++ {
				out = append(out, &Integer{Value: k})
			}
			return &Array{Elements: out}, true
		}
		for k := int64(0); k < v; k++ {
			if r := i.yield(blk, &Integer{Value: k}); isError(r) {
				return r, true
			}
		}
		return n, true
	case "upto", "downto":
		if len(args) < 1 {
			return newError("wrong number of arguments (given 0, expected 1)"), true
		}
		lim, ok := toInt64Strict(args[0])
		if !ok {
			return newError("comparison of Integer with %s failed", args[0].Type()), true
		}
		var out []Object
		if name == "upto" {
			for k := v; k <= lim; k++ {
				out = append(out, &Integer{Value: k})
			}
		} else {
			for k := v; k >= lim; k-- {
				out = append(out, &Integer{Value: k})
			}
		}
		if blk == nil {
			return &Array{Elements: out}, true
		}
		for _, e := range out {
			if r := i.yield(blk, e); isError(r) {
				return r, true
			}
		}
		return n, true
	case "step":
		if len(args) < 2 {
			return newError("wrong number of arguments (given %d, expected 2)", len(args)), true
		}
		lim, ok1 := toFloat(args[0])
		step, ok2 := toFloat(args[1])
		if !ok1 || !ok2 || step == 0 {
			return newError("step can't be 0"), true
		}
		_, intStep := args[1].(*Integer)
		_, intLim := args[0].(*Integer)
		var out []Object
		for x := float64(v); (step > 0 && x <= lim) || (step < 0 && x >= lim); x += step {
			if intStep && intLim {
				out = append(out, &Integer{Value: int64(x)})
			} else {
				out = append(out, &Float{Value: x})
			}
		}
		if blk == nil {
			return &Array{Elements: out}, true
		}
		for _, e := range out {
			if r := i.yield(blk, e); isError(r) {
				return r, true
			}
		}
		return n, true
	case "to_s", "inspect":
		base := int64(10)
		if len(args) > 0 {
			base, _ = toInt64Strict(args[0])
		}
		if base < 2 || base > 36 {
			return newError("invalid radix %d", base), true
		}
		return &String{Value: strconv.FormatInt(v, int(base))}, true
	case "to_i", "to_int", "floor", "ceil", "round", "truncate", "ord":
		if len(args) > 0 && (name == "round" || name == "floor" || name == "ceil" || name == "truncate") {
			d, _ := toInt64Strict(args[0])
			if d < 0 {
				p := int64(math.Pow(10, float64(-d)))
				switch name {
				case "floor":
					return &Integer{Value: floorDiv(v, p) * p}, true
				case "ceil":
					return &Integer{Value: -floorDiv(-v, p) * p}, true
				case "truncate":
					return &Integer{Value: v / p * p}, true
				}
				return &Integer{Value: int64(math.Round(float64(v)/float64(p))) * p}, true
			}
		}
		return n, true
	case "to_f":
		return &Float{Value: float64(v)}, true
	case "to_r", "to_c":
		return n, true
	case "chr":
		return &String{Value: string(rune(v))}, true
	case "even?":
		return boolObj(v%2 == 0), true
	case "odd?":
		return boolObj(v%2 != 0), true
	case "zero?":
		return boolObj(v == 0), true
	case "positive?":
		return boolObj(v > 0), true
	case "negative?":
		return boolObj(v < 0), true
	case "nonzero?":
		if v == 0 {
			return &Nil{}, true
		}
		return n, true
	case "succ", "next":
		return &Integer{Value: v + 1}, true
	case "pred":
		return &Integer{Value: v - 1}, true
	case "abs", "magnitude":
		if v < 0 {
			return &Integer{Value: -v}, true
		}
		return n, true
	case "-@":
		return &Integer{Value: -v}, true
	case "+@":
		return n, true
	case "integer?", "finite?":
		return boolObj(true), true
	case "nan?", "infinite?":
		if name == "infinite?" {
			return &Nil{}, true
		}
		return boolObj(false), true
	case "divmod":
		if len(args) != 1 {
			return newError("wrong number of arguments"), true
		}
		if o, ok := args[0].(*Integer); ok {
			if o.Value == 0 {
				return newError("divided by 0"), true
			}
			return &Array{Elements: []Object{&Integer{Value: floorDiv(v, o.Value)}, &Integer{Value: floorMod(v, o.Value)}}}, true
		}
		if f, ok := args[0].(*Float); ok {
			q := math.Floor(float64(v) / f.Value)
			return &Array{Elements: []Object{&Float{Value: q}, &Float{Value: float64(v) - q*f.Value}}}, true
		}
	case "div":
		if o, ok := args[0].(*Integer); ok {
			if o.Value == 0 {
				return newError("divided by 0"), true
			}
			return &Integer{Value: floorDiv(v, o.Value)}, true
		}
	case "modulo":
		return i.infix("%", n, args[0]), true
	case "remainder":
		if o, ok := args[0].(*Integer); ok && o.Value != 0 {
			return &Integer{Value: v % o.Value}, true
		}
	case "fdiv":
		if f, ok := toFloat(args[0]); ok {
			return &Float{Value: float64(v) / f}, true
		}
	case "pow":
		if len(args) == 2 {
			m, _ := toInt64Strict(args[1])
			e, _ := toInt64Strict(args[0])
			if m == 0 {
				return newError("divided by 0"), true
			}
			result, base := int64(1), floorMod(v, m)
			for ; e > 0; e >>= 1 {
				if e&1 == 1 {
					result = result * base % m
				}
				base = base * base % m
			}
			return &Integer{Value: result}, true
		}
		return i.infix("**", n, args[0]), true
	case "gcd", "lcm":
		o, ok := args[0].(*Integer)
		if !ok {
			return newError("not an integer"), true
		}
		a, b := v, o.Value
		if a < 0 {
			a = -a
		}
		if b < 0 {
			b = -b
		}
		x, y := a, b
		for y != 0 {
			x, y = y, x%y
		}
		if name == "gcd" {
			return &Integer{Value: x}, true
		}
		if x == 0 {
			return &Integer{Value: 0}, true
		}
		return &Integer{Value: a / x * b}, true
	case "digits":
		x := v
		if x < 0 {
			return newError("out of domain"), true
		}
		var out []Object
		if x == 0 {
			out = append(out, &Integer{Value: 0})
		}
		for ; x > 0; x /= 10 {
			out = append(out, &Integer{Value: x % 10})
		}
		return &Array{Elements: out}, true
	case "bit_length":
		l := 0
		for x := v; x > 0; x >>= 1 {
			l++
		}
		return &Integer{Value: int64(l)}, true
	case "clamp", "between?":
		return numClamp(i, n, name, args)
	case "size":
		return &Integer{Value: 8}, true
	case "coerce":
		return &Array{Elements: []Object{args[0], n}}, true
	case "+", "-", "*", "/", "%", "**", "<", ">", "<=", ">=", "&", "|", "^", "<<", ">>", "<=>":
		if len(args) == 1 {
			return i.infix(name, n, args[0]), true
		}
	}
	return nil, false
}

func numClamp(i *Interpreter, n Object, name string, args []Object) (Object, bool) {
	if name == "between?" {
		if len(args) != 2 {
			return newError("wrong number of arguments (given %d, expected 2)", len(args)), true
		}
		lo, ok1 := i.compare(n, args[0])
		hi, ok2 := i.compare(n, args[1])
		if !ok1 || !ok2 {
			return newError("comparison failed"), true
		}
		return boolObj(lo >= 0 && hi <= 0), true
	}
	var lo, hi Object
	if len(args) == 1 {
		if r, ok := args[0].(*Range); ok {
			lo = &Integer{Value: r.Start}
			hi = &Integer{Value: r.End}
			if r.Exclusive {
				return newError("cannot clamp with an exclusive range"), true
			}
			if r.BeginLess {
				lo = nil
			}
			if r.Endless {
				hi = nil
			}
		}
	} else if len(args) == 2 {
		lo, hi = args[0], args[1]
	} else {
		return newError("wrong number of arguments (given %d, expected 1..2)", len(args)), true
	}
	if lo != nil && hi != nil {
		if c, ok := i.compare(lo, hi); ok && c > 0 {
			return newError("min argument must be less than or equal to max argument"), true
		}
	}
	if lo != nil {
		if c, ok := i.compare(n, lo); ok && c < 0 {
			return lo, true
		}
	}
	if hi != nil {
		if c, ok := i.compare(n, hi); ok && c > 0 {
			return hi, true
		}
	}
	return n, true
}

func roundHalfAwayFromZero(x float64, digits int64) float64 {
	p := math.Pow(10, float64(digits))
	r := math.Round(x*p) / p
	if math.IsInf(r, 0) || math.IsNaN(r) {
		return x
	}
	return r
}

func (i *Interpreter) floatMethod(f *Float, name string, args []Object, blk Object) (Object, bool) {
	v := f.Value
	digits, hasDigits := int64(0), len(args) > 0
	if hasDigits {
		digits, _ = toInt64Strict(args[0])
	}
	toInt := func(x float64) Object {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return newError("%s", strconv.FormatFloat(x, 'g', -1, 64))
		}
		return &Integer{Value: int64(x)}
	}
	switch name {
	case "to_s", "inspect":
		return &String{Value: f.Inspect()}, true
	case "to_f":
		return f, true
	case "to_i", "to_int", "truncate":
		if name == "truncate" && hasDigits && digits > 0 {
			p := math.Pow(10, float64(digits))
			return &Float{Value: math.Trunc(v*p) / p}, true
		}
		return toInt(math.Trunc(v)), true
	case "floor":
		if hasDigits && digits > 0 {
			p := math.Pow(10, float64(digits))
			return &Float{Value: math.Floor(v*p) / p}, true
		}
		return toInt(math.Floor(v)), true
	case "ceil":
		if hasDigits && digits > 0 {
			p := math.Pow(10, float64(digits))
			return &Float{Value: math.Ceil(v*p) / p}, true
		}
		return toInt(math.Ceil(v)), true
	case "round":
		if hasDigits {
			if h, ok := args[len(args)-1].(*Hash); ok {
				_ = h
				digits = 0
			}
			if digits > 0 {
				return &Float{Value: roundHalfAwayFromZero(v, digits)}, true
			}
			if digits < 0 {
				p := math.Pow(10, float64(-digits))
				return &Integer{Value: int64(math.Round(v/p) * p)}, true
			}
		}
		return toInt(math.Round(v)), true
	case "abs", "magnitude":
		return &Float{Value: math.Abs(v)}, true
	case "-@":
		return &Float{Value: -v}, true
	case "+@":
		return f, true
	case "nan?":
		return boolObj(math.IsNaN(v)), true
	case "infinite?":
		if math.IsInf(v, 1) {
			return &Integer{Value: 1}, true
		}
		if math.IsInf(v, -1) {
			return &Integer{Value: -1}, true
		}
		return &Nil{}, true
	case "finite?":
		return boolObj(!math.IsInf(v, 0) && !math.IsNaN(v)), true
	case "integer?":
		return boolObj(false), true
	case "zero?":
		return boolObj(v == 0), true
	case "positive?":
		return boolObj(v > 0), true
	case "negative?":
		return boolObj(v < 0), true
	case "divmod":
		if o, ok := toFloat(args[0]); ok {
			if o == 0 {
				return newError("divided by 0"), true
			}
			q := math.Floor(v / o)
			return &Array{Elements: []Object{&Float{Value: q}, &Float{Value: v - q*o}}}, true
		}
	case "fdiv", "quo":
		if o, ok := toFloat(args[0]); ok {
			return &Float{Value: v / o}, true
		}
	case "clamp", "between?":
		return numClamp(i, f, name, args)
	case "step":
		lim, ok1 := toFloat(args[0])
		step, ok2 := toFloat(args[1])
		if !ok1 || !ok2 || step == 0 {
			return newError("step can't be 0"), true
		}
		var out []Object
		for k := 0; ; k++ {
			x := v + float64(k)*step
			if (step > 0 && x > lim+1e-12) || (step < 0 && x < lim-1e-12) {
				break
			}
			out = append(out, &Float{Value: x})
		}
		if blk == nil {
			return &Array{Elements: out}, true
		}
		for _, e := range out {
			if r := i.yield(blk, e); isError(r) {
				return r, true
			}
		}
		return f, true
	case "+", "-", "*", "/", "%", "**", "<", ">", "<=", ">=", "<=>":
		if len(args) == 1 {
			return i.infix(name, f, args[0]), true
		}
	}
	return nil, false
}

// ---------------------------------------------------------------------------
// nil, booleans, symbols, regex
// ---------------------------------------------------------------------------

func (i *Interpreter) nilMethod(name string, args []Object) (Object, bool) {
	switch name {
	case "to_s":
		return &String{Value: ""}, true
	case "to_a":
		return &Array{}, true
	case "to_h":
		return &Hash{Pairs: map[string]Object{}}, true
	case "to_i":
		return &Integer{Value: 0}, true
	case "to_f":
		return &Float{Value: 0}, true
	case "inspect":
		return &String{Value: "nil"}, true
	case "&":
		return boolObj(false), true
	case "|":
		return boolObj(isTruthy(args[0])), true
	}
	return nil, false
}

func (i *Interpreter) boolMethod(b *Boolean, name string, args []Object) (Object, bool) {
	switch name {
	case "to_s", "inspect":
		return &String{Value: strconv.FormatBool(b.Value)}, true
	case "&":
		return boolObj(b.Value && isTruthy(args[0])), true
	case "|":
		return boolObj(b.Value || isTruthy(args[0])), true
	case "^":
		return boolObj(b.Value != isTruthy(args[0])), true
	}
	return nil, false
}

func (i *Interpreter) symbolMethod(s *Symbol, name string, args []Object, blk Object) (Object, bool) {
	switch name {
	case "to_s", "name", "id2name":
		return &String{Value: s.Name}, true
	case "to_sym":
		return s, true
	case "inspect":
		return &String{Value: s.Inspect()}, true
	case "to_proc":
		b, _ := i.toBlock(s)
		return b, true
	case "<=>":
		if o, ok := args[0].(*Symbol); ok {
			return &Integer{Value: int64(strings.Compare(s.Name, o.Name))}, true
		}
		return &Nil{}, true
	case "length", "size", "upcase", "downcase", "capitalize", "swapcase", "succ", "start_with?", "end_with?", "[]", "empty?":
		res, ok := i.stringMethod(&String{Value: s.Name}, name, args, blk)
		if str, isStr := res.(*String); isStr && ok && (name == "upcase" || name == "downcase" || name == "capitalize" || name == "swapcase" || name == "succ") {
			return &Symbol{Name: str.Value}, true
		}
		return res, ok
	}
	return nil, false
}

func (i *Interpreter) regexMethod(r *RegexObject, name string, args []Object, blk Object) (Object, bool) {
	switch name {
	case "match", "match?", "=~", "===":
		if len(args) == 0 {
			return newError("wrong number of arguments"), true
		}
		str, ok := args[0].(*String)
		if !ok {
			if sym, isSym := args[0].(*Symbol); isSym {
				str = &String{Value: sym.Name}
			} else {
				if name == "match?" || name == "===" {
					return boolObj(false), true
				}
				return &Nil{}, true
			}
		}
		switch name {
		case "match?", "===":
			return boolObj(r.Re.MatchString(str.Value)), true
		case "=~":
			return i.matchOp(str, r), true
		}
		return i.regexMatch(r, str.Value), true
	case "source":
		return &String{Value: r.Pattern}, true
	case "to_s", "inspect":
		return &String{Value: r.Inspect()}, true
	case "names":
		var out []Object
		for _, n := range r.Re.SubexpNames() {
			if n != "" {
				out = append(out, &String{Value: n})
			}
		}
		return &Array{Elements: out}, true
	}
	return nil, false
}

// regexMatch returns a MatchData-like hash for the first match, or nil.
func (i *Interpreter) regexMatch(r *RegexObject, s string) Object {
	i.setMatchGlobals(r, s)
	m := r.Re.FindStringSubmatch(s)
	if m == nil {
		return &Nil{}
	}
	return i.matchData(r, m)
}

func builtinResponds(recv Object, name string) bool {
	probe := map[string]bool{
		"to_s": true, "inspect": true, "class": true, "is_a?": true, "nil?": true, "respond_to?": true,
		"send": true, "freeze": true, "frozen?": true, "dup": true, "clone": true, "tap": true, "then": true,
		"hash": true, "object_id": true, "itself": true, "==": true, "!=": true, "method": true,
	}
	if probe[name] {
		return true
	}
	var names []string
	switch recv.(type) {
	case *Integer:
		names = intNames
	case *Float:
		names = floatNames
	case *String:
		names = stringNames
	case *Array:
		names = arrayNames
	case *Hash:
		for k := range hashMethodNames {
			if k == name {
				return true
			}
		}
	case *Range:
		names = rangeNames
	case *Function, *Builtin, *BoundMethod:
		names = procNames
	case *Symbol:
		names = []string{"to_proc", "to_sym", "length", "size", "upcase", "downcase", "<=>"}
	}
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

var intNames = strings.Fields("times upto downto step to_i to_f to_s chr even? odd? zero? positive? negative? succ next pred abs divmod pow gcd lcm digits clamp between? floor ceil round truncate + - * / % ** < > <= >= <=>")
var floatNames = strings.Fields("to_i to_f to_s floor ceil round truncate abs nan? infinite? finite? zero? positive? negative? divmod clamp between? + - * / % ** < > <= >= <=>")
var rangeNames = strings.Fields("each to_a map select filter reject reduce inject sum min max first last size count include? cover? step reverse_each each_slice each_with_index any? all? none? find sort sort_by group_by partition zip take drop begin end")
var procNames = strings.Fields("call () yield [] to_proc arity curry lambda? parameters >> <<")

var blockTakers = func() map[string]bool {
	m := map[string]bool{}
	for _, n := range strings.Fields("each map collect flat_map select filter reject find detect find_all each_with_index filter_map group_by partition sort_by min_by max_by take_while drop_while each_slice each_cons each_with_object any? all? none? count sum uniq each_key each_value each_pair map_values transform_values transform_keys") {
		m[n] = true
	}
	return m
}()
