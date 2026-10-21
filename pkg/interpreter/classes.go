package interpreter

import (
	"sort"
	"strings"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
)

// nativeMethod is a method implemented in Go on a class.
type nativeMethod func(i *Interpreter, self Object, args []Object, blk Object) Object

// methodRef is a user-defined method together with the class or module that owns it.
type methodRef struct {
	fn    *Function
	owner *ClassObject
}

func newClassObject(name string) *ClassObject {
	return &ClassObject{
		Name:       name,
		Methods:    map[string]*Function{},
		Statics:    map[string]*Function{},
		Natives:    map[string]nativeMethod{},
		NativeStat: map[string]nativeMethod{},
		Consts:     map[string]Object{},
		Ivars:      map[string]Object{},
		Cvars:      map[string]Object{},
	}
}

// ancestors returns the method resolution order: the class, its mixins
// (last included first), then the same for each parent class.
func (c *ClassObject) ancestors() []*ClassObject {
	var out []*ClassObject
	seen := map[*ClassObject]bool{}
	var add func(k *ClassObject)
	add = func(k *ClassObject) {
		if seen[k] {
			return
		}
		seen[k] = true
		out = append(out, k)
		for idx := len(k.Includes) - 1; idx >= 0; idx-- {
			add(k.Includes[idx])
		}
	}
	for k := c; k != nil; k = k.Parent {
		add(k)
	}
	return out
}

// isA reports whether c is other or inherits from / includes it.
func (c *ClassObject) isA(other *ClassObject) bool {
	if other == nil {
		return false
	}
	for _, a := range c.ancestors() {
		if a == other {
			return true
		}
	}
	return false
}

func (c *ClassObject) findMethod(name string) *methodRef {
	for _, a := range c.ancestors() {
		if fn, ok := a.Methods[name]; ok {
			return &methodRef{fn: fn, owner: a}
		}
	}
	return nil
}

// findMethodAfter finds the next definition of name after current in c's ancestors.
func (c *ClassObject) findMethodAfter(current *ClassObject, name string) *methodRef {
	found := false
	for _, a := range c.ancestors() {
		if !found {
			found = a == current
			continue
		}
		if fn, ok := a.Methods[name]; ok {
			return &methodRef{fn: fn, owner: a}
		}
	}
	return nil
}

func (c *ClassObject) findNativeAfter(current *ClassObject, name string) nativeMethod {
	found := current == nil
	for _, a := range c.ancestors() {
		if !found {
			found = a == current
			continue
		}
		if n, ok := a.Natives[name]; ok {
			return n
		}
	}
	return nil
}

func (c *ClassObject) findNative(name string) nativeMethod {
	return c.findNativeAfter(nil, name)
}

func (c *ClassObject) findStatic(name string) *Function {
	for k := c; k != nil; k = k.Parent {
		if fn, ok := k.Statics[name]; ok {
			return fn
		}
		for idx := len(k.Extends) - 1; idx >= 0; idx-- {
			if fn, ok := k.Extends[idx].Methods[name]; ok {
				return fn
			}
		}
	}
	return nil
}

func (c *ClassObject) findNativeStatic(name string) nativeMethod {
	for k := c; k != nil; k = k.Parent {
		if n, ok := k.NativeStat[name]; ok {
			return n
		}
	}
	return nil
}

func (c *ClassObject) findConst(name string) (Object, bool) {
	for _, a := range c.ancestors() {
		if v, ok := a.Consts[name]; ok {
			return v, true
		}
	}
	return nil, false
}

func (c *ClassObject) findCvarOwner(name string) *ClassObject {
	for k := c; k != nil; k = k.Parent {
		if _, ok := k.Cvars[name]; ok {
			return k
		}
	}
	return nil
}

// GetMethod looks up a method on the class or any ancestor.
func (c *ClassObject) GetMethod(name string) (*Function, bool) {
	if m := c.findMethod(name); m != nil {
		return m.fn, true
	}
	return nil, false
}

// findMethodInChain returns the owner of a method and the method itself.
func findMethodInChain(class *ClassObject, name string) (*ClassObject, *Function) {
	if m := class.findMethod(name); m != nil {
		return m.owner, m.fn
	}
	return nil, nil
}

func (i *Interpreter) newFunction(node *ast.FunctionLiteral) *Function {
	return &Function{
		Parameters: node.Parameters,
		Defaults:   node.ParamDefaults,
		Body:       node.Body,
		Env:        i.env,
		TypeParams: node.TypeParams,
		ParamTypes: node.ParamTypes,
		ReturnType: node.ReturnType,
		Variadic:   node.Variadic,
		Keyword:    node.Keyword,
		KwRest:     node.KwRest,
		BlockParam: node.BlockParam,
	}
}

