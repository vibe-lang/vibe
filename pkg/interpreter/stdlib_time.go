package interpreter

import (
	"fmt"
	"time"
)

// TimeObject wraps Go's time.Time for the Vibe language.
type TimeObject struct {
	Value time.Time
}

func (t *TimeObject) Type() ObjectType { return "TIME" }
func (t *TimeObject) Inspect() string  { return t.Value.Format(time.RFC3339) }

func loadTime(env *Environment) {
	// Time namespace hash for static methods
	timeNS := &Hash{
		Pairs: map[string]Object{},
		Order: []string{},
	}

	// Time.now() -- returns the current time
	timeNS.Pairs["now"] = &Builtin{Fn: func(args ...Object) Object {
		return &TimeObject{Value: time.Now()}
	}}
	timeNS.Order = append(timeNS.Order, "now")

	// Time.parse(str, layout) -- parse a time string
	timeNS.Pairs["parse"] = &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return newError("Time.parse: expected 2 arguments (string, layout), got %d", len(args))
		}
		str, ok := args[0].(*String)
		if !ok {
			return newError("Time.parse: first argument must be a string, got %s", args[0].Type())
		}
		layout, ok := args[1].(*String)
		if !ok {
			return newError("Time.parse: second argument must be a string, got %s", args[1].Type())
		}
		t, err := time.Parse(layout.Value, str.Value)
		if err != nil {
			return newError("Time.parse: %s", err.Error())
		}
		return &TimeObject{Value: t}
	}}
	timeNS.Order = append(timeNS.Order, "parse")

	// Time.since(time) -- returns milliseconds elapsed since the given time
	timeNS.Pairs["since"] = &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("Time.since: expected 1 argument, got %d", len(args))
		}
		t, ok := args[0].(*TimeObject)
		if !ok {
			return newError("Time.since: argument must be a Time, got %s", args[0].Type())
		}
		return &Integer{Value: time.Since(t.Value).Milliseconds()}
	}}
	timeNS.Order = append(timeNS.Order, "since")

	// Time.measure(fn) -- measures execution time of a function in milliseconds
	timeNS.Pairs["measure"] = &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("Time.measure: expected 1 argument (function), got %d", len(args))
		}
		start := time.Now()
		interp := getBuiltinInterpreter()
		interp.applyFunction(args[0], []Object{})
		elapsed := time.Since(start).Milliseconds()
		return &Integer{Value: elapsed}
	}}
	timeNS.Order = append(timeNS.Order, "measure")

	env.Set("Time", timeNS)

	// -----------------------------------------------------------------------
	// Field access for TimeObject (year, month, day, hour, minute, second, unix, format)
	// -----------------------------------------------------------------------

	env.Set("year", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("year: expected 1 argument, got %d", len(args))
		}
		t, ok := args[0].(*TimeObject)
		if !ok {
			return newError("year: argument must be a Time, got %s", args[0].Type())
		}
		return &Integer{Value: int64(t.Value.Year())}
	}})

	env.Set("month", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("month: expected 1 argument, got %d", len(args))
		}
		t, ok := args[0].(*TimeObject)
		if !ok {
			return newError("month: argument must be a Time, got %s", args[0].Type())
		}
		return &Integer{Value: int64(t.Value.Month())}
	}})

	env.Set("day", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("day: expected 1 argument, got %d", len(args))
		}
		t, ok := args[0].(*TimeObject)
		if !ok {
			return newError("day: argument must be a Time, got %s", args[0].Type())
		}
		return &Integer{Value: int64(t.Value.Day())}
	}})

	env.Set("hour", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("hour: expected 1 argument, got %d", len(args))
		}
		t, ok := args[0].(*TimeObject)
		if !ok {
			return newError("hour: argument must be a Time, got %s", args[0].Type())
		}
		return &Integer{Value: int64(t.Value.Hour())}
	}})

	env.Set("minute", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("minute: expected 1 argument, got %d", len(args))
		}
		t, ok := args[0].(*TimeObject)
		if !ok {
			return newError("minute: argument must be a Time, got %s", args[0].Type())
		}
		return &Integer{Value: int64(t.Value.Minute())}
	}})

	env.Set("second", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("second: expected 1 argument, got %d", len(args))
		}
		t, ok := args[0].(*TimeObject)
		if !ok {
			return newError("second: argument must be a Time, got %s", args[0].Type())
		}
		return &Integer{Value: int64(t.Value.Second())}
	}})

	env.Set("unix", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 1 {
			return newError("unix: expected 1 argument, got %d", len(args))
		}
		t, ok := args[0].(*TimeObject)
		if !ok {
			return newError("unix: argument must be a Time, got %s", args[0].Type())
		}
		return &Integer{Value: t.Value.Unix()}
	}})

	env.Set("format_time", &Builtin{Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return newError("format_time: expected 2 arguments (time, layout), got %d", len(args))
		}
		t, ok := args[0].(*TimeObject)
		if !ok {
			return newError("format_time: first argument must be a Time, got %s", args[0].Type())
		}
		layout, ok := args[1].(*String)
		if !ok {
			return newError("format_time: second argument must be a string, got %s", args[1].Type())
		}
		return &String{Value: t.Value.Format(layout.Value)}
	}})

	// Suppress unused import
	_ = fmt.Sprintf
}
