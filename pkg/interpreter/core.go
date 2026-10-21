package interpreter

import (
	"fmt"
	"math"
	"os"
	"strings"
)

func posInf() float64 { return math.Inf(1) }

// newErrorClass builds an Error that is rescued as the named exception class.
func newErrorClass(class string, format string, a ...interface{}) *Error {
	e := newError(format, a...)
	e.Class = class
	return e
}

func registerCore(i *Interpreter) {
	env := i.env
	for _, name := range []string{"Object", "BasicObject", "Kernel", "Comparable", "Enumerable", "Numeric", "Integer", "Float", "String",
		"Symbol", "Array", "Hash", "Range", "Proc", "NilClass", "TrueClass", "FalseClass", "Regexp", "Module", "Class"} {
		if _, exists := env.Get(name); exists {
			continue
		}
		c := newClassObject(name)
		c.IsModule = name == "Kernel" || name == "Comparable" || name == "Enumerable"
		env.Set(name, c)
	}
	if c, ok := env.Get("Comparable"); ok {
		i.installComparable(c.(*ClassObject))
	}
	if c, ok := env.Get("Enumerable"); ok {
		i.installEnumerable(c.(*ClassObject))
	}
	i.installClassNew("Hash", func(in *Interpreter, args []Object, blk Object) Object {
		h := &Hash{Pairs: map[string]Object{}}
		if blk != nil {
			h.Default = blk
		} else if len(args) > 0 {
			h.Default = args[0]
		}
		return h
	})
	i.installClassNew("Array", func(in *Interpreter, args []Object, blk Object) Object {
		a := &Array{}
		if len(args) == 0 {
			return a
		}
		if src, ok := args[0].(*Array); ok {
			a.Elements = append([]Object{}, src.Elements...)
			return a
		}
		n, ok := toInt64Strict(args[0])
		if !ok || n < 0 {
			return newError("negative array size")
		}
		for k := int64(0); k < n; k++ {
			switch {
			case blk != nil:
				v := in.yield(blk, &Integer{Value: k})
				if isError(v) {
					return v
				}
				a.Elements = append(a.Elements, v)
			case len(args) > 1:
				a.Elements = append(a.Elements, args[1])
			default:
				a.Elements = append(a.Elements, &Nil{})
			}
		}
		return a
	})
	i.installClassNew("String", func(in *Interpreter, args []Object, blk Object) Object {
		if len(args) > 0 {
			return &String{Value: in.display(args[0])}
		}
		return &String{}
	})
	i.installClassNew("Object", func(in *Interpreter, args []Object, blk Object) Object {
		return &ClassInstance{Class: in.globalClass("Object"), Fields: map[string]Object{}}
	})
	i.installClassNew("Proc", func(in *Interpreter, args []Object, blk Object) Object {
		if blk == nil {
			return newError("tried to create Proc object without a block")
		}
		return blk
	})
	i.installClassNew("Regexp", func(in *Interpreter, args []Object, blk Object) Object {
		src, _ := in.strArg(args, 0)
		flags := ""
		if len(args) > 1 {
			if n, ok := args[1].(*Integer); ok {
				if n.Value&1 != 0 {
					flags += "i"
				}
				if n.Value&4 != 0 {
					flags += "m"
				}
				if n.Value&2 != 0 {
					flags += "x"
				}
			} else if isTruthy(args[1]) {
				flags = "i"
			}
		}
		if r, ok := args[0].(*RegexObject); ok {
			return r
		}
		re, err := newRegex(src, flags)
		if err != nil {
			return newErrorClass("RegexpError", "%s", err.Error())
		}
		return re
	})
	if c, ok := env.Get("Regexp"); ok {
		rc := c.(*ClassObject)
		rc.Consts["IGNORECASE"] = &Integer{Value: 1}
		rc.Consts["EXTENDED"] = &Integer{Value: 2}
		rc.Consts["MULTILINE"] = &Integer{Value: 4}
		rc.NativeStat["escape"] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
			s, _ := in.strArg(args, 0)
			return &String{Value: regexpQuote(s)}
		}
	}
	if c, ok := env.Get("Integer"); ok {
		c.(*ClassObject).Consts["MAX"] = &Integer{Value: math.MaxInt64}
	}
	if c, ok := env.Get("Float"); ok {
		fc := c.(*ClassObject)
		fc.Consts["INFINITY"] = &Float{Value: math.Inf(1)}
		fc.Consts["NAN"] = &Float{Value: math.NaN()}
		fc.Consts["EPSILON"] = &Float{Value: 2.220446049250313e-16}
		fc.Consts["MAX"] = &Float{Value: math.MaxFloat64}
		fc.Consts["MIN"] = &Float{Value: 2.2250738585072014e-308}
	}

	set := func(name string, fn func(args ...Object) Object) {
		if c, ok := env.Get(name); ok {
			if cls, isClass := c.(*ClassObject); isClass {
				cls.Call = fn
				return
			}
		}
		env.Set(name, &Builtin{Fn: fn})
	}
	errClass := newClassObject("Error")
	if sc, ok := env.Get("StandardError"); ok {
		errClass.Parent, _ = sc.(*ClassObject)
	}
	errClass.Call = builtinError
	env.Set("Error", errClass)

	set("puts", func(args ...Object) Object {
		if len(args) == 0 {
			fmt.Println()
		}
		var emit func(o Object)
		emit = func(o Object) {
			if a, ok := o.(*Array); ok {
				if len(a.Elements) == 0 {
					return
				}
				for _, e := range a.Elements {
					emit(e)
				}
				return
			}
			s := i.display(o)
			fmt.Print(s)
			if !strings.HasSuffix(s, "\n") {
				fmt.Println()
			}
		}
		for _, a := range args {
			emit(a)
		}
		return &Nil{}
	})
	set("print", func(args ...Object) Object {
		for _, a := range args {
			fmt.Print(i.display(a))
		}
		return &Nil{}
	})
	set("p", func(args ...Object) Object {
		for _, a := range args {
			fmt.Println(i.inspectValue(a))
		}
		switch len(args) {
		case 0:
			return &Nil{}
		case 1:
			return args[0]
		}
		return &Array{Elements: args}
	})
	set("pp", func(args ...Object) Object {
		for _, a := range args {
			fmt.Println(i.inspectValue(a))
		}
		if len(args) == 1 {
			return args[0]
		}
		return &Nil{}
	})
	set("printf", func(args ...Object) Object {
		if len(args) == 0 {
			return &Nil{}
		}
		f, _ := i.strArg(args, 0)
		r := i.sprintf(f, args[1:])
		if s, ok := r.(*String); ok {
			fmt.Print(s.Value)
			return &Nil{}
		}
		return r
	})
	set("sprintf", func(args ...Object) Object {
		if len(args) == 0 {
			return newError("too few arguments")
		}
		f, _ := i.strArg(args, 0)
		return i.sprintf(f, args[1:])
	})
	if sp, ok := env.Get("sprintf"); ok {
		env.Set("format", sp)
	}
	set("lambda", func(args ...Object) Object {
		if len(args) == 0 {
			return newError("tried to create Proc object without a block")
		}
		if f, ok := args[len(args)-1].(*Function); ok {
			cp := *f
			cp.IsLambda = true
			cp.IsBlock = false
			return &cp
		}
		return args[len(args)-1]
	})
	set("proc", func(args ...Object) Object {
		if len(args) == 0 {
			return newError("tried to create Proc object without a block")
		}
		return args[len(args)-1]
	})
	set("loop", func(args ...Object) Object {
		if len(args) == 0 {
			return newError("no block given (loop)")
		}
		blk := args[len(args)-1]
		for {
			r := i.yield(blk)
			if e, ok := r.(*Error); ok {
				if e.Class == "StopIteration" {
					return &Nil{}
				}
			}
			if isError(r) {
				return r
			}
		}
	})
	set("typeof", func(args ...Object) Object {
		if len(args) != 1 {
			return newError("wrong number of arguments (given %d, expected 1)", len(args))
		}
		return &String{Value: i.className(args[0])}
	})
	set("freeze", func(args ...Object) Object {
		if len(args) == 1 {
			i.freeze(args[0])
			return args[0]
		}
		return &Nil{}
	})
	set("Integer", func(args ...Object) Object {
		if len(args) == 0 {
			return newError("wrong number of arguments (given 0, expected 1..2)")
		}
		switch v := args[0].(type) {
		case *Integer:
			return v
		case *Float:
			if math.IsNaN(v.Value) || math.IsInf(v.Value, 0) {
				return newErrorClass("FloatDomainError", "%v", v.Value)
			}
			return &Integer{Value: int64(v.Value)}
		case *String:
			s := strings.ReplaceAll(strings.TrimSpace(v.Value), "_", "")
			base := 10
			if len(args) > 1 {
				if b, ok := args[1].(*Integer); ok {
					base = int(b.Value)
				}
			}
			t := strings.TrimLeft(s, "+-")
			switch {
			case strings.HasPrefix(t, "0x") || strings.HasPrefix(t, "0X"):
				base = 16
			case strings.HasPrefix(t, "0b") || strings.HasPrefix(t, "0B"):
				base = 2
			}
			digits := t
			if base == 16 || base == 2 {
				digits = t[2:]
				if len(t) < 2 || !(t[1] == 'x' || t[1] == 'X' || t[1] == 'b' || t[1] == 'B') {
					digits = t
				}
			}
			ok := digits != ""
			for k := 0; k < len(digits); k++ {
				if d := digitVal(digits[k]); d < 0 || d >= base {
					ok = false
				}
			}
			if !ok {
				if len(args) > 1 {
					if h, isH := args[len(args)-1].(*Hash); isH {
						if ex, has := h.Pairs["exception"]; has && !isTruthy(ex) {
							return &Nil{}
						}
					}
				}
				return newErrorClass("ArgumentError", "invalid value for Integer(): %s", strconvQuote(v.Value))
			}
			return &Integer{Value: parseIntPrefix(s, base)}
		case *Nil:
			return newErrorClass("TypeError", "can't convert nil into Integer")
		}
		return newErrorClass("TypeError", "can't convert %s into Integer", i.className(args[0]))
	})
	set("Float", func(args ...Object) Object {
		if len(args) == 0 {
			return newError("wrong number of arguments (given 0, expected 1)")
		}
		switch v := args[0].(type) {
		case *Integer:
			return &Float{Value: float64(v.Value)}
		case *Float:
			return v
		case *String:
			s := strings.ReplaceAll(strings.TrimSpace(v.Value), "_", "")
			var f float64
			if _, err := fmt.Sscanf(s, "%g", &f); err != nil || fmt.Sprint(parseFloatPrefix(s)) == "0" && s != "0" && s != "0.0" || len(s) == 0 {
				if h, isH := args[len(args)-1].(*Hash); isH {
					if ex, has := h.Pairs["exception"]; has && !isTruthy(ex) {
						return &Nil{}
					}
				}
				return newErrorClass("ArgumentError", "invalid value for Float(): %s", strconvQuote(v.Value))
			}
			return &Float{Value: f}
		}
		return newErrorClass("TypeError", "can't convert %s into Float", i.className(args[0]))
	})
	set("String", func(args ...Object) Object { return &String{Value: i.display(args[0])} })
	set("Array", func(args ...Object) Object {
		switch v := args[0].(type) {
		case *Array:
			return v
		case *Nil:
			return &Array{}
		case *Hash:
			return &Array{Elements: i.hashPairs(v)}
		case *Range:
			return v.ToArray()
		}
		return &Array{Elements: []Object{args[0]}}
	})
	set("Hash", func(args ...Object) Object {
		if h, ok := args[0].(*Hash); ok {
			return h
		}
		return &Hash{Pairs: map[string]Object{}}
	})
	set("rand", func(args ...Object) Object {
		if len(args) > 0 {
			if n, ok := args[0].(*Integer); ok && n.Value > 0 {
				return &Integer{Value: randInt63() % n.Value}
			}
			if r, ok := args[0].(*Range); ok && !r.Endless {
				span := r.End - r.Start + 1
				if r.Exclusive {
					span--
				}
				if span > 0 {
					return &Integer{Value: r.Start + randInt63()%span}
				}
			}
		}
		return &Float{Value: float64(randInt63()%1000000) / 1000000}
	})
	set("srand", func(args ...Object) Object { return &Integer{Value: 0} })
	set("sleep", func(args ...Object) Object {
		if len(args) > 0 {
			if f, ok := toFloat(args[0]); ok {
				sleepSeconds(f)
			}
		}
		return &Integer{Value: 0}
	})
	set("abort", func(args ...Object) Object {
		if len(args) > 0 {
			fmt.Fprintln(os.Stderr, i.display(args[0]))
		}
		os.Exit(1)
		return &Nil{}
	})
	set("at_exit", func(args ...Object) Object { return &Nil{} })
	set("gets", func(args ...Object) Object {
		var line string
		if _, err := fmt.Scanln(&line); err != nil {
			return &Nil{}
		}
		return &String{Value: line}
	})
	set("require", func(args ...Object) Object { return boolObj(true) })
	set("require_relative", func(args ...Object) Object { return boolObj(true) })
	set("object_id", func(args ...Object) Object { return &Integer{Value: int64(objectID(args[0]))} })
	set("binding", func(args ...Object) Object { return &Nil{} })
	set("catch", func(args ...Object) Object {
		if len(args) == 0 {
			return newError("no block given (catch)")
		}
		blk := args[len(args)-1]
		var tag Object = &Symbol{Name: "__catch__"}
		if len(args) > 1 {
			tag = args[0]
		}
		r := i.yield(blk, tag)
		if t, ok := r.(*ThrowSignal); ok {
			_ = t
		}
		return r
	})
}

func strconvQuote(s string) string { return fmt.Sprintf("%q", s) }