// evalFunctionLiteral evaluates fn/def. Inside a class body, a def registers a method.
func (i *Interpreter) evalFunctionLiteral(node *ast.FunctionLiteral) Object {
	fn := i.newFunction(node)
	if node.Name == nil {
		return fn
	}
	fn.Name = node.Name.Value

	if dc, ok := i.env.GetLocal("__defclass__"); ok {
		class := dc.(*ClassObject)
		singleton := node.Receiver != nil
		moduleFn := false
		if mf, ok := i.env.GetLocal("__module_function__"); ok {
			moduleFn = mf == Object(&Boolean{Value: true}) || isTruthy(mf)
		}
		if vis, ok := i.env.GetLocal("__visibility__"); ok {
			if s, ok := vis.(*String); ok && !singleton {
				fn.Visibility = s.Value
			}
		}
		switch {
		case singleton:
			class.Statics[fn.Name] = fn
		case moduleFn:
			class.Methods[fn.Name] = fn
			class.Statics[fn.Name] = fn
		default:
			class.Methods[fn.Name] = fn
		}
		return &Symbol{Name: fn.Name}
	}

	if node.Receiver != nil {
		recv := i.eval(node.Receiver)
		if c, ok := recv.(*ClassObject); ok {
			c.Statics[fn.Name] = fn
		}
		return fn
	}

	// Top-level (or nested function) def: define in the nearest scope that holds a method table.
	i.env.Set(fn.Name, fn)
	return fn
}

// evalClassLiteral defines (or reopens) a class or module and runs its body.
func (i *Interpreter) evalClassLiteral(node *ast.ClassLiteral) Object {
	name := node.Name.Value
	var classObj *ClassObject
	if existing, ok := i.env.Get(name); ok {
		if c, ok := existing.(*ClassObject); ok && c.IsModule == node.IsModule && !c.native() {
			classObj = c
		}
	}
	if classObj == nil {
		classObj = newClassObject(name)
		classObj.IsModule = node.IsModule
		classObj.TypeParams = node.TypeParams
	}

	if node.Parent != nil {
		parentObj, ok := i.env.Get(node.Parent.Value)
		if !ok {
			return newError("undefined parent class: %s", node.Parent.Value)
		}
		parentClass, ok := parentObj.(*ClassObject)
		if !ok {
			return newError("'%s' is not a class", node.Parent.Value)
		}
		classObj.Parent = parentClass
	}

	// Nested classes register under their enclosing class too (Outer::Inner).
	if outer, ok := i.env.GetLocal("__defclass__"); ok {
		outer.(*ClassObject).Consts[name] = classObj
	}
	i.env.Set(name, classObj)

	classEnv := NewEnclosedEnvironment(i.env)
	classEnv.Set("self", classObj)
	classEnv.Set("__defclass__", classObj)
	classEnv.Set("__visibility__", &String{Value: ""})
	classEnv.Set("__module_function__", &Boolean{Value: false})

	old := i.env
	i.env = classEnv
	var result Object = &Nil{}
	for _, stmt := range node.Body.Statements {
		result = i.eval(stmt)
		if isError(result) {
			i.env = old
			return result
		}
	}
	i.env = old
	for _, iname := range node.Implements {
		if err := i.checkImplements(classObj, iname); err != nil {
			return err
		}
	}
	return classObj
}

func (i *Interpreter) checkImplements(c *ClassObject, iname string) *Error {
	v, ok := i.env.Get(iname)
	iface, isIface := v.(*Interface)
	if !ok || !isIface {
		return newError("'%s' is not an interface", iname)
	}
	for _, m := range iface.Methods {
		if c.findMethod(m.Name) == nil && c.findNative(m.Name) == nil {
			return newErrorClass("TypeError", "class %s does not implement %s: missing method '%s'", c.Name, iname, m.Name)
		}
	}
	return nil
}

func (c *ClassObject) native() bool {
	return len(c.Natives) > 0 || len(c.NativeStat) > 0
}

// declareProperty handles `prop name: Type` in a class body.
func (i *Interpreter) evalPropertyStatement(node *ast.PropertyStatement) Object {
	dc, ok := i.env.GetLocal("__defclass__")
	if !ok {
		return newError("'prop' is only valid inside a class")
	}
	class := dc.(*ClassObject)
	class.Props = append(class.Props, node.Name.Value)
	class.defineAccessor(node.Name.Value, true, true)
	return &Nil{}
}

