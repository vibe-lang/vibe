package interpreter

import (
	"strings"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
)

// currentClass returns the class whose body or method is currently running.
func (i *Interpreter) currentClass() *ClassObject {
	for _, key := range []string{"__defclass__", "__current_class__"} {
		if v, ok := i.env.Get(key); ok {
			if c, ok := v.(*ClassObject); ok {
				return c
			}
		}
	}
	if s, ok := i.env.Get("self"); ok {
		switch o := s.(type) {
		case *ClassInstance:
			return o.Class
		case *ClassObject:
			return o
		}
	}
	return nil
}

func (i *Interpreter) getCvar(name string) Object {
	c := i.currentClass()
	if c == nil {
		return newError("class variable %s accessed from toplevel", name)
	}
	if owner := c.findCvarOwner(name); owner != nil {
		return owner.Cvars[name]
	}
	return newError("uninitialized class variable %s in %s", name, c.Name)
}

func (i *Interpreter) setCvar(name string, v Object) Object {
	c := i.currentClass()
	if c == nil {
		return newError("class variable %s accessed from toplevel", name)
	}
	if owner := c.findCvarOwner(name); owner != nil {
		owner.Cvars[name] = v
	} else {
		c.Cvars[name] = v
	}
	return v
}

// lookupConstant resolves a capitalised name through the lexical class and globals.
func (i *Interpreter) lookupConstant(name string) Object {
	if c := i.currentClass(); c != nil {
		for k := c; k != nil; k = k.Parent {
			if v, ok := k.findConst(name); ok {
				return v
			}
		}
	}
	return nil
}

func (i *Interpreter) evalReturn(node *ast.ReturnStatement) Object {
	var val Object = &Nil{}
	if node.Value != nil {
		val = i.eval(node.Value)
		if isError(val) {
			return val
		}
	}
	return &ReturnValue{Value: val, Next: node.Token.Type == lexer.NEXT}
}

func (i *Interpreter) evalBreak(node *ast.BreakStatement) Object {
	bs := &BreakSignal{}
	if node.Value != nil {
		v := i.eval(node.Value)
		if isError(v) {
			return v
		}
		bs.Value = v
	}
	return bs
}

// loopControl interprets the result of a loop body: it reports whether the loop
// should stop, and the value/signal the loop should produce when it does.
func loopControl(res Object) (stop bool, out Object) {
	switch r := res.(type) {
	case *BreakSignal:
		if r.Value != nil {
			return true, r.Value
		}
		return true, &Nil{}
	case *ContinueSignal:
		return false, nil
	case *ReturnValue:
		if r.Next {
			return false, nil
		}
		return true, r
	case *Error, *ThrowSignal, *RetrySignal:
		return true, r
	}
	return false, nil
}

func (i *Interpreter) evalWhileLoop(node *ast.WhileLoop) Object {
	return i.runLoop(node.Condition, node.Body, false, false)
}

func (i *Interpreter) evalUntilLoop(node *ast.UntilLoop) Object {
	return i.runLoop(node.Condition, node.Body, true, false)
}

func (i *Interpreter) runLoop(cond ast.Expression, body *ast.BlockStatement, until, doWhile bool) Object {
	first := doWhile
	for {
		if !first {
			c := i.eval(cond)
			if isError(c) {
				return c
			}
			if isTruthy(c) == until {
				return &Nil{}
			}
		}
		first = false
		res := i.evalBlockStatement(body)
		if stop, out := loopControl(res); stop {
			return out
		}
	}
}

func (i *Interpreter) evalPostfixLoop(node *ast.PostfixLoop) Object {
	first := node.DoWhile
	for {
		if !first {
			c := i.eval(node.Condition)
			if isError(c) {
				return c
			}
			if isTruthy(c) == node.Until {
				return &Nil{}
			}
		}
		first = false
		res := i.eval(node.Statement)
		if stop, out := loopControl(res); stop {
			return out
		}
	}
}

