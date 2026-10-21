package interpreter

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/vibe-lang/vibe/pkg/ast"
)

func newRegex(pattern, flags string) (*RegexObject, error) {
	prefix := ""
	for _, f := range flags {
		switch f {
		case 'i':
			prefix += "i"
		case 'm':
			prefix += "s"
		case 'x':
			pattern = stripExtended(pattern)
		}
	}
	src := pattern
	if prefix != "" {
		src = "(?" + prefix + ")" + pattern
	}
	re, err := regexp.Compile(src)
	if err != nil {
		return nil, err
	}
	return &RegexObject{Re: re, Pattern: pattern, Flags: flags}, nil
}

func stripExtended(p string) string {
	var out strings.Builder
	for _, line := range strings.Split(p, "\n") {
		if idx := strings.Index(line, " #"); idx >= 0 {
			line = line[:idx]
		}
		out.WriteString(strings.Join(strings.Fields(line), ""))
	}
	return out.String()
}

func (i *Interpreter) evalRegexLiteral(node *ast.RegexLiteral) Object {
	r, err := newRegex(node.Pattern, node.Flags)
	if err != nil {
		return newError("invalid regex /%s/: %s", node.Pattern, err.Error())
	}
	return r
}

// equal implements == with Ruby semantics.
func (i *Interpreter) equal(a, b Object) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch av := a.(type) {
	case *Integer:
		switch bv := b.(type) {
		case *Integer:
			return av.Value == bv.Value
		case *Float:
			return float64(av.Value) == bv.Value
		}
		return false
	case *Float:
		switch bv := b.(type) {
		case *Integer:
			return av.Value == float64(bv.Value)
		case *Float:
			return av.Value == bv.Value
		}
		return false
	case *String:
		bv, ok := b.(*String)
		return ok && av.Value == bv.Value
	case *Symbol:
		bv, ok := b.(*Symbol)
		return ok && av.Name == bv.Name
	case *Boolean:
		bv, ok := b.(*Boolean)
		return ok && av.Value == bv.Value
	case *Nil:
		_, ok := b.(*Nil)
		return ok
	case *Array:
		bv, ok := b.(*Array)
		if !ok || len(av.Elements) != len(bv.Elements) {
			return false
		}
		for idx := range av.Elements {
			if !i.equal(av.Elements[idx], bv.Elements[idx]) {
				return false
			}
		}
		return true
	case *Hash:
		bv, ok := b.(*Hash)
		if !ok || len(av.Pairs) != len(bv.Pairs) {
			return false
		}
		for k, v := range av.Pairs {
			o, ok := bv.Pairs[k]
			if !ok || !i.equal(v, o) {
				return false
			}
		}
		return true
	case *Range:
		bv, ok := b.(*Range)
		return ok && *av == *bv
	case *ClassInstance:
		if m := av.Class.findMethod("=="); m != nil {
			return isTruthy(i.callMethodWithBlock(av, m.owner, m.fn, []Object{b}, nil))
		}
		if m := av.Class.findMethod("<=>"); m != nil && av.Class.findNative("==") != nil {
			r := i.callMethodWithBlock(av, m.owner, m.fn, []Object{b}, nil)
			n, ok := r.(*Integer)
			return ok && n.Value == 0
		}
		return false
	case *StructInstance:
		bv, ok := b.(*StructInstance)
		if !ok || av.Struct != bv.Struct || len(av.Fields) != len(bv.Fields) {
			return false
		}
		for k, v := range av.Fields {
			if !i.equal(v, bv.Fields[k]) {
				return false
			}
		}
		return true
	case *ClassObject:
		return false
	}
	return a.Type() == b.Type() && a.Inspect() == b.Inspect()
}

func objectsEqual(a, b Object) bool {
	if a == nil || b == nil {
		return a == b
	}
	return getBuiltinInterpreter().equal(a, b)
}