// defineAccessor creates native getter/setter methods for an instance field.
func (c *ClassObject) defineAccessor(field string, reader, writer bool) {
	if reader {
		c.Natives[field] = func(i *Interpreter, self Object, args []Object, blk Object) Object {
			inst, ok := self.(*ClassInstance)
			if !ok {
				return &Nil{}
			}
			if v, ok := inst.Fields[field]; ok {
				return v
			}
			return &Nil{}
		}
	}
	if writer {
		c.Natives[field+"="] = func(i *Interpreter, self Object, args []Object, blk Object) Object {
			inst, ok := self.(*ClassInstance)
			if !ok || len(args) != 1 {
				return newError("wrong number of arguments (given %d, expected 1)", len(args))
			}
			if inst.Frozen {
				return newError("can't modify frozen %s", inst.Class.Name)
			}
			if t := c.propType(field); t != nil && !i.typeMatches(args[0], t, i.env, c.TypeParams) {
				return newErrorClass("TypeError", "type mismatch for property '%s': expected %s, got %s",
					field, t.String(), i.typeLabel(args[0]))
			}
			inst.Fields[field] = args[0]
			return args[0]
		}
	}
}

func (c *ClassObject) propType(field string) ast.Expression {
	for k := c; k != nil; k = k.Parent {
		if t, ok := k.PropTypes[field]; ok {
			return t
		}
	}
	return nil
}

// evalTypedAttr handles `attr_accessor name: string, age: int` in a class body.
func (i *Interpreter) evalTypedAttr(name string, node *ast.CallExpression) (Object, bool) {
	typed := false
	for _, a := range node.Arguments {
		if _, ok := a.(*ast.KeywordArgument); ok {
			typed = true
		}
	}
	if !typed {
		return nil, false
	}
	selfObj, _ := i.env.Get("self")
	c, ok := selfObj.(*ClassObject)
	if !ok {
		return nil, false
	}
	var out []Object
	for _, a := range node.Arguments {
		field := ""
		if kw, ok := a.(*ast.KeywordArgument); ok {
			field = kw.Name
			if c.PropTypes == nil {
				c.PropTypes = map[string]ast.Expression{}
			}
			c.PropTypes[field] = kw.Value
		} else {
			v := i.eval(a)
			if isError(v) {
				return v, true
			}
			field = symOrString(v)
		}
		if r, ok := i.classMacro(c, name, []Object{&Symbol{Name: field}}, nil).(*Array); ok {
			out = append(out, r.Elements...)
		}
	}
	return &Array{Elements: out}, true
}

// callClassConstructor creates a new instance of a class and calls initialize.
func (i *Interpreter) callClassConstructor(classObj *ClassObject, args []Object, blk Object) Object {
	if classObj.Call != nil {
		return classObj.Call(args...)
	}
	if classObj.IsModule {
		return newError("undefined method 'new' for module %s", classObj.Name)
	}
	instance := &ClassInstance{Class: classObj, Fields: make(map[string]Object)}
	if m := classObj.findMethod("initialize"); m != nil {
		if r := i.callMethodWithBlock(instance, m.owner, m.fn, args, blk); isError(r) {
			return r
		}
	} else if nat := classObj.findNative("initialize"); nat != nil {
		if r := nat(i, instance, args, blk); isError(r) {
			return r
		}
	} else if len(args) > 0 {
		return newError("wrong number of arguments (given %d, expected 0)", len(args))
	}
	return instance
}

// callMethodOnInstance calls a method on a class instance, binding self.
func (i *Interpreter) callMethodOnInstance(instance *ClassInstance, method *Function, args []Object) Object {
	owner := instance.Class
	if m := instance.Class.findMethod(method.Name); m != nil && m.fn == method {
		owner = m.owner
	}
	return i.callMethodWithBlock(instance, owner, method, args, nil)
}

func (i *Interpreter) callMethodOnInstanceFromClass(instance *ClassInstance, definingClass *ClassObject, method *Function, args []Object) Object {
	return i.callMethodWithBlock(instance, definingClass, method, args, nil)
}

// callMethodWithBlock runs a user method with self bound to the receiver.
func (i *Interpreter) callMethodWithBlock(self Object, owner *ClassObject, method *Function, args []Object, blk Object) Object {
	env := NewEnclosedEnvironment(method.Env)
	env.Set("self", self)
	env.Set("__current_class__", owner)
	return i.invoke(method, args, env, blk)
}