func (i *Interpreter) evalHashLiteral(node *ast.HashLiteral) Object {
	h := &Hash{Pairs: map[string]Object{}}
	if len(node.Entries) == 0 && len(node.Pairs) > 0 {
		for k, v := range node.Pairs {
			node.Entries = append(node.Entries, ast.HashEntry{Key: k, Value: v})
		}
	}
	for _, e := range node.Entries {
		if e.Splat {
			v := i.eval(e.Value)
			if isError(v) {
				return v
			}
			switch src := v.(type) {
			case *Hash:
				for _, k := range src.Order {
					h.set(k, src.keyObj(k), src.Pairs[k])
				}
			case *Nil:
			default:
				return newError("no implicit conversion of %s into Hash", v.Type())
			}
			continue
		}
		key := i.eval(e.Key)
		if isError(key) {
			return key
		}
		val := i.eval(e.Value)
		if isError(val) {
			return val
		}
		h.set(hashKey(key), key, val)
	}
	return h
}

func (i *Interpreter) evalIndexAssignment(node *ast.IndexAssignment) Object {
	left := i.eval(node.Left)
	if isError(left) {
		return left
	}
	index := i.eval(node.Index)
	if isError(index) {
		return index
	}
	value := i.eval(node.Value)
	if isError(value) {
		return value
	}
	return i.indexSet(left, index, value)
}

func (i *Interpreter) indexSet(left, index, value Object) Object {
	switch obj := left.(type) {
	case *Array:
		if frozen(obj) {
			return newError("can't modify frozen Array")
		}
		if r, ok := index.(*Range); ok {
			start, length, ok := rangeBounds(r, len(obj.Elements))
			if !ok {
				return newError("range %s out of range", r.Inspect())
			}
			var repl []Object
			if arr, isArr := value.(*Array); isArr {
				repl = arr.Elements
			} else {
				repl = []Object{value}
			}
			if start > len(obj.Elements) {
				for len(obj.Elements) < start {
					obj.Elements = append(obj.Elements, &Nil{})
				}
			}
			end := start + length
			if end > len(obj.Elements) {
				end = len(obj.Elements)
			}
			out := append([]Object{}, obj.Elements[:start]...)
			out = append(out, repl...)
			out = append(out, obj.Elements[end:]...)
			obj.Elements = out
			return value
		}
		idx, ok := index.(*Integer)
		if !ok {
			return newError("array index must be an integer, got %s", index.Type())
		}
		n := idx.Value
		if n < 0 {
			n += int64(len(obj.Elements))
		}
		if n < 0 {
			return newError("array index out of bounds: %d", idx.Value)
		}
		for int64(len(obj.Elements)) <= n {
			obj.Elements = append(obj.Elements, &Nil{})
		}
		obj.Elements[n] = value
		return value
	case *Hash:
		if obj.Frozen {
			return newError("can't modify frozen Hash")
		}
		obj.set(hashKey(index), index, value)
		return value
	case *String:
		rs := []rune(obj.Value)
		sub, ok := value.(*String)
		if !ok {
			return newError("no implicit conversion of %s into String", value.Type())
		}
		switch ix := index.(type) {
		case *Integer:
			n := int(ix.Value)
			if n < 0 {
				n += len(rs)
			}
			if n < 0 || n >= len(rs) {
				return newError("index %d out of string", ix.Value)
			}
			obj.Value = string(rs[:n]) + sub.Value + string(rs[n+1:])
		case *Range:
			start, length, ok := rangeBounds(ix, len(rs))
			if !ok {
				return newError("range %s out of string", ix.Inspect())
			}
			end := start + length
			if end > len(rs) {
				end = len(rs)
			}
			obj.Value = string(rs[:start]) + sub.Value + string(rs[end:])
		case *String:
			if !strings.Contains(obj.Value, ix.Value) {
				return newError("string not matched")
			}
			obj.Value = strings.Replace(obj.Value, ix.Value, sub.Value, 1)
		default:
			return newError("string index must be an integer, got %s", index.Type())
		}
		return value
	case *ClassInstance:
		if m := obj.Class.findMethod("[]="); m != nil {
			if r := i.callMethodWithBlock(obj, m.owner, m.fn, []Object{index, value}, nil); isError(r) {
				return r
			}
			return value
		}
	case *StructInstance:
		if k, ok := index.(*String); ok {
			return i.assignMember(obj, k.Value, value)
		}
		if k, ok := index.(*Symbol); ok {
			return i.assignMember(obj, k.Name, value)
		}
	}
	return newError("index assignment not supported for %s", left.Type())
}

