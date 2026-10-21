package interpreter

import (
	"fmt"
	"strings"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
)

// Additional object types introduced for Ruby-style syntax support.
const (
	SYMBOL_OBJ = "SYMBOL"
	REGEX_OBJ  = "REGEX"
	METHOD_OBJ = "METHOD"
	FRAME_OBJ  = "FRAME"
)

// Symbol is an interned-by-name identifier value such as :name.
type Symbol struct{ Name string }

func (s *Symbol) Type() ObjectType { return SYMBOL_OBJ }
func (s *Symbol) Inspect() string  { return ":" + s.Name }

// Frame identifies one activation of a method so non-local returns from
// blocks can find the method they belong to.
type Frame struct{}

func (f *Frame) Type() ObjectType { return FRAME_OBJ }
func (f *Frame) Inspect() string  { return "<frame>" }

// BoundMethod is a method looked up on a receiver, e.g. obj.method(:name).
type BoundMethod struct {
	Recv Object // nil for a global function
	Name string
}

func (b *BoundMethod) Type() ObjectType { return METHOD_OBJ }
func (b *BoundMethod) Inspect() string  { return "<method " + b.Name + ">" }

// callMode describes how a member is being accessed.
type callMode struct {
	isCall   bool // written as a call: recv.name(...)
	selfCall bool // the receiver is `self` (private methods are visible)
}

// hashKey converts a value to the string used as a Hash key.
func hashKey(o Object) string {
	if s, ok := o.(*Symbol); ok {
		return s.Name
	}
	return o.Inspect()
}