// callStatic runs a singleton method with the class as self.
func (i *Interpreter) callStatic(class *ClassObject, method *Function, args []Object, blk Object) Object {
	env := NewEnclosedEnvironment(method.Env)
	env.Set("self", class)
	env.Set("__current_class__", class)
	return i.invoke(method, args, env, blk)
}

// callMember invokes name on recv: user methods first, then builtin methods,
// then global functions (UFCS).
func (i *Interpreter) callMember(recv Object, name string, args []Object, blk Object, mode callMode) Object {
	switch r := recv.(type) {
	case *SuperProxy:
		return i.callSuperMethod(r, name, args, blk)
	case *ClassInstance:
		if m := r.Class.findMethod(name); m != nil {
			if res := i.checkVisibility(m.fn, name, recv, mode); res != nil {
				return res
			}
			return i.callMethodWithBlock(r, m.owner, m.fn, args, blk)
		}
		if nat := r.Class.findNative(name); nat != nil {
			return nat(i, r, args, blk)
		}
		if f, ok := r.Fields[name]; ok {
			if isCallable(f) && mode.isCall {
				return i.callFunction(f, args, blk)
			}
			if len(args) == 0 {
				return f
			}
		}
		if mm := r.Class.findMethod("method_missing"); mm != nil {
			margs := append([]Object{&Symbol{Name: name}}, args...)
			return i.callMethodWithBlock(r, mm.owner, mm.fn, margs, blk)
		}
	case *ClassObject:
		if res, ok := i.callClassMember(r, name, args, blk, mode); ok {
			return res
		}
	case *StructInstance:
		if m, ok := r.Struct.Methods[name]; ok {
			env := NewEnclosedEnvironment(m.Env)
			env.Set("self", r)
			return catchBreak(i.invoke(m, args, env, blk), blk)
		}
		if f, ok := r.Fields[name]; ok {
			if isCallable(f) && mode.isCall {
				return i.callFunction(f, args, blk)
			}
			if len(args) == 0 {
				return f
			}
		}
		if res, ok := i.callStructMethod(r, name, args, blk); ok {
			return res
		}
	case *Hash:
		if v, ok := r.Pairs[name]; ok && !hasBuiltinMethod(recv, name) {
			if isCallable(v) && mode.isCall {
				return i.callFunction(v, args, blk)
			}
			if len(args) == 0 {
				return v
			}
		}
	}

	if res, found := i.callBuiltinMethod(recv, name, args, blk); found {
		return res
	}

	// Universal function-call syntax: x.f(a) is f(x, a).
	if fn, ok := i.env.Get(name); ok && isCallable(fn) {
		all := make([]Object, 0, len(args)+1)
		all = append(all, recv)
		all = append(all, args...)
		return i.callFunction(fn, all, blk)
	}

	if si, ok := recv.(*StructInstance); ok {
		return newError("undefined field '%s' in struct '%s'", name, si.Struct.Name)
	}
	if ci, ok := recv.(*ClassInstance); ok {
		return newError("undefined method '%s' for %s", name, ci.Class.Name)
	}
	return newError("undefined method '%s' for %s", name, i.describeType(recv))
}

func (i *Interpreter) describeType(o Object) string {
	switch v := o.(type) {
	case *Nil:
		return "nil"
	case *ClassObject:
		return "class " + v.Name
	}
	return string(o.Type())
}

func isCallable(o Object) bool {
	switch o.(type) {
	case *Function, *Builtin, *BoundMethod:
		return true
	}
	return false
}

func (i *Interpreter) checkVisibility(fn *Function, name string, recv Object, mode callMode) Object {
	switch fn.Visibility {
	case "private":
		if !mode.selfCall {
			return newError("private method '%s' called for %s", name, i.describeInstance(recv))
		}
	case "protected":
		if !mode.selfCall {
			self, ok := i.env.Get("self")
			ci, isInst := recv.(*ClassInstance)
			si, selfInst := self.(*ClassInstance)
			if !ok || !isInst || !selfInst || !si.Class.isA(ci.Class) && !ci.Class.isA(si.Class) {
				return newError("protected method '%s' called for %s", name, i.describeInstance(recv))
			}
		}
	}
	return nil
}

func (i *Interpreter) describeInstance(o Object) string {
	if ci, ok := o.(*ClassInstance); ok {
		return "an instance of " + ci.Class.Name
	}
	return i.describeType(o)
}