func frozen(a *Array) bool { return a.Frozen }

// rangeBounds converts a range into a start offset and length over a sequence of n items.
func rangeBounds(r *Range, n int) (start, length int, ok bool) {
	s := int(r.Start)
	if r.BeginLess {
		s = 0
	}
	e := int(r.End)
	if r.Endless {
		e = n - 1
	} else if r.Exclusive {
		e--
	}
	if s < 0 {
		s += n
	}
	if e < 0 {
		e += n
	}
	if s < 0 || s > n {
		return 0, 0, false
	}
	if e >= n {
		e = n - 1
	}
	length = e - s + 1
	if length < 0 {
		length = 0
	}
	return s, length, true
}

func (i *Interpreter) evalArrowFunction(node *ast.ArrowFunction) Object {
	fn := &Function{
		Parameters: node.Parameters,
		ParamTypes: node.ParamTypes,
		Defaults:   node.ParamDefaults,
		Variadic:   node.Variadic,
		Keyword:    node.Keyword,
		KwRest:     node.KwRest,
		BlockParam: node.BlockParam,
		Body:       node.Body,
		Env:        i.env,
		ReturnType: node.ReturnType,
		IsBlock:    node.IsBlock,
		IsLambda:   !node.IsBlock,
	}
	if node.IsBlock {
		var frame *Frame
		if f, ok := i.env.Get("__frame__"); ok {
			frame, _ = f.(*Frame)
		}
		fn.Frame = frame
	}
	return fn
}

func (i *Interpreter) evalPipeExpression(node *ast.PipeExpression) Object {
	left := i.eval(node.Left)
	if isError(left) {
		return left
	}
	if call, ok := node.Right.(*ast.CallExpression); ok {
		var blk Object
		if call.Block != nil {
			blk = i.evalArrowFunction(call.Block)
		}
		args, bp, err := i.evalArgs(call.Arguments)
		if err != nil {
			return err
		}
		if blk == nil {
			blk = bp
		}
		if dot, ok := call.Function.(*ast.DotExpression); ok {
			recv := i.eval(dot.Left)
			if isError(recv) {
				return recv
			}
			return i.callMember(recv, dot.Field.Value, append([]Object{left}, args...), blk, callMode{isCall: true})
		}
		fn := i.eval(call.Function)
		if isError(fn) {
			return fn
		}
		return i.callFunction(fn, append([]Object{left}, args...), blk)
	}
	fn := i.eval(node.Right)
	if isError(fn) {
		return fn
	}
	return i.callFunction(fn, []Object{left}, nil)
}

func (i *Interpreter) evalDestructureAssignment(node *ast.DestructureAssignment) Object {
	val := i.eval(node.Value)
	if isError(val) {
		return val
	}
	targets := node.Targets
	if len(targets) == 0 {
		for _, n := range node.Names {
			targets = append(targets, n)
		}
	}
	if r := i.destructureInto(targets, val); isError(r) {
		return r
	}
	return val
}

