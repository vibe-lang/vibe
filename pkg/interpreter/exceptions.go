package interpreter

import (
	"strings"

	"github.com/vibe-lang/vibe/pkg/ast"
)

// RetrySignal is produced by `retry` inside a rescue clause.
type RetrySignal struct{}

const RETRY_OBJ = "RETRY"

func (r *RetrySignal) Type() ObjectType { return RETRY_OBJ }
func (r *RetrySignal) Inspect() string  { return "retry" }

// exceptionTree lists builtin exception classes as {name, parent}.
var exceptionTree = [][2]string{
	{"Exception", ""},
	{"ScriptError", "Exception"},
	{"NotImplementedError", "ScriptError"},
	{"StandardError", "Exception"},
	{"RuntimeError", "StandardError"},
	{"ArgumentError", "StandardError"},
	{"TypeError", "StandardError"},
	{"ZeroDivisionError", "StandardError"},
	{"NameError", "StandardError"},
	{"NoMethodError", "NameError"},
	{"KeyError", "IndexError"},
	{"IndexError", "StandardError"},
	{"StopIteration", "IndexError"},
	{"RangeError", "StandardError"},
	{"IOError", "StandardError"},
	{"FrozenError", "RuntimeError"},
	{"FloatDomainError", "RangeError"},
	{"SystemExit", "Exception"},
	{"NoMatchingPatternError", "StandardError"},
	{"NoMatchingPatternKeyError", "NoMatchingPatternError"},
	{"RegexpError", "StandardError"},
	{"EncodingError", "StandardError"},
	{"SecurityError", "Exception"},
	{"Interrupt", "Exception"},
}

// registerExceptions installs the builtin exception hierarchy.
func registerExceptions(i *Interpreter) {
	env := i.env
	classes := map[string]*ClassObject{}
	// IndexError must exist before KeyError; process in two passes.
	pending := exceptionTree
	for len(pending) > 0 {
		var next [][2]string
		for _, e := range pending {
			var parent *ClassObject
			if e[1] != "" {
				parent = classes[e[1]]
				if parent == nil {
					next = append(next, e)
					continue
				}
			}
			c := newClassObject(e[0])
			c.Parent = parent
			classes[e[0]] = c
			env.Set(e[0], c)
		}
		pending = next
	}

	base := classes["Exception"]
	base.Natives["initialize"] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
		inst := self.(*ClassInstance)
		args = dropKw(args)
		if len(args) > 0 {
			inst.Fields["message"] = args[0]
		}
		return &Nil{}
	}
	base.Natives["message"] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
		return &String{Value: in.exceptionMessage(self.(*ClassInstance))}
	}
	base.Natives["to_s"] = base.Natives["message"]
	base.Natives["full_message"] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
		inst := self.(*ClassInstance)
		return &String{Value: inst.Class.Name + ": " + in.exceptionMessage(inst)}
	}
	base.Natives["inspect"] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
		inst := self.(*ClassInstance)
		return &String{Value: "#<" + inst.Class.Name + ": " + in.exceptionMessage(inst) + ">"}
	}
	base.Natives["backtrace"] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
		return &Array{}
	}
	base.Natives["cause"] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
		if v, ok := self.(*ClassInstance).Fields["__cause__"]; ok {
			return v
		}
		return &Nil{}
	}
	base.NativeStat["exception"] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
		return in.callClassConstructor(self.(*ClassObject), args, blk)
	}

	env.Set("raise", &Builtin{Fn: func(args ...Object) Object { return getBuiltinInterpreter().builtinRaise(args) }})
	env.Set("fail", &Builtin{Fn: func(args ...Object) Object { return getBuiltinInterpreter().builtinRaise(args) }})
}

func dropKw(args []Object) []Object {
	if n := len(args); n > 0 {
		if h, ok := args[n-1].(*Hash); ok && h.IsKw {
			return args[:n-1]
		}
	}
	return args
}