// callSuperMethod handles `super.method(args)`.
func (i *Interpreter) callSuperMethod(sp *SuperProxy, name string, args []Object, blk Object) Object {
	if sp.After != nil {
		if m := sp.Instance.Class.findMethodAfter(sp.After, name); m != nil {
			return i.callMethodWithBlock(sp.Instance, m.owner, m.fn, args, blk)
		}
		if nat := sp.Instance.Class.findNativeAfter(sp.After, name); nat != nil {
			return nat(i, sp.Instance, args, blk)
		}
	} else if sp.Parent != nil {
		if m := sp.Parent.findMethod(name); m != nil {
			return i.callMethodWithBlock(sp.Instance, m.owner, m.fn, args, blk)
		}
		if nat := sp.Parent.findNative(name); nat != nil {
			return nat(i, sp.Instance, args, blk)
		}
	}
	if res, found := i.callBuiltinMethod(sp.Instance, name, args, blk); found {
		return res
	}
	return newError("super: no parent method '%s'", name)
}

// callClassMember handles ClassName.method: singleton methods, constants, and Class builtins.
func (i *Interpreter) callClassMember(c *ClassObject, name string, args []Object, blk Object, mode callMode) (Object, bool) {
	if fn := c.findStatic(name); fn != nil {
		return i.callStatic(c, fn, args, blk), true
	}
	if nat := c.findNativeStatic(name); nat != nil {
		return nat(i, c, args, blk), true
	}
	if v, ok := c.findConst(name); ok && len(args) == 0 {
		return v, true
	}
	switch name {
	case "new":
		return i.callClassConstructor(c, args, blk), true
	case "allocate":
		return &ClassInstance{Class: c, Fields: map[string]Object{}}, true
	case "name", "to_s", "inspect":
		return &String{Value: c.Name}, true
	case "superclass":
		if c.Parent == nil {
			return &Nil{}, true
		}
		return c.Parent, true
	case "ancestors":
		var out []Object
		for _, a := range c.ancestors() {
			out = append(out, a)
		}
		return &Array{Elements: out}, true
	case "class":
		if c.IsModule {
			return i.lookupGlobalClass("Module"), true
		}
		return i.lookupGlobalClass("Class"), true
	case "instance_methods", "public_instance_methods":
		seen := map[string]bool{}
		var names []string
		inherit := !(len(args) > 0 && !isTruthy(args[0]))
		for _, a := range c.ancestors() {
			if !inherit && a != c {
				break
			}
			for n, fn := range a.Methods {
				if fn.Visibility == "" && !seen[n] {
					seen[n] = true
					names = append(names, n)
				}
			}
			for n := range a.Natives {
				if !seen[n] && !strings.HasSuffix(n, "initialize") {
					seen[n] = true
					names = append(names, n)
				}
			}
		}
		sort.Strings(names)
		out := make([]Object, len(names))
		for idx, n := range names {
			out[idx] = &Symbol{Name: n}
		}
		return &Array{Elements: out}, true
	case "method_defined?", "instance_method_defined?", "public_method_defined?":
		if len(args) == 1 {
			n := symOrString(args[0])
			if m := c.findMethod(n); m != nil && m.fn.Visibility == "" {
				return &Boolean{Value: true}, true
			}
			return &Boolean{Value: c.findNative(n) != nil}, true
		}
	case "private_method_defined?":
		if len(args) == 1 {
			m := c.findMethod(symOrString(args[0]))
			return &Boolean{Value: m != nil && m.fn.Visibility == "private"}, true
		}
	case "const_get":
		if len(args) == 1 {
			if v, ok := c.findConst(symOrString(args[0])); ok {
				return v, true
			}
			return newError("uninitialized constant %s::%s", c.Name, symOrString(args[0])), true
		}
	case "const_set":
		if len(args) == 2 {
			c.Consts[symOrString(args[0])] = args[1]
			return args[1], true
		}
	case "constants":
		var names []string
		for n := range c.Consts {
			names = append(names, n)
		}
		sort.Strings(names)
		out := make([]Object, len(names))
		for idx, n := range names {
			out[idx] = &Symbol{Name: n}
		}
		return &Array{Elements: out}, true
	case "class_variable_get":
		if len(args) == 1 {
			n := symOrString(args[0])
			if o := c.findCvarOwner(n); o != nil {
				return o.Cvars[n], true
			}
			return &Nil{}, true
		}
	case "class_variable_set":
		if len(args) == 2 {
			c.Cvars[symOrString(args[0])] = args[1]
			return args[1], true
		}
	case "instance_variable_get":
		if len(args) == 1 {
			if v, ok := c.Ivars[strings.TrimPrefix(symOrString(args[0]), "@")]; ok {
				return v, true
			}
			return &Nil{}, true
		}
	case "instance_variable_set":
		if len(args) == 2 {
			c.Ivars[strings.TrimPrefix(symOrString(args[0]), "@")] = args[1]
			return args[1], true
		}
	case "include?":
		if len(args) == 1 {
			if m, ok := args[0].(*ClassObject); ok {
				return &Boolean{Value: c.isA(m) && m != c}, true
			}
		}
	case "<", "<=":
		if len(args) == 1 {
			if o, ok := args[0].(*ClassObject); ok {
				if name == "<" && o == c {
					return &Boolean{Value: false}, true
				}
				return &Boolean{Value: c.isA(o)}, true
			}
		}
	case ">", ">=":
		if len(args) == 1 {
			if o, ok := args[0].(*ClassObject); ok {
				if name == ">" && o == c {
					return &Boolean{Value: false}, true
				}
				return &Boolean{Value: o.isA(c)}, true
			}
		}
	case "===":
		if len(args) == 1 {
			return &Boolean{Value: i.isInstanceOf(args[0], c)}, true
		}
	case "include", "extend", "attr_accessor", "attr_reader", "attr_writer", "private", "public",
		"protected", "module_function", "private_class_method", "public_class_method", "alias_method",
		"define_method", "private_constant", "attr", "remove_method", "undef_method":
		return i.classMacro(c, name, args, blk), true
	case "instance_method":
		if len(args) == 1 {
			return &BoundMethod{Recv: nil, Name: symOrString(args[0])}, true
		}
	case "class_eval", "class_exec", "module_eval":
		if blk != nil {
			if fn, ok := blk.(*Function); ok {
				env := NewEnclosedEnvironment(fn.Env)
				env.Set("self", c)
				env.Set("__defclass__", c)
				env.Set("__visibility__", &String{Value: ""})
				cp := *fn
				return i.invoke(&cp, nil, env, nil), true
			}
		}
	}
	return nil, false
}