func (i *Interpreter) destructureInto(targets []ast.Expression, val Object) Object {
	var items []Object
	switch v := val.(type) {
	case *Array:
		items = v.Elements
	case *Hash:
		// legacy: a, b = hash binds by key name
		for _, t := range targets {
			id, ok := t.(*ast.Identifier)
			if !ok {
				continue
			}
			item, ok := v.Pairs[id.Value]
			if !ok {
				item = &Nil{}
			}
			i.setVar(id.Value, item)
		}
		return val
	case *StructInstance:
		for _, t := range targets {
			id, ok := t.(*ast.Identifier)
			if !ok {
				continue
			}
			item, ok := v.Fields[id.Value]
			if !ok {
				item = &Nil{}
			}
			i.setVar(id.Value, item)
		}
		return val
	case *Nil:
		items = nil
	default:
		items = []Object{val}
	}
	splat := -1
	for idx, t := range targets {
		if _, ok := t.(*ast.SplatExpression); ok {
			splat = idx
		}
	}
	assign := func(t ast.Expression, v Object) Object {
		return i.assignTarget(t, v)
	}
	for idx, t := range targets {
		var v Object = &Nil{}
		switch {
		case splat < 0 || idx < splat:
			if idx < len(items) {
				v = items[idx]
			}
		case idx == splat:
			after := len(targets) - splat - 1
			end := len(items) - after
			if end < idx {
				end = idx
			}
			if idx > len(items) {
				v = &Array{}
			} else {
				v = &Array{Elements: append([]Object{}, items[idx:min(end, len(items))]...)}
			}
		default:
			pos := len(items) - (len(targets) - idx)
			if pos >= splat && pos >= 0 && pos < len(items) {
				v = items[pos]
			}
		}
		if sp, ok := t.(*ast.SplatExpression); ok {
			t = sp.Value
		}
		if r := assign(t, v); isError(r) {
			return r
		}
	}
	return val
}

func (i *Interpreter) setVar(name string, v Object) Object {
	switch {
	case strings.HasPrefix(name, "@@"):
		return i.setCvar(name, v)
	case strings.HasPrefix(name, "$"):
		return i.rootEnv().Set(name, v)
	}
	if r, ok := i.env.Update(name, v); ok {
		return r
	}
	return i.env.Set(name, v)
}

// assignTarget stores v into an identifier, member, index or nested destructuring target.
func (i *Interpreter) assignTarget(t ast.Expression, v Object) Object {
	switch tt := t.(type) {
	case *ast.Identifier:
		if r := i.setVar(tt.Value, v); isError(r) {
			return r
		}
		return v
	case *ast.TypedIdentifier:
		return i.assignTarget(tt.Identifier, v)
	case *ast.DotExpression:
		if tt.Token.Type == lexer.IVAR {
			return i.setIvar(tt.Field.Value, v)
		}
		recv := i.eval(tt.Left)
		if isError(recv) {
			return recv
		}
		return i.assignMember(recv, tt.Field.Value, v)
	case *ast.IndexExpression:
		recv := i.eval(tt.Left)
		if isError(recv) {
			return recv
		}
		idx := i.eval(tt.Index)
		if isError(idx) {
			return idx
		}
		return i.indexSet(recv, idx, v)
	case *ast.DestructureAssignment:
		return i.destructureInto(tt.Targets, v)
	}
	return newError("cannot assign to %T", t)
}

func (i *Interpreter) evalLogicalAssignment(node *ast.LogicalAssignment) Object {
	var cur Object
	switch t := node.Target.(type) {
	case *ast.Identifier:
		if v, ok := i.env.Get(t.Value); ok {
			cur = v
		} else {
			cur = &Nil{}
			if strings.HasPrefix(t.Value, "@@") {
				if c := i.currentClass(); c != nil && c.findCvarOwner(t.Value) != nil {
					cur = i.getCvar(t.Value)
				}
			}
		}
	case *ast.IndexExpression:
		recv := i.eval(t.Left)
		if isError(recv) {
			return recv
		}
		idx := i.eval(t.Index)
		if isError(idx) {
			return idx
		}
		cur = i.indexGet(recv, idx)
		if isError(cur) {
			return cur
		}
		return i.logicalStep(node, cur, func(v Object) Object { return i.indexSet(recv, idx, v) })
	case *ast.DotExpression:
		recv := i.eval(t.Left)
		if isError(recv) {
			return recv
		}
		if t.Token.Type == lexer.IVAR {
			cur = i.getIvar(t.Field.Value)
			return i.logicalStep(node, cur, func(v Object) Object { return i.setIvar(t.Field.Value, v) })
		}
		cur = i.callMember(recv, t.Field.Value, nil, nil, callMode{})
		if isError(cur) {
			cur = &Nil{}
		}
		return i.logicalStep(node, cur, func(v Object) Object { return i.assignMember(recv, t.Field.Value, v) })
	default:
		return newError("invalid target for %s=", node.Operator)
	}
	return i.logicalStep(node, cur, func(v Object) Object { return i.assignTarget(node.Target, v) })
}