func (i *Interpreter) isException(inst *ClassInstance) bool {
	for k := inst.Class; k != nil; k = k.Parent {
		if k.Name == "Exception" && k.Parent == nil {
			return true
		}
	}
	return false
}

func (i *Interpreter) exceptionMessage(inst *ClassInstance) string {
	if m, ok := inst.Fields["message"]; ok {
		if _, isNil := m.(*Nil); !isNil {
			return i.display(m)
		}
	}
	return inst.Class.Name
}

func (i *Interpreter) globalClass(name string) *ClassObject {
	if v, ok := i.rootEnv().Get(name); ok {
		if c, ok := v.(*ClassObject); ok {
			return c
		}
	}
	return nil
}

func (i *Interpreter) rootEnv() *Environment {
	e := i.env
	for e.outer != nil {
		e = e.outer
	}
	return e
}

func (i *Interpreter) newException(class string, msg string) *ClassInstance {
	c := i.globalClass(class)
	if c == nil {
		c = i.globalClass("StandardError")
	}
	return &ClassInstance{Class: c, Fields: map[string]Object{"message": &String{Value: msg}}}
}

// builtinRaise implements raise / fail.
func (i *Interpreter) builtinRaise(args []Object) Object {
	args = dropKw(args)
	var cause Object
	if c, ok := i.rootEnv().Get("$!"); ok {
		cause = c
	}
	switch len(args) {
	case 0:
		if cause != nil {
			if _, isNil := cause.(*Nil); !isNil {
				return &ThrowSignal{Value: cause}
			}
		}
		return &ThrowSignal{Value: i.newException("RuntimeError", "unhandled exception")}
	}
	var exc Object
	switch first := args[0].(type) {
	case *String:
		exc = i.newException("RuntimeError", first.Value)
	case *ClassObject:
		rest := args[1:min(len(args), 2)]
		inst := i.callClassConstructor(first, rest, nil)
		if isError(inst) {
			return inst
		}
		exc = inst
	case *ClassInstance:
		exc = first
		if len(args) > 1 {
			first.Fields["message"] = args[1]
		}
	default:
		exc = i.newException("TypeError", "exception class/object expected")
	}
	if ci, ok := exc.(*ClassInstance); ok && cause != nil && cause != Object(ci) {
		if _, isNil := cause.(*Nil); !isNil {
			if _, has := ci.Fields["__cause__"]; !has {
				ci.Fields["__cause__"] = cause
			}
		}
	}
	return &ThrowSignal{Value: exc}
}

// errorToException converts a runtime error into an exception instance.
func (i *Interpreter) errorToException(e *Error) *ClassInstance {
	msg := e.Message
	class := "RuntimeError"
	switch {
	case strings.Contains(msg, "division by zero") || strings.Contains(msg, "divided by 0") || strings.Contains(msg, "modulo by zero"):
		class = "ZeroDivisionError"
	case strings.HasPrefix(msg, "undefined method") || strings.Contains(msg, "is not a method"):
		class = "NoMethodError"
	case strings.HasPrefix(msg, "identifier not found") || strings.HasPrefix(msg, "uninitialized constant"):
		class = "NameError"
	case strings.Contains(msg, "wrong number of arguments") || strings.Contains(msg, "keyword:") ||
		strings.Contains(msg, "invalid value for"):
		class = "ArgumentError"
	case strings.HasPrefix(msg, "key not found"):
		class = "KeyError"
	case strings.Contains(msg, "out of bounds") || strings.Contains(msg, "out of range") || strings.HasPrefix(msg, "index "):
		class = "IndexError"
	case strings.HasPrefix(msg, "type mismatch") || strings.Contains(msg, "can't be coerced") ||
		strings.Contains(msg, "no implicit conversion") || strings.Contains(msg, "type error") ||
		strings.HasPrefix(msg, "unknown operator") || strings.HasPrefix(msg, "unsupported"):
		class = "TypeError"
	case strings.Contains(msg, "frozen"):
		class = "FrozenError"
	case strings.Contains(msg, "iteration reached an end"):
		class = "StopIteration"
	}
	if e.Class != "" {
		class = e.Class
	}
	return i.newException(class, msg)
}