func symOrString(o Object) string {
	switch v := o.(type) {
	case *Symbol:
		return v.Name
	case *String:
		return v.Value
	}
	return o.Inspect()
}

// classMacro implements include/extend/attr_*/private/... for class bodies.
func (i *Interpreter) classMacro(c *ClassObject, name string, args []Object, blk Object) Object {
	switch name {
	case "include":
		for _, a := range args {
			m, ok := a.(*ClassObject)
			if !ok {
				return newError("wrong argument type %s (expected Module)", a.Type())
			}
			if !c.isA(m) {
				c.Includes = append(c.Includes, m)
			}
			if hook := m.findStatic("included"); hook != nil {
				if r := i.callStatic(m, hook, []Object{c}, nil); isError(r) {
					return r
				}
			}
		}
		return c
	case "extend":
		for _, a := range args {
			m, ok := a.(*ClassObject)
			if !ok {
				return newError("wrong argument type %s (expected Module)", a.Type())
			}
			c.Extends = append(c.Extends, m)
			if hook := m.findStatic("extended"); hook != nil {
				if r := i.callStatic(m, hook, []Object{c}, nil); isError(r) {
					return r
				}
			}
		}
		return c
	case "attr_accessor", "attr_reader", "attr_writer", "attr":
		var out []Object
		for _, a := range flattenArgs(args) {
			n := symOrString(a)
			c.Props = append(c.Props, n)
			c.defineAccessor(n, name != "attr_writer", name == "attr_accessor" || name == "attr_writer")
			if name != "attr_writer" {
				out = append(out, &Symbol{Name: n})
			}
			if name == "attr_accessor" || name == "attr_writer" {
				out = append(out, &Symbol{Name: n + "="})
			}
		}
		return &Array{Elements: out}
	case "private", "public", "protected":
		return i.setVisibility(c, name, args)
	case "module_function":
		if len(args) == 0 {
			i.env.Set("__module_function__", &Boolean{Value: true})
			return &Nil{}
		}
		for _, a := range flattenArgs(args) {
			n := symOrString(a)
			if fn, ok := c.Methods[n]; ok {
				c.Statics[n] = fn
			}
		}
		return &Nil{}
	case "private_class_method", "public_class_method", "private_constant", "remove_method", "undef_method":
		if name == "remove_method" || name == "undef_method" {
			for _, a := range flattenArgs(args) {
				delete(c.Methods, symOrString(a))
			}
		}
		return &Nil{}
	case "alias_method":
		if len(args) == 2 {
			return i.aliasMethod(c, symOrString(args[0]), symOrString(args[1]))
		}
		return newError("wrong number of arguments (given %d, expected 2)", len(args))
	case "define_method":
		if len(args) >= 1 {
			n := symOrString(args[0])
			var body Object = blk
			if len(args) > 1 {
				body = args[1]
			}
			fn, ok := body.(*Function)
			if !ok {
				return newError("tried to create Proc object without a block")
			}
			cp := *fn
			cp.Name = n
			cp.IsBlock = false
			c.Methods[n] = &cp
			return &Symbol{Name: n}
		}
	}
	return &Nil{}
}

