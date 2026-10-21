package interpreter

import (
	"strings"

	"github.com/vibe-lang/vibe/pkg/ast"
)

// TypeAlias is the runtime value of `type Name = ...`.
type TypeAlias struct {
	Name string
	Expr ast.Expression
}

func (t *TypeAlias) Type() ObjectType { return "TYPE" }
func (t *TypeAlias) Inspect() string  { return "type " + t.Name + " = " + t.Expr.String() }

// Interface is the runtime value of `interface Name ... end`.
type Interface struct {
	Name    string
	Methods []*ast.MethodSignature
}

func (t *Interface) Type() ObjectType { return "INTERFACE" }
func (t *Interface) Inspect() string  { return "interface " + t.Name }

func (i *Interpreter) evalTypeAlias(node *ast.TypeAliasStatement) Object {
	a := &TypeAlias{Name: node.Name.Value, Expr: node.Type}
	i.env.Set(a.Name, a)
	return a
}

func (i *Interpreter) evalInterface(node *ast.InterfaceStatement) Object {
	t := &Interface{Name: node.Name.Value, Methods: node.Methods}
	i.env.Set(t.Name, t)
	return t
}

// splitTypeArgs splits "a, b<c, d>" at top-level commas.
func splitTypeArgs(s string) []string {
	var out []string
	depth, start := 0, 0
	for k, r := range s {
		switch r {
		case '<', '[', '(':
			depth++
		case '>', ']', ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:k]))
				start = k + 1
			}
		}
	}
	return append(out, strings.TrimSpace(s[start:]))
}

// typeMatches reports whether val satisfies the type expression. Names that are
// not known types (for example generic type parameters) accept any value.
func (i *Interpreter) typeMatches(val Object, expr ast.Expression, env *Environment, params []string) bool {
	switch t := expr.(type) {
	case nil:
		return true
	case *ast.UnionTypeAnnotation:
		for _, m := range t.Types {
			if i.typeMatches(val, m, env, params) {
				return true
			}
		}
		return false
	case *ast.ArrayTypeAnnotation:
		arr, ok := val.(*Array)
		if !ok {
			return false
		}
		for _, e := range arr.Elements {
			if !i.typeMatches(e, t.BaseType, env, params) {
				return false
			}
		}
		return true
	case *ast.CompoundTypeAnnotation:
		var elems []Object
		switch v := val.(type) {
		case *Array:
			elems = v.Elements
		case *Compound:
			elems = v.Elements
		default:
			return false
		}
		if len(elems) != len(t.Types) {
			return false
		}
		for k, e := range elems {
			if !i.typeMatches(e, t.Types[k], env, params) {
				return false
			}
		}
		return true
	case *ast.TypeAnnotation:
		if t.IsCompoundType {
			return true
		}
		return i.typeNameMatches(val, t.Name, env, params)
	case *ast.Identifier:
		return i.typeNameMatches(val, t.Value, env, params)
	}
	return true
}

func (i *Interpreter) typeNameMatches(val Object, name string, env *Environment, params []string) bool {
	for _, p := range params {
		if p == name {
			return true
		}
	}
	if strings.HasPrefix(name, "(") {
		return isCallable(val)
	}
	base, args := name, []string(nil)
	if k := strings.Index(name, "<"); k > 0 && strings.HasSuffix(name, ">") {
		base = name[:k]
		args = splitTypeArgs(name[k+1 : len(name)-1])
	}
	for strings.HasSuffix(base, "[]") {
		arr, ok := val.(*Array)
		if !ok {
			return false
		}
		inner := strings.TrimSuffix(base, "[]")
		for _, e := range arr.Elements {
			if !i.typeNameMatches(e, inner, env, params) {
				return false
			}
		}
		return true
	}
	switch strings.ToLower(base) {
	case "any", "object", "void", "unknown":
		return true
	case "int", "integer":
		_, ok := val.(*Integer)
		return ok
	case "float":
		switch val.(type) {
		case *Float, *Integer:
			return true
		}
		return false
	case "number", "numeric":
		switch val.(type) {
		case *Float, *Integer:
			return true
		}
		return false
	case "string":
		_, ok := val.(*String)
		return ok
	case "boolean", "bool":
		_, ok := val.(*Boolean)
		return ok
	case "nil", "null":
		_, ok := val.(*Nil)
		return ok
	case "symbol":
		_, ok := val.(*Symbol)
		return ok
	case "range":
		_, ok := val.(*Range)
		return ok
	case "proc", "function", "fn", "callable", "lambda":
		return isCallable(val)
	case "array", "list":
		arr, ok := val.(*Array)
		if !ok {
			return false
		}
		if len(args) == 1 {
			for _, e := range arr.Elements {
				if !i.typeNameMatches(e, args[0], env, params) {
					return false
				}
			}
		}
		return true
	case "hash", "map", "dict":
		h, ok := val.(*Hash)
		if !ok {
			return false
		}
		if len(args) == 2 {
			for _, k := range h.Order {
				var key Object = &String{Value: k}
				if ko, has := h.Keys[k]; has {
					key = ko
				}
				if !i.typeNameMatches(key, args[0], env, params) || !i.typeNameMatches(h.Pairs[k], args[1], env, params) {
					return false
				}
			}
		}
		return true
	}
	decl, ok := env.Get(base)
	if !ok {
		return true
	}
	switch d := decl.(type) {
	case *TypeAlias:
		return i.typeMatches(val, d.Expr, env, params)
	case *Interface:
		for _, m := range d.Methods {
			if !i.respondTo(val, m.Name, true) {
				return false
			}
		}
		return true
	case *ClassObject:
		return i.isInstanceOf(val, d)
	case *Struct:
		si, ok := val.(*StructInstance)
		return ok && si.Struct == d
	}
	return true
}

// typeLabel names a value's type for error messages.
func (i *Interpreter) typeLabel(val Object) string {
	if ci, ok := val.(*ClassInstance); ok {
		return ci.Class.Name
	}
	return objectTypeName(val)
}

// checkParamType validates one bound argument against its annotation.
func (i *Interpreter) checkParamType(fn *Function, idx int, val Object, env *Environment) *Error {
	if idx >= len(fn.ParamTypes) || fn.ParamTypes[idx] == nil {
		return nil
	}
	if !i.typeMatches(val, fn.ParamTypes[idx], env, fn.TypeParams) {
		return newErrorClass("TypeError", "type mismatch for parameter '%s': expected %s, got %s",
			fn.Parameters[idx].Value, fn.ParamTypes[idx].String(), i.typeLabel(val))
	}
	return nil
}

func (i *Interpreter) checkReturnType(fn *Function, val Object, env *Environment) *Error {
	if fn.ReturnType == nil || fn.IsBlock || fn.Name == "initialize" {
		return nil
	}
	if !i.typeMatches(val, fn.ReturnType, env, fn.TypeParams) {
		return newErrorClass("TypeError", "type mismatch: %s must return %s, got %s",
			fn.Name, fn.ReturnType.String(), i.typeLabel(val))
	}
	return nil
}
