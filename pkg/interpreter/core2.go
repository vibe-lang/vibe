package interpreter

import (
	"math/rand"
	"time"
)

func randInt63() int64 { return rand.Int63() }

func sleepSeconds(f float64) { time.Sleep(time.Duration(f * float64(time.Second))) }

func (i *Interpreter) installClassNew(name string, fn func(in *Interpreter, args []Object, blk Object) Object) {
	if c, ok := i.env.Get(name); ok {
		if co, ok := c.(*ClassObject); ok {
			co.NativeStat["new"] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
				return fn(in, args, blk)
			}
		}
	}
}

// installComparable adds <, >, <=, >=, between? and clamp, driven by the host's <=>.
func (i *Interpreter) installComparable(c *ClassObject) {
	cmp := func(in *Interpreter, self, other Object) (int, Object) {
		ci, ok := self.(*ClassInstance)
		if !ok {
			return 0, newError("Comparable requires a class instance")
		}
		m := ci.Class.findMethod("<=>")
		if m == nil {
			return 0, newError("undefined method '<=>' for %s", ci.Class.Name)
		}
		r := in.callMethodWithBlock(ci, m.owner, m.fn, []Object{other}, nil)
		if isError(r) {
			return 0, r
		}
		n, ok := toInt64Strict(r)
		if !ok {
			return 0, newErrorClass("ArgumentError", "comparison of %s with %s failed", ci.Class.Name, in.describeCmp(other))
		}
		return int(n), nil
	}
	for _, op := range []string{"<", ">", "<=", ">="} {
		op := op
		c.Natives[op] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
			n, err := cmp(in, self, args[0])
			if err != nil {
				return err
			}
			return compareResult(op, n)
		}
	}
	c.Natives["between?"] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
		a, err := cmp(in, self, args[0])
		if err != nil {
			return err
		}
		b, err := cmp(in, self, args[1])
		if err != nil {
			return err
		}
		return boolObj(a >= 0 && b <= 0)
	}
	c.Natives["clamp"] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
		lo, hi := args[0], Object(nil)
		if len(args) > 1 {
			hi = args[1]
		} else if r, ok := args[0].(*Array); ok && len(r.Elements) == 2 {
			lo, hi = r.Elements[0], r.Elements[1]
		}
		if n, err := cmp(in, self, lo); err != nil {
			return err
		} else if n < 0 {
			return lo
		}
		if hi != nil {
			if n, err := cmp(in, self, hi); err != nil {
				return err
			} else if n > 0 {
				return hi
			}
		}
		return self
	}
}

var enumerableNames = []string{"map", "collect", "flat_map", "select", "filter", "reject", "find", "detect", "find_all", "each_with_index",
	"each_with_object", "reduce", "inject", "sum", "count", "min", "max", "min_by", "max_by", "sort", "sort_by", "group_by", "partition",
	"to_a", "entries", "first", "take", "drop", "take_while", "drop_while", "include?", "member?", "any?", "all?", "none?", "one?", "zip",
	"tally", "uniq", "each_slice", "each_cons", "filter_map", "find_index", "reverse_each", "minmax", "to_h", "chunk_while", "slice_when",
	"lazy", "cycle", "each_entry"}

// installEnumerable adds the Enumerable methods to anything that defines each.
func (i *Interpreter) installEnumerable(c *ClassObject) {
	for _, n := range enumerableNames {
		n := n
		c.Natives[n] = func(in *Interpreter, self Object, args []Object, blk Object) Object {
			items, ok := in.iterItems(self)
			if !ok {
				return newError("%s requires an each method", in.className(self))
			}
			r, found := in.enumMethod(items, n, args, blk, self)
			if !found {
				return newError("undefined method '%s' for %s", n, in.className(self))
			}
			return r
		}
	}
}