// matchesRescue reports whether a rescue clause handles the thrown value.
func (i *Interpreter) matchesRescue(rc *ast.RescueClause, val Object) (bool, Object) {
	if len(rc.Types) == 0 {
		if ci, ok := val.(*ClassInstance); ok {
			std := i.globalClass("StandardError")
			return ci.Class.isA(std) || !i.isException(ci), nil
		}
		return true, nil
	}
	for _, t := range rc.Types {
		tv := i.eval(t)
		if isError(tv) {
			return false, tv
		}
		candidates := []Object{tv}
		if arr, ok := tv.(*Array); ok {
			candidates = arr.Elements
		}
		for _, c := range candidates {
			if cls, ok := c.(*ClassObject); ok && i.isInstanceOf(val, cls) {
				return true, nil
			}
		}
	}
	return false, nil
}

func isControl(o Object) bool {
	if o == nil {
		return false
	}
	switch o.Type() {
	case RETURN_VALUE_OBJ, ERROR_OBJ, BREAK_OBJ, CONTINUE_OBJ, THROW_OBJ, RETRY_OBJ:
		return true
	}
	return false
}

// evalTryExpression handles try/catch/finally and begin/rescue/else/ensure.
func (i *Interpreter) evalTryExpression(node *ast.TryExpression) Object {
	var final Object
	for {
		final = i.evalBlockStatement(node.Body)
		retry := false

		var thrown Object
		switch r := final.(type) {
		case *ThrowSignal:
			thrown = r.Value
		case *Error:
			if r.Signal == nil {
				thrown = i.errorToException(r)
			}
		}

		if thrown == nil {
			if node.ElseBody != nil && !isControl(final) {
				final = i.evalBlockStatement(node.ElseBody)
			}
			break
		}

		handled := false
		for _, rc := range node.Rescues {
			ok, err := i.matchesRescue(rc, thrown)
			if err != nil {
				final = err
				handled = true
				break
			}
			if !ok {
				continue
			}
			if rc.Var != nil {
				i.env.Set(rc.Var.Value, thrown)
			}
			prev, _ := i.rootEnv().Get("$!")
			i.rootEnv().Set("$!", thrown)
			final = i.evalBlockStatement(rc.Body)
			if prev != nil {
				i.rootEnv().Set("$!", prev)
			} else {
				i.rootEnv().Set("$!", &Nil{})
			}
			if _, isRetry := final.(*RetrySignal); isRetry {
				retry = true
			}
			handled = true
			break
		}

		if !handled && node.CatchBody != nil {
			if node.CatchVar != nil {
				switch r := final.(type) {
				case *ThrowSignal:
					i.env.Set(node.CatchVar.Value, r.Value)
				case *Error:
					i.env.Set(node.CatchVar.Value, &String{Value: r.Message})
				}
			}
			final = i.evalBlockStatement(node.CatchBody)
		}
		if retry {
			continue
		}
		break
	}

	if node.FinallyBody != nil {
		if fr := i.evalBlockStatement(node.FinallyBody); isControl(fr) {
			if _, isThrow := fr.(*ThrowSignal); isThrow || fr.Type() == RETURN_VALUE_OBJ {
				return fr
			}
		}
	}
	return final
}

func (i *Interpreter) evalThrowStatement(node *ast.ThrowStatement) Object {
	value := i.eval(node.Value)
	if isError(value) {
		return value
	}
	return &ThrowSignal{Value: value}
}

// uncaught converts an exception that reached the top level into a reportable error.
func (i *Interpreter) uncaught(t *ThrowSignal) *Error {
	if ci, ok := t.Value.(*ClassInstance); ok && i.isException(ci) {
		return &Error{Message: i.exceptionMessage(ci) + " (" + ci.Class.Name + ")"}
	}
	return &Error{Message: "uncaught throw: " + t.Value.Inspect()}
}