func toFloat(o Object) (float64, bool) {
	switch v := o.(type) {
	case *Integer:
		return float64(v.Value), true
	case *Float:
		return v.Value, true
	}
	return 0, false
}

// compare implements <=> for comparable values; ok is false when they are not comparable.
func (i *Interpreter) compare(a, b Object) (int, bool) {
	if x, ok := toFloat(a); ok {
		if y, ok := toFloat(b); ok {
			if ai, ok := a.(*Integer); ok {
				if bi, ok := b.(*Integer); ok {
					switch {
					case ai.Value < bi.Value:
						return -1, true
					case ai.Value > bi.Value:
						return 1, true
					}
					return 0, true
				}
			}
			switch {
			case x < y:
				return -1, true
			case x > y:
				return 1, true
			case x == y:
				return 0, true
			}
		}
		return 0, false
	}
	switch av := a.(type) {
	case *String:
		if bv, ok := b.(*String); ok {
			return strings.Compare(av.Value, bv.Value), true
		}
	case *Symbol:
		if bv, ok := b.(*Symbol); ok {
			return strings.Compare(av.Name, bv.Name), true
		}
	case *Array:
		if bv, ok := b.(*Array); ok {
			for idx := 0; idx < len(av.Elements) && idx < len(bv.Elements); idx++ {
				c, ok := i.compare(av.Elements[idx], bv.Elements[idx])
				if !ok {
					return 0, false
				}
				if c != 0 {
					return c, true
				}
			}
			switch {
			case len(av.Elements) < len(bv.Elements):
				return -1, true
			case len(av.Elements) > len(bv.Elements):
				return 1, true
			}
			return 0, true
		}
	case *Boolean:
		if bv, ok := b.(*Boolean); ok && av.Value == bv.Value {
			return 0, true
		}
	case *Nil:
		if _, ok := b.(*Nil); ok {
			return 0, true
		}
	case *ClassInstance:
		if m := av.Class.findMethod("<=>"); m != nil {
			r := i.callMethodWithBlock(av, m.owner, m.fn, []Object{b}, nil)
			if n, ok := r.(*Integer); ok {
				return int(n.Value), true
			}
		}
	}
	return 0, false
}

func boolObj(b bool) Object { return &Boolean{Value: b} }

// evalInfixExpression evaluates binary operators.
func (i *Interpreter) evalInfixExpression(node *ast.InfixExpression) Object {
	switch node.Operator {
	case "&&":
		left := i.eval(node.Left)
		if isError(left) || !isTruthy(left) {
			return left
		}
		return i.eval(node.Right)
	case "||":
		left := i.eval(node.Left)
		if isError(left) || isTruthy(left) {
			return left
		}
		return i.eval(node.Right)
	}

	left := i.eval(node.Left)
	if isError(left) {
		return left
	}
	right := i.eval(node.Right)
	if isError(right) {
		return right
	}
	return i.infix(node.Operator, left, right)
}