func (i *Interpreter) logicalStep(node *ast.LogicalAssignment, cur Object, store func(Object) Object) Object {
	_, isNil := cur.(*Nil)
	switch node.Operator {
	case "||":
		if isTruthy(cur) {
			return cur
		}
	case "&&":
		if !isTruthy(cur) {
			return cur
		}
	case "??":
		if !isNil {
			return cur
		}
	}
	v := i.eval(node.Value)
	if isError(v) {
		return v
	}
	if r := store(v); isError(r) {
		return r
	}
	return v
}

func (i *Interpreter) evalAlias(node *ast.AliasStatement) Object {
	if c, ok := i.env.Get("__defclass__"); ok {
		if cls, ok := c.(*ClassObject); ok {
			return i.aliasMethod(cls, node.NewName, node.OldName)
		}
	}
	old, ok := i.env.Get(node.OldName)
	if !ok {
		return newError("undefined method '%s' for alias", node.OldName)
	}
	i.rootEnv().Set(node.NewName, old)
	return &Nil{}
}

func (i *Interpreter) evalPostfixCondition(node *ast.PostfixCondition) Object {
	condition := i.eval(node.Condition)
	if isError(condition) {
		return condition
	}
	run := isTruthy(condition)
	if node.Unless {
		run = !run
	}
	if run {
		return i.eval(node.Statement)
	}
	if exprStmt, ok := node.Statement.(*ast.ExpressionStatement); ok {
		if assign, ok := exprStmt.Expression.(*ast.AssignmentExpression); ok {
			if _, exists := i.env.Get(assign.Name.Value); !exists {
				i.env.Set(assign.Name.Value, &Nil{})
			}
		}
	}
	return &Nil{}
}

func (i *Interpreter) evalInExpression(node *ast.InExpression) Object {
	left := i.eval(node.Left)
	if isError(left) {
		return left
	}
	right := i.eval(node.Right)
	if isError(right) {
		return right
	}
	switch c := right.(type) {
	case *Array:
		return boolObj(i.arrayHas(c, left))
	case *Hash:
		_, ok := c.Pairs[hashKey(left)]
		return boolObj(ok)
	case *String:
		if s, ok := left.(*String); ok {
			return boolObj(strings.Contains(c.Value, s.Value))
		}
		return boolObj(false)
	case *Range:
		return i.caseEqual(c, left)
	}
	return newError("'in' operator not supported for %s", right.Type())
}