func flattenArgs(args []Object) []Object {
	var out []Object
	for _, a := range args {
		if arr, ok := a.(*Array); ok {
			out = append(out, arr.Elements...)
		} else {
			out = append(out, a)
		}
	}
	return out
}

func (i *Interpreter) aliasMethod(c *ClassObject, newName, oldName string) Object {
	if m := c.findMethod(oldName); m != nil {
		cp := *m.fn
		c.Methods[newName] = &cp
		return &Symbol{Name: newName}
	}
	if nat := c.findNative(oldName); nat != nil {
		c.Natives[newName] = nat
		return &Symbol{Name: newName}
	}
	// Alias to a builtin method: delegate.
	c.Natives[newName] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
		return in.callMember(self, oldName, args, blk, callMode{isCall: true, selfCall: true})
	}
	return &Symbol{Name: newName}
}

func (i *Interpreter) setVisibility(c *ClassObject, vis string, args []Object) Object {
	level := vis
	if vis == "public" {
		level = ""
	}
	if len(args) == 0 {
		i.env.Set("__visibility__", &String{Value: level})
		i.env.Set("__module_function__", &Boolean{Value: false})
		return &Nil{}
	}
	for _, a := range flattenArgs(args) {
		n := symOrString(a)
		if m := c.findMethod(n); m != nil {
			if m.owner != c {
				cp := *m.fn
				cp.Visibility = level
				c.Methods[n] = &cp
			} else {
				m.fn.Visibility = level
			}
		}
	}
	if len(args) == 1 {
		return args[0]
	}
	return &Array{Elements: args}
}

// isInstanceOf implements is_a? / === for classes, including builtin type names.
func (i *Interpreter) isInstanceOf(o Object, c *ClassObject) bool {
	switch v := o.(type) {
	case *ClassInstance:
		return v.Class.isA(c)
	case *ClassObject:
		return c.Name == "Class" || c.Name == "Module" || c.Name == "Object"
	}
	switch c.Name {
	case "Object", "BasicObject":
		return true
	case "Integer", "Numeric", "Comparable":
		if _, ok := o.(*Integer); ok {
			return true
		}
		if c.Name == "Numeric" || c.Name == "Comparable" {
			_, ok := o.(*Float)
			return ok
		}
	case "Float":
		_, ok := o.(*Float)
		return ok
	case "String":
		_, ok := o.(*String)
		return ok
	case "Symbol":
		_, ok := o.(*Symbol)
		return ok
	case "Array":
		_, ok := o.(*Array)
		return ok
	case "Hash":
		_, ok := o.(*Hash)
		return ok
	case "NilClass":
		_, ok := o.(*Nil)
		return ok
	case "TrueClass":
		b, ok := o.(*Boolean)
		return ok && b.Value
	case "FalseClass":
		b, ok := o.(*Boolean)
		return ok && !b.Value
	case "Range":
		_, ok := o.(*Range)
		return ok
	case "Proc", "Function":
		_, ok := o.(*Function)
		return ok
	case "Regexp":
		_, ok := o.(*RegexObject)
		return ok
	}
	if si, ok := o.(*StructInstance); ok {
		return si.Struct.Name == c.Name
	}
	return false
}

func (i *Interpreter) lookupGlobalClass(name string) Object {
	if v, ok := i.env.Get(name); ok {
		return v
	}
	return &Nil{}
}