// infix applies a binary operator to two evaluated operands.
func (i *Interpreter) infix(op string, left, right Object) Object {
	// User-defined operators on objects take priority.
	switch l := left.(type) {
	case *ClassInstance:
		if res, ok := i.userOperator(l, op, right); ok {
			return res
		}
	}

	if (op == ">>" || op == "<<") && isCallable(left) && isCallable(right) {
		res, _ := i.procMethod(left, op, []Object{right}, nil)
		return res
	}

	switch op {
	case "==":
		return boolObj(i.equal(left, right))
	case "!=":
		if ci, ok := left.(*ClassInstance); ok {
			if m := ci.Class.findMethod("!="); m != nil {
				return i.callMethodWithBlock(ci, m.owner, m.fn, []Object{right}, nil)
			}
		}
		return boolObj(!i.equal(left, right))
	case "<=>":
		if c, ok := i.compare(left, right); ok {
			return &Integer{Value: int64(c)}
		}
		return &Nil{}
	case "===":
		return i.caseEqual(left, right)
	case "=~":
		return i.matchOp(left, right)
	case "!~":
		return boolObj(!isTruthy(i.matchOp(left, right)))
	}

	switch l := left.(type) {
	case *Integer:
		switch r := right.(type) {
		case *Integer:
			return i.intInfix(op, l.Value, r.Value)
		case *Float:
			return i.floatInfix(op, float64(l.Value), r.Value)
		}
	case *Float:
		if r, ok := toFloat(right); ok {
			return i.floatInfix(op, l.Value, r)
		}
	case *String:
		if res, ok := i.stringInfix(op, l, right); ok {
			return res
		}
	case *Array:
		if res, ok := i.arrayInfix(op, l, right); ok {
			return res
		}
	case *Hash:
		if op == "+" {
			if r, ok := right.(*Hash); ok {
				return hashMerge(l, r)
			}
		}
	case *Symbol:
		if r, ok := right.(*Symbol); ok {
			if c, ok := i.compare(l, r); ok {
				return compareResult(op, c)
			}
		}
	case *Boolean:
		if r, ok := right.(*Boolean); ok {
			switch op {
			case "&":
				return boolObj(l.Value && r.Value)
			case "|":
				return boolObj(l.Value || r.Value)
			case "^":
				return boolObj(l.Value != r.Value)
			}
		}
	case *Nil:
		if op == "&" {
			return boolObj(false)
		}
	case *Range:
		if op == "==" {
			return boolObj(i.equal(left, right))
		}
	}

	if op == "+" {
		if s, ok := left.(*String); ok {
			return &String{Value: s.Value + i.display(right)}
		}
	}
	if c, ok := i.compareOp(op, left, right); ok {
		return c
	}
	return newError("unknown operator: %s for types %s and %s", op, left.Type(), right.Type())
}

func compareResult(op string, c int) Object {
	switch op {
	case "<":
		return boolObj(c < 0)
	case ">":
		return boolObj(c > 0)
	case "<=":
		return boolObj(c <= 0)
	case ">=":
		return boolObj(c >= 0)
	}
	return nil
}

func (i *Interpreter) compareOp(op string, left, right Object) (Object, bool) {
	switch op {
	case "<", ">", "<=", ">=":
		c, ok := i.compare(left, right)
		if !ok {
			return nil, false
		}
		return compareResult(op, c), true
	}
	return nil, false
}