func (i *Interpreter) evalCaseExpression(node *ast.CaseExpression) Object {
	var subject Object
	if node.Subject != nil {
		subject = i.eval(node.Subject)
		if isError(subject) {
			return subject
		}
	}

	for _, when := range node.Whens {
		for _, val := range when.Values {
			if sp, ok := val.(*ast.SplatExpression); ok {
				list := i.eval(sp.Value)
				if isError(list) {
					return list
				}
				for _, cand := range splatValues(list) {
					if subject != nil && isTruthy(i.caseEqual(cand, subject)) {
						return i.evalBlockStatement(when.Body)
					}
				}
				continue
			}
			matchVal := i.eval(val)
			if isError(matchVal) {
				return matchVal
			}
			if node.Subject == nil {
				if isTruthy(matchVal) {
					return i.evalBlockStatement(when.Body)
				}
				continue
			}
			if isTruthy(i.caseEqual(matchVal, subject)) {
				return i.evalBlockStatement(when.Body)
			}
		}
	}

	for _, in := range node.Ins {
		binds := map[string]Object{}
		ok, err := i.matchPattern(in.Pattern, subject, binds)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		for k, v := range binds {
			i.setVar(k, v)
		}
		if in.Guard != nil {
			g := i.eval(in.Guard)
			if isError(g) {
				return g
			}
			if isTruthy(g) == in.Unless {
				continue
			}
		}
		return i.evalBlockStatement(in.Body)
	}

	if node.Default != nil {
		return i.evalBlockStatement(node.Default)
	}
	if len(node.Ins) > 0 && subject != nil {
		return i.throwNew("NoMatchingPatternError", i.inspectValue(subject))
	}
	return &Nil{}
}

func (i *Interpreter) throwNew(class, msg string) Object {
	return &ThrowSignal{Value: i.newException(class, msg)}
}

func (i *Interpreter) evalMatchPattern(node *ast.MatchPattern) Object {
	v := i.eval(node.Value)
	if isError(v) {
		return v
	}
	binds := map[string]Object{}
	ok, err := i.matchPattern(node.Pattern, v, binds)
	if err != nil {
		return err
	}
	if ok {
		for k, b := range binds {
			i.setVar(k, b)
		}
	}
	if node.Strict {
		if !ok {
			return i.throwNew("NoMatchingPatternError", i.inspectValue(v))
		}
		return &Nil{}
	}
	return boolObj(ok)
}

// matchPattern tests v against a pattern node, collecting bindings.
func (i *Interpreter) matchPattern(p ast.Expression, v Object, binds map[string]Object) (bool, Object) {
	switch pt := p.(type) {
	case *ast.ValuePattern:
		pv := i.eval(pt.Value)
		if isError(pv) {
			return false, pv
		}
		return isTruthy(i.caseEqual(pv, v)), nil
	case *ast.PinPattern:
		pv := i.eval(pt.Value)
		if isError(pv) {
			return false, pv
		}
		return isTruthy(i.caseEqual(pv, v)), nil
	case *ast.BindPattern:
		if pt.Name != "_" {
			binds[pt.Name] = v
		}
		return true, nil
	case *ast.AltPattern:
		for _, a := range pt.Alts {
			trial := map[string]Object{}
			ok, err := i.matchPattern(a, v, trial)
			if err != nil {
				return false, err
			}
			if ok {
				for k, b := range trial {
					binds[k] = b
				}
				return true, nil
			}
		}
		return false, nil
	case *ast.CapturePattern:
		ok, err := i.matchPattern(pt.Pattern, v, binds)
		if err != nil || !ok {
			return false, err
		}
		binds[pt.Name] = v
		return true, nil
	case *ast.ArrayPattern:
		return i.matchArrayPattern(pt, v, binds)
	case *ast.HashPattern:
		return i.matchHashPattern(pt, v, binds)
	}
	// Any other expression is treated as a value pattern.
	pv := i.eval(p)
	if isError(pv) {
		return false, pv
	}
	return isTruthy(i.caseEqual(pv, v)), nil
}

func (i *Interpreter) patternClassOK(class ast.Expression, v Object) (bool, Object) {
	if class == nil {
		return true, nil
	}
	c := i.eval(class)
	if isError(c) {
		return false, c
	}
	return isTruthy(i.caseEqual(c, v)), nil
}