// className returns the Ruby-style class name of a value.
func (i *Interpreter) className(o Object) string {
	switch v := o.(type) {
	case *Integer:
		return "Integer"
	case *Float:
		return "Float"
	case *String:
		return "String"
	case *Symbol:
		return "Symbol"
	case *Array:
		return "Array"
	case *Hash:
		return "Hash"
	case *Nil:
		return "NilClass"
	case *Boolean:
		if v.Value {
			return "TrueClass"
		}
		return "FalseClass"
	case *Range:
		return "Range"
	case *Function, *Builtin, *BoundMethod:
		return "Proc"
	case *ClassInstance:
		return v.Class.Name
	case *StructInstance:
		return v.Struct.Name
	case *ClassObject:
		if v.IsModule {
			return "Module"
		}
		return "Class"
	case *RegexObject:
		return "Regexp"
	}
	return string(o.Type())
}

// evalDotExpression evaluates obj.name (a zero-argument member access).
func (i *Interpreter) evalDotExpression(node *ast.DotExpression) Object {
	if node.Token.Type == lexer.IVAR {
		return i.getIvar(node.Field.Value)
	}
	left := i.evalReceiver(node.Left)
	if isError(left) {
		return left
	}
	if node.Safe {
		if _, isNil := left.(*Nil); isNil {
			return &Nil{}
		}
	}
	if node.Token.Type == lexer.DOUBLE_COLON {
		if c, ok := left.(*ClassObject); ok {
			if v, ok := c.findConst(node.Field.Value); ok {
				return v
			}
		}
	}
	selfCall := false
	if id, ok := node.Left.(*ast.Identifier); ok && id.Value == "self" {
		selfCall = true
	}
	return i.callMember(left, node.Field.Value, nil, nil, callMode{selfCall: selfCall})
}

// getIvar reads @name from the current self.
func (i *Interpreter) getIvar(name string) Object {
	self, ok := i.env.Get("self")
	if !ok {
		return &Nil{}
	}
	switch s := self.(type) {
	case *ClassInstance:
		if v, ok := s.Fields[name]; ok {
			return v
		}
	case *ClassObject:
		if v, ok := s.Ivars[name]; ok {
			return v
		}
	case *StructInstance:
		if v, ok := s.Fields[name]; ok {
			return v
		}
	}
	return &Nil{}
}

func (i *Interpreter) setIvar(name string, v Object) Object {
	self, ok := i.env.Get("self")
	if !ok {
		return newError("instance variable @%s used outside of an object", name)
	}
	switch s := self.(type) {
	case *ClassInstance:
		if s.Frozen {
			return newError("can't modify frozen %s", s.Class.Name)
		}
		s.Fields[name] = v
	case *ClassObject:
		s.Ivars[name] = v
	case *StructInstance:
		s.Fields[name] = v
	default:
		return newError("instance variable @%s used outside of an object", name)
	}
	return v
}

// evalDotAssignment evaluates obj.field = value.
func (i *Interpreter) evalDotAssignment(node *ast.DotAssignment) Object {
	if node.Token.Type == lexer.IVAR {
		value := i.eval(node.Value)
		if isError(value) {
			return value
		}
		return i.setIvar(node.Field, value)
	}
	left := i.eval(node.Left)
	if isError(left) {
		return left
	}
	value := i.eval(node.Value)
	if isError(value) {
		return value
	}
	return i.assignMember(left, node.Field, value)
}

func (i *Interpreter) assignMember(left Object, field string, value Object) Object {
	switch obj := left.(type) {
	case *ClassInstance:
		if m := obj.Class.findMethod(field + "="); m != nil {
			if r := i.callMethodWithBlock(obj, m.owner, m.fn, []Object{value}, nil); isError(r) {
				return r
			}
			return value
		}
		if nat := obj.Class.findNative(field + "="); nat != nil {
			if r := nat(i, obj, []Object{value}, nil); isError(r) {
				return r
			}
			return value
		}
		if obj.Frozen {
			return newError("can't modify frozen %s", obj.Class.Name)
		}
		obj.Fields[field] = value
		return value
	case *ClassObject:
		if fn := obj.findStatic(field + "="); fn != nil {
			i.callStatic(obj, fn, []Object{value}, nil)
			return value
		}
		obj.Ivars[field] = value
		return value
	case *StructInstance:
		if _, exists := obj.Fields[field]; !exists {
			return newError("undefined field '%s' in struct '%s'", field, obj.Struct.Name)
		}
		if obj.Frozen {
			return newError("can't modify frozen %s", obj.Struct.Name)
		}
		obj.Fields[field] = value
		return value
	case *Hash:
		obj.set(field, &String{Value: field}, value)
		return value
	}
	return newError("dot assignment not supported for %s", left.Type())
}