// display returns the string used by puts, print and interpolation.
func (i *Interpreter) display(o Object) string {
	switch v := o.(type) {
	case nil:
		return ""
	case *Symbol:
		return v.Name
	case *Nil:
		return ""
	case *ClassInstance:
		if m := v.Class.findMethod("to_s"); m != nil {
			r := i.callMethodWithBlock(v, m.owner, m.fn, nil, nil)
			if s, ok := r.(*String); ok {
				return s.Value
			}
		}
		if i.isException(v) {
			return i.exceptionMessage(v)
		}
		return v.Inspect()
	case *Array:
		parts := make([]string, len(v.Elements))
		for idx, e := range v.Elements {
			parts[idx] = i.inspectValue(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *Hash:
		return i.inspectValue(v)
	default:
		return o.Inspect()
	}
}

// inspectValue is the p/inspect representation (strings are quoted).
func (i *Interpreter) inspectValue(o Object) string {
	switch v := o.(type) {
	case *String:
		return fmt.Sprintf("%q", v.Value)
	case *Array:
		parts := make([]string, len(v.Elements))
		for idx, e := range v.Elements {
			parts[idx] = i.inspectValue(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *Hash:
		parts := make([]string, 0, len(v.Order))
		for _, k := range v.Order {
			parts = append(parts, k+": "+i.inspectValue(v.Pairs[k]))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case *Nil:
		return "nil"
	case *ClassInstance:
		if m := v.Class.findMethod("inspect"); m != nil {
			r := i.callMethodWithBlock(v, m.owner, m.fn, nil, nil)
			if s, ok := r.(*String); ok {
				return s.Value
			}
		}
		return v.Inspect()
	default:
		return o.Inspect()
	}
}

// applyFunction applies a callable object to arguments without a block.
func (i *Interpreter) applyFunction(fn Object, args []Object) Object {
	return i.callFunction(fn, args, nil)
}

// callFunction applies a callable object to arguments and an optional block.
func (i *Interpreter) callFunction(fn Object, args []Object, blk Object) Object {
	switch f := fn.(type) {
	case *Builtin:
		if blk != nil {
			withBlock := make([]Object, 0, len(args)+1)
			withBlock = append(withBlock, args...)
			args = append(withBlock, blk)
		}
		return f.Fn(args...)
	case *Function:
		return i.applyUserFunctionWithTypeArgs(f, args, nil, blk)
	case *ClassObject:
		return i.callClassConstructor(f, args, blk)
	case *Struct:
		return i.constructStruct(f, args)
	case *BoundMethod:
		if f.Recv == nil {
			target, ok := i.env.Get(f.Name)
			if !ok {
				return newError("undefined method '%s'", f.Name)
			}
			return i.callFunction(target, args, blk)
		}
		return i.callMember(f.Recv, f.Name, args, blk, callMode{isCall: true, selfCall: true})
	case *SuperProxy:
		return i.callSuperMethod(f, "initialize", args, blk)
	case *Symbol:
		name := f.Name
		if len(args) == 0 {
			return newError("no receiver given")
		}
		return i.callMember(args[0], name, args[1:], blk, callMode{isCall: true})
	default:
		if fn == nil {
			return newError("not a function: nil")
		}
		return newError("not a function: %s", fn.Type())
	}
}

// applyUserFunction applies a user-defined function to the given arguments.
func (i *Interpreter) applyUserFunction(fn *Function, args []Object) Object {
	return i.applyUserFunctionWithTypeArgs(fn, args, nil, nil)
}

// applyUserFunctionWithTypeArgs resolves generics and then runs the function.
func (i *Interpreter) applyUserFunctionWithTypeArgs(fn *Function, args []Object, typeArgs []string, blk Object) Object {
	if len(fn.TypeParams) > 0 {
		if err := i.resolveTypeParams(fn, args, typeArgs); err != nil {
			return err
		}
	}
	env := NewEnclosedEnvironment(fn.Env)
	return i.invoke(fn, args, env, blk)
}

func (i *Interpreter) resolveTypeParams(fn *Function, args []Object, typeArgs []string) *Error {
	resolvedTypes := make(map[string]string)

	if len(typeArgs) > 0 {
		if len(typeArgs) != len(fn.TypeParams) {
			return newError("wrong number of type arguments for %s: expected %d, got %d",
				fn.Name, len(fn.TypeParams), len(typeArgs))
		}
		for idx, tp := range fn.TypeParams {
			resolvedTypes[tp] = typeArgs[idx]
		}
	} else {
		for paramIdx, paramType := range fn.ParamTypes {
			if paramIdx >= len(args) {
				break
			}
			if paramType == nil {
				continue
			}
			typeAnno := extractTypeName(paramType)
			for _, tp := range fn.TypeParams {
				if typeAnno != tp {
					continue
				}
				argTypeName := objectTypeName(args[paramIdx])
				if existing, ok := resolvedTypes[tp]; ok {
					if existing != argTypeName {
						return newError("type parameter %s inconsistently inferred: %s vs %s",
							tp, existing, argTypeName)
					}
				} else {
					resolvedTypes[tp] = argTypeName
				}
			}
		}
	}

	for paramIdx, paramType := range fn.ParamTypes {
		if paramIdx >= len(args) {
			break
		}
		if paramType == nil {
			continue
		}
		typeAnno := extractTypeName(paramType)
		if resolvedType, ok := resolvedTypes[typeAnno]; ok {
			argTypeName := objectTypeName(args[paramIdx])
			if argTypeName != resolvedType {
				return newError("type mismatch for parameter '%s': expected %s (from %s = %s), got %s",
					fn.Parameters[paramIdx].Value, resolvedType, typeAnno, resolvedType, argTypeName)
			}
		}
	}
	return nil
}

// invoke binds parameters in env, runs the body and resolves control flow.
func (i *Interpreter) invoke(fn *Function, args []Object, env *Environment, blk Object) Object {
	var frame *Frame
	if fn.IsBlock {
		frame = fn.Frame
	} else {
		frame = &Frame{}
		env.Set("__frame__", frame)
		env.Set("__method__", fn)
		if blk != nil {
			env.Set("__block__", blk)
		} else {
			env.Set("__block__", &Nil{})
		}
	}
	if err := i.bindParams(fn, args, env, blk); err != nil {
		return err
	}
	old := i.env
	i.env = env
	result := i.eval(fn.Body)
	i.env = old
	res := i.finishBody(fn, frame, result)
	if !isError(res) {
		if err := i.checkReturnType(fn, res, env); err != nil {
			return err
		}
	}
	return res
}

// finishBody converts control-flow results of a function body into values.
func (i *Interpreter) finishBody(fn *Function, frame *Frame, result Object) Object {
	switch r := result.(type) {
	case *ReturnValue:
		if fn.IsBlock && !r.Next {
			return &Error{Message: "return from block", Signal: r, Frame: fn.Frame}
		}
		return r.Value
	case *BreakSignal:
		if fn.IsBlock {
			return &Error{Message: "break from block", Signal: r, Owner: fn}
		}
		return &Nil{}
	case *ContinueSignal:
		return &Nil{}
	case *Error:
		if rv, ok := r.Signal.(*ReturnValue); ok && !fn.IsBlock && r.Frame == frame {
			return rv.Value
		}
	}
	return result
}

// bindParams binds call arguments to the function's parameters.
func (i *Interpreter) bindParams(fn *Function, args []Object, env *Environment, blk Object) *Error {
	hasKw := fn.KwRest != nil
	for _, k := range fn.Keyword {
		if k {
			hasKw = true
		}
	}

	var kw *Hash
	if hasKw && len(args) > 0 {
		if h, ok := args[len(args)-1].(*Hash); ok && h.IsKw {
			kw = h
			args = args[:len(args)-1]
		}
	}

	isKeyword := func(idx int) bool { return idx < len(fn.Keyword) && fn.Keyword[idx] }

	positional := []int{}
	for idx := range fn.Parameters {
		if !isKeyword(idx) {
			positional = append(positional, idx)
		}
	}

	if fn.IsBlock && len(args) == 1 && len(positional) > 1 {
		if arr, ok := args[0].(*Array); ok {
			args = arr.Elements
		}
	}

	variadicIdx := -1
	if fn.Variadic && len(positional) > 0 {
		variadicIdx = positional[len(positional)-1]
	}

	required, optional := 0, 0
	for _, idx := range positional {
		if idx == variadicIdx {
			continue
		}
		if idx < len(fn.Defaults) && fn.Defaults[idx] != nil {
			optional++
		} else {
			required++
		}
	}
	if !fn.IsBlock && kw == nil && len(args) > 0 && len(args) <= required && variadicIdx < 0 {
		if h, ok := args[len(args)-1].(*Hash); ok && h.IsKw {
			named := map[string]bool{}
			for _, idx := range positional {
				named[fn.Parameters[idx].Value] = true
			}
			all := true
			for _, k := range h.Order {
				if !named[k] {
					all = false
				}
			}
			if all {
				rest := append([]Object{}, args[:len(args)-1]...)
				for n, idx := range positional {
					if n < len(rest) {
						continue
					}
					if v, ok := h.Pairs[fn.Parameters[idx].Value]; ok {
						rest = append(rest, v)
					} else {
						break
					}
				}
				if len(rest) >= required {
					args = rest
				}
			}
		}
	}
	if !fn.IsBlock {
		if len(args) < required {
			return newError("wrong number of arguments (given %d, expected %s)", len(args), arityString(required, optional, variadicIdx >= 0))
		}
		if variadicIdx < 0 && len(args) > required+optional {
			return newError("wrong number of arguments (given %d, expected %s)", len(args), arityString(required, optional, false))
		}
	}

	// Distribute optional slots left to right after required ones are satisfied.
	optAvail := len(args) - required
	if optAvail < 0 {
		optAvail = 0
	}
	argIdx := 0
	for _, idx := range positional {
		name := fn.Parameters[idx].Value
		if idx == variadicIdx {
			trailingRequired := 0
			rest := []Object{}
			if argIdx < len(args) {
				rest = args[argIdx:]
			}
			env.Set(name, &Array{Elements: append([]Object{}, rest...)})
			argIdx = len(args)
			_ = trailingRequired
			continue
		}
		hasDefault := idx < len(fn.Defaults) && fn.Defaults[idx] != nil
		switch {
		case hasDefault && optAvail <= 0:
			v, err := i.evalIn(env, fn.Defaults[idx])
			if err != nil {
				return err
			}
			env.Set(name, v)
		case argIdx < len(args):
			if hasDefault {
				optAvail--
			}
			env.Set(name, args[argIdx])
			argIdx++
		default:
			env.Set(name, &Nil{})
		}
	}

	if hasKw {
		used := map[string]bool{}
		for idx, p := range fn.Parameters {
			if !isKeyword(idx) {
				continue
			}
			name := p.Value
			if kw != nil {
				if v, ok := kw.Pairs[name]; ok {
					env.Set(name, v)
					used[name] = true
					continue
				}
			}
			if idx < len(fn.Defaults) && fn.Defaults[idx] != nil {
				v, err := i.evalIn(env, fn.Defaults[idx])
				if err != nil {
					return err
				}
				env.Set(name, v)
				continue
			}
			return newError("missing keyword: %s", name)
		}
		rest := &Hash{Pairs: map[string]Object{}}
		if kw != nil {
			for _, k := range kw.Order {
				if !used[k] {
					rest.Pairs[k] = kw.Pairs[k]
					rest.Order = append(rest.Order, k)
				}
			}
		}
		if fn.KwRest != nil {
			env.Set(fn.KwRest.Value, rest)
		} else if len(rest.Order) > 0 {
			return newError("unknown keyword: %s", rest.Order[0])
		}
	}

	if fn.BlockParam != nil {
		if blk != nil {
			env.Set(fn.BlockParam.Value, blk)
		} else {
			env.Set(fn.BlockParam.Value, &Nil{})
		}
	}
	for idx, p := range fn.Parameters {
		if idx >= len(fn.ParamTypes) || fn.ParamTypes[idx] == nil {
			continue
		}
		v, _ := env.GetLocal(p.Value)
		if v == nil {
			continue
		}
		if idx == variadicIdx {
			if arr, ok := v.(*Array); ok {
				for _, e := range arr.Elements {
					if err := i.checkParamType(fn, idx, e, env); err != nil {
						return err
					}
				}
			}
			continue
		}
		if err := i.checkParamType(fn, idx, v, env); err != nil {
			return err
		}
	}
	return nil
}

func arityString(required, optional int, variadic bool) string {
	switch {
	case variadic:
		return fmt.Sprintf("%d+", required)
	case optional > 0:
		return fmt.Sprintf("%d..%d", required, required+optional)
	}
	return fmt.Sprintf("%d", required)
}

// evalIn evaluates an expression in the given environment.
func (i *Interpreter) evalIn(env *Environment, expr ast.Node) (Object, *Error) {
	old := i.env
	i.env = env
	v := i.eval(expr)
	i.env = old
	if e, ok := v.(*Error); ok {
		return nil, e
	}
	return v, nil
}

// evalArgs evaluates call arguments, expanding splats and collecting
// keyword arguments into a trailing keyword Hash.
func (i *Interpreter) evalArgs(exprs []ast.Expression) (args []Object, blockPass Object, err Object) {
	var kw *Hash
	addKw := func(k string, v Object) {
		if kw == nil {
			kw = &Hash{Pairs: map[string]Object{}, IsKw: true}
		}
		if _, ok := kw.Pairs[k]; !ok {
			kw.Order = append(kw.Order, k)
		}
		kw.Pairs[k] = v
	}
	for _, e := range exprs {
		switch a := e.(type) {
		case *ast.SplatExpression:
			v := i.eval(a.Value)
			if isError(v) {
				return nil, nil, v
			}
			args = append(args, splatValues(v)...)
		case *ast.KeywordArgument:
			v := i.eval(a.Value)
			if isError(v) {
				return nil, nil, v
			}
			addKw(a.Name, v)
		case *ast.DoubleSplatExpression:
			v := i.eval(a.Value)
			if isError(v) {
				return nil, nil, v
			}
			h, ok := v.(*Hash)
			if !ok {
				if _, isNil := v.(*Nil); isNil {
					continue
				}
				return nil, nil, newError("no implicit conversion of %s into Hash", v.Type())
			}
			for _, k := range h.Order {
				addKw(k, h.Pairs[k])
			}
		case *ast.BlockPass:
			v := i.eval(a.Value)
			if isError(v) {
				return nil, nil, v
			}
			blockPass, err = i.toBlock(v)
			if err != nil {
				return nil, nil, err
			}
		default:
			v := i.eval(e)
			if isError(v) {
				return nil, nil, v
			}
			args = append(args, v)
		}
	}
	if kw != nil {
		args = append(args, kw)
	}
	return args, blockPass, nil
}

// splatValues expands a value for *splat.
func splatValues(v Object) []Object {
	switch s := v.(type) {
	case *Array:
		return append([]Object{}, s.Elements...)
	case *Range:
		return s.ToArray().Elements
	case *Nil:
		return nil
	case *Hash:
		out := make([]Object, 0, len(s.Order))
		for _, k := range s.Order {
			out = append(out, &Array{Elements: []Object{&String{Value: k}, s.Pairs[k]}})
		}
		return out
	}
	return []Object{v}
}

// toBlock converts the operand of &expr into a callable block.
func (i *Interpreter) toBlock(v Object) (Object, Object) {
	switch b := v.(type) {
	case *Nil:
		return nil, nil
	case *Function, *Builtin, *BoundMethod:
		return v, nil
	case *Symbol:
		name := b.Name
		return &Builtin{Fn: func(args ...Object) Object {
			if len(args) == 0 {
				return newError("no receiver given")
			}
			return i.callMember(args[0], name, args[1:], nil, callMode{isCall: true})
		}}, nil
	}
	return nil, newError("wrong argument type %s (expected Proc)", v.Type())
}

// catchBreak resolves a `break` thrown by blk once the call that received it returns.
func catchBreak(result Object, blk Object) Object {
	if e, ok := result.(*Error); ok && e.Signal != nil && blk != nil {
		if bs, ok := e.Signal.(*BreakSignal); ok && e.Owner == blk {
			if bs.Value != nil {
				return bs.Value
			}
			return &Nil{}
		}
	}
	return result
}

// evalCallExpression evaluates f(args), obj.m(args) and calls with blocks.
func (i *Interpreter) evalCallExpression(node *ast.CallExpression) Object {
	var blk Object
	if node.Block != nil {
		blk = i.evalArrowFunction(node.Block)
	}

	switch fnode := node.Function.(type) {
	case *ast.Identifier:
		switch fnode.Value {
		case "attr_accessor", "attr_reader", "attr_writer":
			if res, handled := i.evalTypedAttr(fnode.Value, node); handled {
				return res
			}
		}
		if res, handled := i.evalSpecialCall(fnode, node, blk); handled {
			return res
		}
	case *ast.DotExpression:
		return i.evalMethodCall(fnode, node, blk)
	}

	args, bp, err := i.evalArgs(node.Arguments)
	if err != nil {
		return err
	}
	if blk == nil {
		blk = bp
	}

	// Calls to a bare name prefer methods of `self` over global functions.
	if id, ok := node.Function.(*ast.Identifier); ok {
		if res, found := i.callSelfMethod(id.Value, args, blk); found {
			return catchBreak(res, blk)
		}
	}

	function := i.eval(node.Function)
	if isError(function) {
		return function
	}
	if len(node.TypeArgs) > 0 {
		if fn, ok := function.(*Function); ok {
			return catchBreak(i.applyUserFunctionWithTypeArgs(fn, args, node.TypeArgs, blk), blk)
		}
	}
	return catchBreak(i.callFunction(function, args, blk), blk)
}

// callSelfMethod calls name as a method of the current self, if one exists.
func (i *Interpreter) callSelfMethod(name string, args []Object, blk Object) (Object, bool) {
	selfObj, ok := i.env.Get("self")
	if !ok {
		return nil, false
	}
	switch s := selfObj.(type) {
	case *ClassInstance:
		if m := s.Class.findMethod(name); m != nil {
			return i.callMethodWithBlock(s, m.owner, m.fn, args, blk), true
		}
		if nat := s.Class.findNative(name); nat != nil {
			return nat(i, s, args, blk), true
		}
	case *ClassObject:
		if m := s.findStatic(name); m != nil {
			return i.callStatic(s, m, args, blk), true
		}
		if nat := s.findNativeStatic(name); nat != nil {
			return nat(i, s, args, blk), true
		}
		if classMacroNames[name] {
			return i.classMacro(s, name, args, blk), true
		}
		switch name {
		case "class_eval", "instance_method", "new", "name", "ancestors", "instance_methods", "const_get", "const_set",
			"class_variable_get", "class_variable_set", "define_singleton_method", "instance_variable_get", "instance_variable_set":
			return i.callClassMember(s, name, args, blk, callMode{isCall: true, selfCall: true})
		}
	}
	return nil, false
}

var classMacroNames = map[string]bool{
	"include": true, "extend": true, "attr_accessor": true, "attr_reader": true, "attr_writer": true, "private": true,
	"public": true, "protected": true, "module_function": true, "private_class_method": true, "public_class_method": true,
	"alias_method": true, "define_method": true, "private_constant": true, "attr": true, "remove_method": true,
	"undef_method": true, "prepend": true,
}

func (i *Interpreter) evalMethodCall(fnode *ast.DotExpression, node *ast.CallExpression, blk Object) Object {
	recv := i.evalReceiver(fnode.Left)
	if isError(recv) {
		return recv
	}
	if fnode.Safe {
		if _, isNil := recv.(*Nil); isNil {
			return &Nil{}
		}
	}
	args, bp, err := i.evalArgs(node.Arguments)
	if err != nil {
		return err
	}
	if blk == nil {
		blk = bp
	}
	selfCall := false
	if id, ok := fnode.Left.(*ast.Identifier); ok && id.Value == "self" {
		selfCall = true
	}
	res := i.callMember(recv, fnode.Field.Value, args, blk, callMode{isCall: true, selfCall: selfCall})
	return catchBreak(res, blk)
}

// evalReceiver evaluates the left side of a method call; a bare `super`
// there is a proxy for the parent implementation, not a call of it.
func (i *Interpreter) evalReceiver(n ast.Expression) Object {
	isSuper := false
	switch x := n.(type) {
	case *ast.Identifier:
		isSuper = x.Value == "super"
	case *ast.CallExpression:
		if id, ok := x.Function.(*ast.Identifier); ok && id.Value == "super" && x.Implicit && len(x.Arguments) == 0 {
			isSuper = true
		}
	}
	if !isSuper {
		return i.eval(n)
	}
	selfObj, ok := i.env.Get("self")
	instance, isInst := selfObj.(*ClassInstance)
	if !ok || !isInst {
		return newError("'super' used outside of a class")
	}
	cur, ok := i.env.Get("__current_class__")
	current, isClass := cur.(*ClassObject)
	if !ok || !isClass {
		return newError("'super' used outside of a class")
	}
	return &SuperProxy{Instance: instance, Parent: current.Parent, After: current}
}

// evalSpecialCall handles forms that need unevaluated operands or frame state.
func (i *Interpreter) evalSpecialCall(id *ast.Identifier, node *ast.CallExpression, blk Object) (Object, bool) {
	switch id.Value {
	case "defined?":
		if len(node.Arguments) != 1 {
			return newError("defined? expects 1 argument"), true
		}
		return i.evalDefined(node.Arguments[0]), true
	case "super":
		return i.evalSuperCall(node, blk), true
	case "block_given?":
		b, ok := i.env.Get("__block__")
		if !ok {
			return &Boolean{Value: false}, true
		}
		_, isNil := b.(*Nil)
		return &Boolean{Value: !isNil}, true
	case "binding":
		return &Nil{}, true
	}
	return nil, false
}

func (i *Interpreter) evalDefined(expr ast.Expression) Object {
	str := func(s string) Object { return &String{Value: s} }
	switch e := expr.(type) {
	case *ast.Identifier:
		if e.Value == "self" {
			return str("self")
		}
		if v, ok := i.env.Get(e.Value); ok {
			switch v.(type) {
			case *Function, *Builtin:
				return str("method")
			}
			if strings.HasPrefix(e.Value, "$") {
				return str("global-variable")
			}
			if strings.HasPrefix(e.Value, "@@") {
				return str("class variable")
			}
			if e.Value != "" && e.Value[0] >= 'A' && e.Value[0] <= 'Z' {
				return str("constant")
			}
			return str("local-variable")
		}
		if sm, found := i.selfHasMethod(e.Value); found {
			_ = sm
			return str("method")
		}
		return &Nil{}
	case *ast.DotExpression:
		if e.Token.Type == lexer.IVAR {
			if self, ok := i.env.Get("self"); ok {
				if ci, ok := self.(*ClassInstance); ok {
					if _, has := ci.Fields[e.Field.Value]; has {
						return str("instance-variable")
					}
				}
			}
			return &Nil{}
		}
		v := i.eval(e)
		if isError(v) {
			return &Nil{}
		}
		return str("method")
	case *ast.CallExpression:
		v := i.eval(e)
		if isError(v) {
			return &Nil{}
		}
		return str("method")
	case *ast.YieldExpression:
		b, ok := i.env.Get("__block__")
		if _, isNil := b.(*Nil); !ok || isNil {
			return &Nil{}
		}
		return str("yield")
	case *ast.AssignmentExpression:
		return str("assignment")
	}
	v := i.eval(expr)
	if isError(v) {
		return &Nil{}
	}
	return str("expression")
}

func (i *Interpreter) selfHasMethod(name string) (*Function, bool) {
	self, ok := i.env.Get("self")
	if !ok {
		return nil, false
	}
	if ci, ok := self.(*ClassInstance); ok {
		if m := ci.Class.findMethod(name); m != nil {
			return m.fn, true
		}
	}
	return nil, false
}

// evalYield calls the block passed to the current method.
func (i *Interpreter) evalYield(node *ast.YieldExpression) Object {
	b, ok := i.env.Get("__block__")
	if !ok {
		return newError("no block given (yield)")
	}
	if _, isNil := b.(*Nil); isNil {
		return newError("no block given (yield)")
	}
	args, _, err := i.evalArgs(node.Arguments)
	if err != nil {
		return err
	}
	return i.callFunction(b, args, nil)
}

// evalSuperCall handles super(args) and bare super (which forwards the current arguments).
func (i *Interpreter) evalSuperCall(node *ast.CallExpression, blk Object) Object {
	selfObj, ok := i.env.Get("self")
	if !ok {
		return newError("'super' used outside of a class")
	}
	instance, ok := selfObj.(*ClassInstance)
	if !ok {
		return newError("'super' used outside of a class")
	}
	curObj, ok := i.env.Get("__current_class__")
	current, isClass := curObj.(*ClassObject)
	if !ok || !isClass {
		return newError("'super' used outside of a class")
	}
	methodName := "initialize"
	if m, ok := i.env.Get("__method__"); ok {
		if fn, ok := m.(*Function); ok && fn.Name != "" {
			methodName = fn.Name
		}
	}

	var args []Object
	if node.Implicit {
		args = i.currentArgs()
		if blk == nil {
			if b, ok := i.env.Get("__block__"); ok {
				if _, isNil := b.(*Nil); !isNil {
					blk = b
				}
			}
		}
	} else {
		var bp Object
		var err Object
		args, bp, err = i.evalArgs(node.Arguments)
		if err != nil {
			return err
		}
		if blk == nil {
			blk = bp
		}
	}

	if m := instance.Class.findMethodAfter(current, methodName); m != nil {
		return i.callMethodWithBlock(instance, m.owner, m.fn, args, blk)
	}
	if nat := instance.Class.findNativeAfter(current, methodName); nat != nil {
		return nat(i, instance, args, blk)
	}
	if methodName == "initialize" {
		return &Nil{}
	}
	if res, found := i.callBuiltinMethod(instance, methodName, args, blk); found {
		return res
	}
	return newError("super: no parent method '%s'", methodName)
}

// currentArgs rebuilds the argument list of the running method for bare `super`.
func (i *Interpreter) currentArgs() []Object {
	m, ok := i.env.Get("__method__")
	if !ok {
		return nil
	}
	fn, ok := m.(*Function)
	if !ok {
		return nil
	}
	var args []Object
	var kw *Hash
	for idx, p := range fn.Parameters {
		v, _ := i.env.Get(p.Value)
		if v == nil {
			v = &Nil{}
		}
		if idx < len(fn.Keyword) && fn.Keyword[idx] {
			if kw == nil {
				kw = &Hash{Pairs: map[string]Object{}, IsKw: true}
			}
			kw.Pairs[p.Value] = v
			kw.Order = append(kw.Order, p.Value)
			continue
		}
		if fn.Variadic && idx == len(fn.Parameters)-1 {
			if arr, ok := v.(*Array); ok {
				args = append(args, arr.Elements...)
				continue
			}
		}
		args = append(args, v)
	}
	if kw != nil {
		args = append(args, kw)
	}
	return args
}