func (i *Interpreter) matchArrayPattern(pt *ast.ArrayPattern, v Object, binds map[string]Object) (bool, Object) {
	if ok, err := i.patternClassOK(pt.Class, v); err != nil || !ok {
		return false, err
	}
	var items []Object
	switch x := v.(type) {
	case *Array:
		items = x.Elements
	case *StructInstance:
		for _, f := range x.Struct.fieldOrder() {
			items = append(items, x.Fields[f])
		}
	case *ClassInstance:
		if m := x.Class.findMethod("deconstruct"); m != nil {
			r := i.callMethodWithBlock(x, m.owner, m.fn, nil, nil)
			arr, ok := r.(*Array)
			if !ok {
				return false, nil
			}
			items = arr.Elements
		} else {
			return false, nil
		}
	default:
		return false, nil
	}
	n := len(pt.Elements)
	if pt.RestIdx < 0 {
		if len(items) != n {
			return false, nil
		}
		for idx, e := range pt.Elements {
			ok, err := i.matchPattern(e, items[idx], binds)
			if err != nil || !ok {
				return false, err
			}
		}
		return true, nil
	}
	before := pt.RestIdx
	after := n - pt.RestIdx - 1
	if len(items) < before+after {
		return false, nil
	}
	for idx := 0; idx < before; idx++ {
		ok, err := i.matchPattern(pt.Elements[idx], items[idx], binds)
		if err != nil || !ok {
			return false, err
		}
	}
	for k := 0; k < after; k++ {
		ok, err := i.matchPattern(pt.Elements[pt.RestIdx+1+k], items[len(items)-after+k], binds)
		if err != nil || !ok {
			return false, err
		}
	}
	if pt.RestName != "" {
		binds[pt.RestName] = &Array{Elements: append([]Object{}, items[before:len(items)-after]...)}
	}
	return true, nil
}

func (s *Struct) fieldOrder() []string {
	if len(s.Order) > 0 {
		return s.Order
	}
	names := make([]string, 0, len(s.Fields))
	for k := range s.Fields {
		names = append(names, k)
	}
	return names
}

func (i *Interpreter) matchHashPattern(pt *ast.HashPattern, v Object, binds map[string]Object) (bool, Object) {
	if ok, err := i.patternClassOK(pt.Class, v); err != nil || !ok {
		return false, err
	}
	get := func(key string) (Object, bool) { return nil, false }
	var allKeys []string
	switch x := v.(type) {
	case *Hash:
		get = func(key string) (Object, bool) { o, ok := x.Pairs[key]; return o, ok }
		allKeys = x.Order
	case *StructInstance:
		get = func(key string) (Object, bool) { o, ok := x.Fields[key]; return o, ok }
		allKeys = x.Struct.fieldOrder()
	case *ClassInstance:
		if m := x.Class.findMethod("deconstruct_keys"); m != nil {
			r := i.callMethodWithBlock(x, m.owner, m.fn, []Object{&Nil{}}, nil)
			h, ok := r.(*Hash)
			if !ok {
				return false, nil
			}
			get = func(key string) (Object, bool) { o, ok := h.Pairs[key]; return o, ok }
			allKeys = h.Order
		} else {
			get = func(key string) (Object, bool) {
				if o, ok := x.Fields[key]; ok {
					return o, true
				}
				return nil, false
			}
			for k := range x.Fields {
				allKeys = append(allKeys, k)
			}
		}
	default:
		return false, nil
	}
	used := map[string]bool{}
	for idx, key := range pt.Keys {
		val, ok := get(key)
		if !ok {
			return false, nil
		}
		used[key] = true
		if pt.Patterns[idx] == nil {
			binds[key] = val
			continue
		}
		ok, err := i.matchPattern(pt.Patterns[idx], val, binds)
		if err != nil || !ok {
			return false, err
		}
	}
	if pt.Exact && len(allKeys) != len(used) {
		return false, nil
	}
	if pt.HasRest && pt.RestName != "" {
		rest := &Hash{Pairs: map[string]Object{}}
		for _, k := range allKeys {
			if !used[k] {
				val, _ := get(k)
				rest.set(k, &Symbol{Name: k}, val)
			}
		}
		binds[pt.RestName] = rest
	}
	return true, nil
}