// userOperator dispatches a binary operator to a method defined on the instance's class.
func (i *Interpreter) userOperator(l *ClassInstance, op string, right Object) (Object, bool) {
	if m := l.Class.findMethod(op); m != nil {
		return i.callMethodWithBlock(l, m.owner, m.fn, []Object{right}, nil), true
	}
	if nat := l.Class.findNative(op); nat != nil {
		return nat(i, l, []Object{right}, nil), true
	}
	return nil, false
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func floorMod(a, b int64) int64 {
	m := a % b
	if m != 0 && ((m < 0) != (b < 0)) {
		m += b
	}
	return m
}

func intPow(base, exp int64) Object {
	if exp < 0 {
		return &Float{Value: math.Pow(float64(base), float64(exp))}
	}
	result := int64(1)
	b := base
	e := exp
	overflow := false
	for e > 0 {
		if e&1 == 1 {
			r := result * b
			if b != 0 && r/b != result {
				overflow = true
				break
			}
			result = r
		}
		e >>= 1
		if e > 0 {
			nb := b * b
			if b != 0 && nb/b != b {
				overflow = true
				break
			}
			b = nb
		}
	}
	if overflow {
		return &Float{Value: math.Pow(float64(base), float64(exp))}
	}
	return &Integer{Value: result}
}

func (i *Interpreter) intInfix(op string, a, b int64) Object {
	switch op {
	case "+":
		return &Integer{Value: a + b}
	case "-":
		return &Integer{Value: a - b}
	case "*":
		return &Integer{Value: a * b}
	case "/":
		if b == 0 {
			return newError("division by zero")
		}
		return &Integer{Value: floorDiv(a, b)}
	case "%":
		if b == 0 {
			return newError("division by zero")
		}
		return &Integer{Value: floorMod(a, b)}
	case "**":
		return intPow(a, b)
	case "<":
		return boolObj(a < b)
	case ">":
		return boolObj(a > b)
	case "<=":
		return boolObj(a <= b)
	case ">=":
		return boolObj(a >= b)
	case "&":
		return &Integer{Value: a & b}
	case "|":
		return &Integer{Value: a | b}
	case "^":
		return &Integer{Value: a ^ b}
	case "<<":
		if b < 0 {
			return &Integer{Value: a >> uint(-b)}
		}
		return &Integer{Value: a << uint(b)}
	case ">>":
		if b < 0 {
			return &Integer{Value: a << uint(-b)}
		}
		return &Integer{Value: a >> uint(b)}
	}
	return newError("unknown operator: INTEGER %s INTEGER", op)
}

func (i *Interpreter) floatInfix(op string, a, b float64) Object {
	switch op {
	case "+":
		return &Float{Value: a + b}
	case "-":
		return &Float{Value: a - b}
	case "*":
		return &Float{Value: a * b}
	case "/":
		if b == 0 {
			return newError("division by zero")
		}
		return &Float{Value: a / b}
	case "%":
		if b == 0 {
			return newError("division by zero")
		}
		m := math.Mod(a, b)
		if m != 0 && (m < 0) != (b < 0) {
			m += b
		}
		return &Float{Value: m}
	case "**":
		return &Float{Value: math.Pow(a, b)}
	case "<":
		return boolObj(a < b)
	case ">":
		return boolObj(a > b)
	case "<=":
		return boolObj(a <= b)
	case ">=":
		return boolObj(a >= b)
	}
	return newError("unknown operator: FLOAT %s FLOAT", op)
}

func (i *Interpreter) stringInfix(op string, l *String, right Object) (Object, bool) {
	switch op {
	case "+":
		if r, ok := right.(*String); ok {
			return &String{Value: l.Value + r.Value}, true
		}
		return &String{Value: l.Value + i.display(right)}, true
	case "*":
		if r, ok := right.(*Integer); ok {
			if r.Value < 0 {
				return newError("string repetition count must be non-negative"), true
			}
			return &String{Value: strings.Repeat(l.Value, int(r.Value))}, true
		}
	case "<<":
		if l.Frozen {
			return i.frozenErr(l), true
		}
		switch r := right.(type) {
		case *String:
			l.Value += r.Value
		case *Integer:
			l.Value += string(rune(r.Value))
		default:
			l.Value += i.display(right)
		}
		return l, true
	case "%":
		return i.formatPercent(l.Value, right), true
	case "<", ">", "<=", ">=":
		if r, ok := right.(*String); ok {
			return compareResult(op, strings.Compare(l.Value, r.Value)), true
		}
	}
	return nil, false
}

func (i *Interpreter) arrayInfix(op string, l *Array, right Object) (Object, bool) {
	switch op {
	case "<<":
		if l.Frozen {
			return i.frozenErr(l), true
		}
		l.Elements = append(l.Elements, right)
		return l, true
	}
	r, isArr := right.(*Array)
	switch op {
	case "+":
		if isArr {
			out := make([]Object, 0, len(l.Elements)+len(r.Elements))
			out = append(out, l.Elements...)
			out = append(out, r.Elements...)
			return &Array{Elements: out}, true
		}
		return newError("no implicit conversion of %s into Array", right.Type()), true
	case "-":
		if isArr {
			var out []Object
			for _, e := range l.Elements {
				if !i.arrayHas(r, e) {
					out = append(out, e)
				}
			}
			return &Array{Elements: out}, true
		}
	case "*":
		switch n := right.(type) {
		case *Integer:
			var out []Object
			for k := int64(0); k < n.Value; k++ {
				out = append(out, l.Elements...)
			}
			return &Array{Elements: out}, true
		case *String:
			parts := make([]string, len(l.Elements))
			for idx, e := range l.Elements {
				parts[idx] = i.display(e)
			}
			return &String{Value: strings.Join(parts, n.Value)}, true
		}
	case "&":
		if isArr {
			var out []Object
			for _, e := range l.Elements {
				if i.arrayHas(r, e) && !i.arrayHas(&Array{Elements: out}, e) {
					out = append(out, e)
				}
			}
			return &Array{Elements: out}, true
		}
	case "|":
		if isArr {
			var out []Object
			for _, e := range append(append([]Object{}, l.Elements...), r.Elements...) {
				if !i.arrayHas(&Array{Elements: out}, e) {
					out = append(out, e)
				}
			}
			return &Array{Elements: out}, true
		}
	case "<", ">", "<=", ">=":
		if isArr {
			if c, ok := i.compare(l, r); ok {
				return compareResult(op, c), true
			}
		}
	}
	return nil, false
}

func (i *Interpreter) arrayHas(a *Array, e Object) bool {
	for _, x := range a.Elements {
		if i.equal(x, e) {
			return true
		}
	}
	return false
}

// caseEqual implements === (used by case/when and rescue-like matching).
func (i *Interpreter) caseEqual(pattern, v Object) Object {
	switch p := pattern.(type) {
	case *ClassObject:
		return boolObj(i.isInstanceOf(v, p))
	case *Range:
		switch n := v.(type) {
		case *Integer:
			return boolObj(p.contains(float64(n.Value)))
		case *Float:
			return boolObj(p.contains(n.Value))
		}
		return boolObj(false)
	case *RegexObject:
		switch s := v.(type) {
		case *String:
			return boolObj(p.Re.MatchString(s.Value))
		case *Symbol:
			return boolObj(p.Re.MatchString(s.Name))
		}
		return boolObj(false)
	case *Function:
		return boolObj(isTruthy(i.callFunction(p, []Object{v}, nil)))
	case *ClassInstance:
		if m := p.Class.findMethod("==="); m != nil {
			return i.callMethodWithBlock(p, m.owner, m.fn, []Object{v}, nil)
		}
	}
	return boolObj(i.equal(pattern, v))
}

func (r *Range) contains(x float64) bool {
	if r.Endless {
		return x >= float64(r.Start)
	}
	if r.BeginLess {
		if r.Exclusive {
			return x < float64(r.End)
		}
		return x <= float64(r.End)
	}
	lo, hi := float64(r.Start), float64(r.End)
	if x < lo {
		return false
	}
	if r.Exclusive {
		return x < hi
	}
	return x <= hi
}

func (i *Interpreter) matchOp(left, right Object) Object {
	var re *RegexObject
	var s string
	switch l := left.(type) {
	case *String:
		s = l.Value
		re, _ = right.(*RegexObject)
	case *RegexObject:
		re = l
		if rs, ok := right.(*String); ok {
			s = rs.Value
		} else {
			return &Nil{}
		}
	default:
		return &Nil{}
	}
	if re == nil {
		return newError("wrong argument type %s (expected Regexp)", right.Type())
	}
	loc := re.Re.FindStringIndex(s)
	i.setMatchGlobals(re, s)
	if loc == nil {
		return &Nil{}
	}
	return &Integer{Value: int64(len([]rune(s[:loc[0]])))}
}

// setMatchGlobals records $~ and $1..$9 after a match.
func (i *Interpreter) setMatchGlobals(re *RegexObject, s string) {
	root := i.rootEnv()
	m := re.Re.FindStringSubmatch(s)
	for n := 1; n <= 9; n++ {
		name := "$" + strconv.Itoa(n)
		if m != nil && n < len(m) {
			root.Set(name, &String{Value: m[n]})
		} else {
			root.Set(name, &Nil{})
		}
	}
	if m == nil {
		root.Set("$~", &Nil{})
		return
	}
	root.Set("$~", i.matchData(re, m))
}

func (i *Interpreter) matchData(re *RegexObject, m []string) Object {
	h := &Hash{Pairs: map[string]Object{}}
	names := re.Re.SubexpNames()
	for idx, v := range m {
		key := strconv.Itoa(idx)
		if idx < len(names) && names[idx] != "" {
			key = names[idx]
		}
		h.set(key, &String{Value: key}, &String{Value: v})
	}
	return h
}

// formatPercent implements String#% (format).
func (i *Interpreter) formatPercent(f string, arg Object) Object {
	var args []Object
	if arr, ok := arg.(*Array); ok {
		args = arr.Elements
	} else {
		args = []Object{arg}
	}
	return i.sprintf(f, args)
}

var fmtDirective = regexp.MustCompile(`%(?:%|\{(\w+)\}|<(\w+)>)?([-+ 0#]*)(\d+|\*)?(?:\.(\d+))?([sdifgeExXobBcpu])?`)

func (i *Interpreter) sprintf(f string, args []Object) Object {
	var out strings.Builder
	argIdx := 0
	next := func() (Object, bool) {
		if argIdx < len(args) {
			a := args[argIdx]
			argIdx++
			return a, true
		}
		return nil, false
	}
	var hash *Hash
	if len(args) == 1 {
		hash, _ = args[0].(*Hash)
	}
	pos := 0
	for pos < len(f) {
		idx := strings.IndexByte(f[pos:], '%')
		if idx < 0 {
			out.WriteString(f[pos:])
			break
		}
		out.WriteString(f[pos : pos+idx])
		pos += idx
		loc := fmtDirective.FindStringSubmatchIndex(f[pos:])
		if loc == nil || loc[0] != 0 {
			out.WriteByte('%')
			pos++
			continue
		}
		base := pos
		whole := f[base : base+loc[1]]
		sub := func(n int) string {
			if loc[2*n] < 0 {
				return ""
			}
			return f[base+loc[2*n] : base+loc[2*n+1]]
		}
		pos += loc[1]
		if whole == "%%" {
			out.WriteByte('%')
			continue
		}
		name, named, flags, width, prec, verb := sub(1), sub(2), sub(3), sub(4), sub(5), sub(6)
		var val Object
		switch {
		case name != "":
			if hash == nil {
				return newError("one hash required")
			}
			v, ok := hash.Pairs[name]
			if !ok {
				return newError("key<%s> not found", name)
			}
			out.WriteString(i.display(v))
			continue
		case named != "":
			if hash == nil {
				return newError("one hash required")
			}
			v, ok := hash.Pairs[named]
			if !ok {
				return newError("key<%s> not found", named)
			}
			val = v
		default:
			if verb == "" {
				out.WriteString(whole)
				continue
			}
			if width == "*" {
				w, _ := next()
				if wi, ok := w.(*Integer); ok {
					width = strconv.Itoa(int(wi.Value))
				} else {
					width = ""
				}
			}
			v, ok := next()
			if !ok {
				return newError("too few arguments")
			}
			val = v
		}
		if verb == "" {
			verb = "s"
		}
		spec := "%" + flags + width
		if prec != "" {
			spec += "." + prec
		}
		switch verb {
		case "d", "i", "u":
			n, ok := toInt64(val)
			if !ok {
				return newError("invalid value for Integer(): %s", val.Inspect())
			}
			out.WriteString(fmt.Sprintf(spec+"d", n))
		case "f", "e", "E", "g":
			x, ok := toFloat(val)
			if !ok {
				if s, isStr := val.(*String); isStr {
					if p, err := strconv.ParseFloat(s.Value, 64); err == nil {
						x, ok = p, true
					}
				}
			}
			if !ok {
				return newError("invalid value for Float(): %s", val.Inspect())
			}
			out.WriteString(fmt.Sprintf(spec+verb, x))
		case "x", "X", "o":
			n, ok := toInt64(val)
			if !ok {
				return newError("invalid value for Integer(): %s", val.Inspect())
			}
			out.WriteString(fmt.Sprintf(spec+verb, n))
		case "b", "B":
			n, ok := toInt64(val)
			if !ok {
				return newError("invalid value for Integer(): %s", val.Inspect())
			}
			out.WriteString(fmt.Sprintf(spec+"b", n))
		case "c":
			if n, ok := val.(*Integer); ok {
				out.WriteString(string(rune(n.Value)))
			} else {
				out.WriteString(i.display(val))
			}
		case "p":
			out.WriteString(fmt.Sprintf(spec+"s", i.inspectValue(val)))
		default:
			out.WriteString(fmt.Sprintf(spec+"s", i.display(val)))
		}
	}
	return &String{Value: out.String()}
}

func toInt64(o Object) (int64, bool) {
	switch v := o.(type) {
	case *Integer:
		return v.Value, true
	case *Float:
		return int64(math.Floor(v.Value)), true
	case *String:
		n, err := strconv.ParseInt(strings.TrimSpace(v.Value), 0, 64)
		return n, err == nil
	}
	return 0, false
}

func hashMerge(a, b *Hash) *Hash {
	out := &Hash{Pairs: map[string]Object{}, Default: a.Default}
	for _, k := range a.Order {
		out.setKey(k, a.keyObj(k), a.Pairs[k])
	}
	for _, k := range b.Order {
		out.setKey(k, b.keyObj(k), b.Pairs[k])
	}
	return out
}

// set inserts or replaces a key, preserving insertion order.
func (h *Hash) set(k string, key Object, v Object) { h.setKey(k, key, v) }

func (h *Hash) setKey(k string, key Object, v Object) {
	if h.Pairs == nil {
		h.Pairs = map[string]Object{}
	}
	if _, ok := h.Pairs[k]; !ok {
		h.Order = append(h.Order, k)
		if _, isStr := key.(*String); !isStr && key != nil {
			if h.Keys == nil {
				h.Keys = map[string]Object{}
			}
			h.Keys[k] = key
		}
	}
	h.Pairs[k] = v
}

// keyObj returns the original key object for an internal key string.
func (h *Hash) keyObj(k string) Object {
	if o, ok := h.Keys[k]; ok {
		return o
	}
	return &String{Value: k}
}

func (h *Hash) remove(k string) {
	if _, ok := h.Pairs[k]; !ok {
		return
	}
	delete(h.Pairs, k)
	delete(h.Keys, k)
	for idx, o := range h.Order {
		if o == k {
			h.Order = append(h.Order[:idx], h.Order[idx+1:]...)
			break
		}
	}
}

// evalPrefixExpression evaluates a prefix expression like -5 or !true.
func (i *Interpreter) evalPrefixExpression(node *ast.PrefixExpression) Object {
	right := i.eval(node.Right)
	if isError(right) {
		return right
	}
	return i.prefix(node.Operator, right)
}

func (i *Interpreter) prefix(op string, right Object) Object {
	if ci, ok := right.(*ClassInstance); ok {
		name := op + "@"
		if op == "!" {
			name = "!"
		}
		if m := ci.Class.findMethod(name); m != nil {
			return i.callMethodWithBlock(ci, m.owner, m.fn, nil, nil)
		}
	}
	switch op {
	case "!":
		return boolObj(!isTruthy(right))
	case "-":
		switch obj := right.(type) {
		case *Integer:
			return &Integer{Value: -obj.Value}
		case *Float:
			return &Float{Value: -obj.Value}
		}
	case "+":
		switch right.(type) {
		case *Integer, *Float:
			return right
		}
	case "~":
		if obj, ok := right.(*Integer); ok {
			return &Integer{Value: ^obj.Value}
		}
	}
	return newError("unknown operator: %s%s", op, right.Type())
}

func (i *Interpreter) evalBangOperator(right Object) Object { return boolObj(!isTruthy(right)) }
