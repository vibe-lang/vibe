package interpreter

import (
	"math/rand"
	"sort"
	"strings"
)

func normIndex(idx int64, n int) (int, bool) {
	if idx < 0 {
		idx += int64(n)
	}
	if idx < 0 || idx >= int64(n) {
		return 0, false
	}
	return int(idx), true
}

func (i *Interpreter) arrayMethod(a *Array, name string, args []Object, blk Object) (Object, bool) {
	els := a.Elements
	n := len(els)
	arr := func(o []Object) Object { return &Array{Elements: o} }
	mut := func() Object {
		if a.Frozen {
			return i.frozenErr(a)
		}
		return nil
	}
	switch name {
	case "length", "size":
		return &Integer{Value: int64(n)}, true
	case "empty?":
		return boolObj(n == 0), true
	case "push", "append", "<<":
		if e := mut(); e != nil {
			return e, true
		}
		a.Elements = append(a.Elements, args...)
		return a, true
	case "pop":
		if e := mut(); e != nil {
			return e, true
		}
		if len(args) > 0 {
			k, _ := toInt64Strict(args[0])
			if int(k) > n {
				k = int64(n)
			}
			out := append([]Object{}, els[n-int(k):]...)
			a.Elements = els[:n-int(k)]
			return arr(out), true
		}
		if n == 0 {
			return &Nil{}, true
		}
		a.Elements = els[:n-1]
		return els[n-1], true
	case "shift":
		if e := mut(); e != nil {
			return e, true
		}
		if len(args) > 0 {
			k, _ := toInt64Strict(args[0])
			if int(k) > n {
				k = int64(n)
			}
			out := append([]Object{}, els[:k]...)
			a.Elements = els[k:]
			return arr(out), true
		}
		if n == 0 {
			return &Nil{}, true
		}
		a.Elements = els[1:]
		return els[0], true
	case "unshift", "prepend":
		if e := mut(); e != nil {
			return e, true
		}
		a.Elements = append(append([]Object{}, args...), els...)
		return a, true
	case "insert":
		if e := mut(); e != nil {
			return e, true
		}
		idx, _ := toInt64Strict(args[0])
		if idx < 0 {
			idx += int64(n) + 1
		}
		if idx < 0 {
			return newError("index %d too small for array", idx), true
		}
		for int(idx) > len(a.Elements) {
			a.Elements = append(a.Elements, &Nil{})
		}
		tail := append([]Object{}, a.Elements[idx:]...)
		a.Elements = append(append(a.Elements[:idx], args[1:]...), tail...)
		return a, true
	case "concat":
		if e := mut(); e != nil {
			return e, true
		}
		for _, o := range args {
			if oa, ok := o.(*Array); ok {
				a.Elements = append(a.Elements, oa.Elements...)
			}
		}
		return a, true
	case "replace":
		if e := mut(); e != nil {
			return e, true
		}
		if oa, ok := args[0].(*Array); ok {
			a.Elements = append([]Object{}, oa.Elements...)
		}
		return a, true
	case "clear":
		if e := mut(); e != nil {
			return e, true
		}
		a.Elements = nil
		return a, true
	case "[]", "slice":
		return i.indexGet(a, firstOrIndex(args)), true
	case "[]=":
		if len(args) >= 2 {
			var idx Object = args[0]
			if len(args) == 3 {
				idx = firstOrIndex(args[:2])
			}
			return i.indexSet(a, idx, args[len(args)-1]), true
		}
	case "at":
		return i.indexGet(a, args[0]), true
	case "dig":
		cur := i.indexGet(a, args[0])
		if len(args) == 1 || isError(cur) {
			return cur, true
		}
		if _, isNil := cur.(*Nil); isNil {
			return cur, true
		}
		return i.callMember(cur, "dig", args[1:], nil, callMode{isCall: true}), true
	case "fetch":
		idx, _ := toInt64Strict(args[0])
		if k, ok := normIndex(idx, n); ok {
			return els[k], true
		}
		if blk != nil {
			return i.yield(blk, args[0]), true
		}
		if len(args) > 1 {
			return args[1], true
		}
		return newErrorClass("IndexError", "index %d outside of array bounds: %d...%d", idx, -n, n), true
	case "values_at":
		out := make([]Object, len(args))
		for k, ix := range args {
			out[k] = i.indexGet(a, ix)
		}
		return arr(out), true
	case "last":
		if len(args) == 0 {
			if n == 0 {
				return &Nil{}, true
			}
			return els[n-1], true
		}
		k, _ := toInt64Strict(args[0])
		if int(k) > n {
			k = int64(n)
		}
		return arr(append([]Object{}, els[n-int(k):]...)), true
	case "reverse":
		out := make([]Object, n)
		for k, e := range els {
			out[n-1-k] = e
		}
		return arr(out), true
	case "reverse!":
		if e := mut(); e != nil {
			return e, true
		}
		for x, y := 0, n-1; x < y; x, y = x+1, y-1 {
			els[x], els[y] = els[y], els[x]
		}
		return a, true
	case "rotate":
		if n == 0 {
			return arr(nil), true
		}
		k, _ := argInt(args, 0, 1)
		k = floorMod(k, int64(n))
		return arr(append(append([]Object{}, els[k:]...), els[:k]...)), true
	case "compact":
		var out []Object
		for _, e := range els {
			if _, isNil := e.(*Nil); !isNil {
				out = append(out, e)
			}
		}
		return arr(out), true
	case "compact!":
		if e := mut(); e != nil {
			return e, true
		}
		r, _ := i.arrayMethod(a, "compact", nil, nil)
		changed := len(r.(*Array).Elements) != n
		a.Elements = r.(*Array).Elements
		if !changed {
			return &Nil{}, true
		}
		return a, true
	case "flatten":
		depth := -1
		if len(args) > 0 {
			d, _ := toInt64Strict(args[0])
			depth = int(d)
		}
		return arr(i.flatten(els, depth)), true
	case "flatten!":
		if e := mut(); e != nil {
			return e, true
		}
		a.Elements = i.flatten(els, -1)
		return a, true
	case "uniq!":
		if e := mut(); e != nil {
			return e, true
		}
		out, err := i.uniqObjects(els, blk)
		if err != nil {
			return err, true
		}
		changed := len(out) != n
		a.Elements = out
		if !changed {
			return &Nil{}, true
		}
		return a, true
	case "sort!", "sort_by!", "map!", "collect!", "select!", "filter!", "keep_if", "reject!", "delete_if", "shuffle!":
		if e := mut(); e != nil {
			return e, true
		}
		base := map[string]string{"sort!": "sort", "sort_by!": "sort_by", "map!": "map", "collect!": "map",
			"select!": "select", "filter!": "select", "keep_if": "select", "reject!": "reject", "delete_if": "reject", "shuffle!": "shuffle"}[name]
		r, ok := i.arrayMethod(a, base, args, blk)
		if !ok || isError(r) {
			return r, ok
		}
		changed := len(r.(*Array).Elements) != n
		a.Elements = r.(*Array).Elements
		if (name == "select!" || name == "filter!" || name == "reject!") && !changed {
			return &Nil{}, true
		}
		return a, true
	case "shuffle":
		out := append([]Object{}, els...)
		rand.Shuffle(len(out), func(x, y int) { out[x], out[y] = out[y], out[x] })
		return arr(out), true
	case "sample":
		if n == 0 {
			return &Nil{}, true
		}
		return els[rand.Intn(n)], true
	case "delete":
		if e := mut(); e != nil {
			return e, true
		}
		var out []Object
		var found Object = &Nil{}
		for _, e := range els {
			if i.equal(e, args[0]) {
				found = e
			} else {
				out = append(out, e)
			}
		}
		a.Elements = out
		return found, true
	case "delete_at":
		if e := mut(); e != nil {
			return e, true
		}
		idx, _ := toInt64Strict(args[0])
		k, ok := normIndex(idx, n)
		if !ok {
			return &Nil{}, true
		}
		v := els[k]
		a.Elements = append(append([]Object{}, els[:k]...), els[k+1:]...)
		return v, true
	case "slice!":
		if e := mut(); e != nil {
			return e, true
		}
		r := i.indexGet(a, firstOrIndex(args))
		if _, ok := r.(*Array); ok {
			if rg, isRg := firstOrIndex(args).(*Range); isRg {
				if st, ln, ok := rangeBounds(rg, n); ok {
					a.Elements = append(append([]Object{}, els[:st]...), els[min(st+ln, n):]...)
				}
			}
		} else if idx, ok := toInt64Strict(args[0]); ok {
			if k, ok := normIndex(idx, n); ok {
				a.Elements = append(append([]Object{}, els[:k]...), els[k+1:]...)
			}
		}
		return r, true
	case "fill":
		if e := mut(); e != nil {
			return e, true
		}
		for k := range els {
			if blk != nil {
				els[k] = i.yield(blk, &Integer{Value: int64(k)})
			} else {
				els[k] = args[0]
			}
		}
		return a, true
	case "index", "find_index", "rindex":
		if name == "rindex" {
			for k := n - 1; k >= 0; k-- {
				if len(args) > 0 && i.equal(els[k], args[0]) {
					return &Integer{Value: int64(k)}, true
				}
				if len(args) == 0 && blk != nil && isTruthy(i.yield(blk, els[k])) {
					return &Integer{Value: int64(k)}, true
				}
			}
			return &Nil{}, true
		}
	case "each_index":
		for k := range els {
			if r := i.yield(blk, &Integer{Value: int64(k)}); isError(r) {
				return r, true
			}
		}
		return a, true
	case "assoc":
		for _, e := range els {
			if p, ok := e.(*Array); ok && len(p.Elements) > 0 && i.equal(p.Elements[0], args[0]) {
				return p, true
			}
		}
		return &Nil{}, true
	case "product":
		res := [][]Object{{}}
		lists := [][]Object{els}
		for _, o := range args {
			if oa, ok := o.(*Array); ok {
				lists = append(lists, oa.Elements)
			}
		}
		for _, l := range lists {
			var next [][]Object
			for _, prefix := range res {
				for _, e := range l {
					next = append(next, append(append([]Object{}, prefix...), e))
				}
			}
			res = next
		}
		out := make([]Object, len(res))
		for k, r := range res {
			out[k] = arr(r)
		}
		return arr(out), true
	case "combination", "permutation":
		k := n
		if len(args) > 0 {
			kk, _ := toInt64Strict(args[0])
			k = int(kk)
		}
		var out []Object
		var rec func(start int, cur []Object, used []bool)
		rec = func(start int, cur []Object, used []bool) {
			if len(cur) == k {
				out = append(out, arr(append([]Object{}, cur...)))
				return
			}
			for x := 0; x < n; x++ {
				if name == "combination" && x < start {
					continue
				}
				if used[x] {
					continue
				}
				used[x] = true
				next := x + 1
				rec(next, append(cur, els[x]), used)
				used[x] = false
			}
		}
		if k >= 0 && k <= n {
			rec(0, nil, make([]bool, n))
		}
		if blk != nil {
			for _, c := range out {
				if r := i.yield(blk, c); isError(r) {
					return r, true
				}
			}
			return a, true
		}
		return arr(out), true
	case "transpose":
		if n == 0 {
			return arr(nil), true
		}
		first, ok := els[0].(*Array)
		if !ok {
			return newError("no implicit conversion into Array"), true
		}
		w := len(first.Elements)
		out := make([]Object, w)
		for c := 0; c < w; c++ {
			col := make([]Object, n)
			for r, row := range els {
				ra, ok := row.(*Array)
				if !ok || len(ra.Elements) != w {
					return newErrorClass("IndexError", "element size differs"), true
				}
				col[r] = ra.Elements[c]
			}
			out[c] = arr(col)
		}
		return arr(out), true
	case "bsearch":
		idx := sort.Search(n, func(k int) bool { return isTruthy(i.yield(blk, els[k])) })
		if idx < n {
			return els[idx], true
		}
		return &Nil{}, true
	case "sum":
		allFloat := true
		for _, e := range els {
			if _, ok := e.(*Float); !ok {
				allFloat = false
			}
		}
		if allFloat && n > 0 && blk == nil && len(args) == 0 {
			var sum, c float64
			for _, e := range els {
				x := e.(*Float).Value
				t := sum + x
				if abs(sum) >= abs(x) {
					c += (sum - t) + x
				} else {
					c += (x - t) + sum
				}
				sum = t
			}
			return &Float{Value: sum + c}, true
		}
	case "to_a", "entries":
		return a, true
	case "to_s", "inspect":
		return &String{Value: i.inspectValue(a)}, true
	case "pack":
		return newError("pack is not supported"), true
	case "<=>", "+", "-", "*", "&", "|", "==", "!=", "<", ">", "<=", ">=":
		if len(args) == 1 {
			if name == "<=>" {
				if c, ok := i.compare(a, args[0]); ok {
					return &Integer{Value: int64(c)}, true
				}
				return &Nil{}, true
			}
			return i.infix(name, a, args[0]), true
		}
	case "intersection", "union", "difference":
		op := map[string]string{"intersection": "&", "union": "|", "difference": "-"}[name]
		cur := Object(a)
		for _, o := range args {
			cur = i.infix(op, cur, o)
		}
		return cur, true
	case "intersect?":
		r := i.infix("&", a, args[0])
		if ra, ok := r.(*Array); ok {
			return boolObj(len(ra.Elements) > 0), true
		}
	}
	return i.enumMethod(els, name, args, blk, a)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func (i *Interpreter) hashPairs(h *Hash) []Object {
	out := make([]Object, 0, len(h.Order))
	for _, k := range h.Order {
		out = append(out, &Array{Elements: []Object{h.keyObj(k), h.Pairs[k]}})
	}
	return out
}

func (i *Interpreter) newHashLike(h *Hash) *Hash {
	return &Hash{Pairs: map[string]Object{}, Default: h.Default}
}

func (i *Interpreter) hashMethod(h *Hash, name string, args []Object, blk Object) (Object, bool) {
	mut := func() Object {
		if h.Frozen {
			return i.frozenErr(h)
		}
		return nil
	}
	arr := func(o []Object) Object { return &Array{Elements: o} }
	lookup := func(key Object) (Object, bool) {
		v, ok := h.Pairs[hashKey(key)]
		return v, ok
	}
	keep := func(want bool) (Object, Object) {
		out := i.newHashLike(h)
		for _, k := range h.Order {
			r := i.yield(blk, h.keyObj(k), h.Pairs[k])
			if isError(r) {
				return nil, r
			}
			if isTruthy(r) == want {
				out.set(k, h.keyObj(k), h.Pairs[k])
			}
		}
		return out, nil
	}
	switch name {
	case "length", "size":
		return &Integer{Value: int64(len(h.Order))}, true
	case "count":
		if blk == nil && len(args) == 0 {
			return &Integer{Value: int64(len(h.Order))}, true
		}
		return i.enumMethod(i.hashPairs(h), name, args, blk, h)
	case "empty?":
		return boolObj(len(h.Order) == 0), true
	case "keys":
		out := make([]Object, len(h.Order))
		for k, key := range h.Order {
			out[k] = h.keyObj(key)
		}
		return arr(out), true
	case "values":
		out := make([]Object, len(h.Order))
		for k, key := range h.Order {
			out[k] = h.Pairs[key]
		}
		return arr(out), true
	case "values_at", "fetch_values":
		out := make([]Object, len(args))
		for k, key := range args {
			v, ok := lookup(key)
			if !ok {
				if name == "fetch_values" {
					return newErrorClass("KeyError", "key not found: %s", i.inspectValue(key)), true
				}
				v = &Nil{}
			}
			out[k] = v
		}
		return arr(out), true
	case "key?", "has_key?", "include?", "member?":
		_, ok := lookup(args[0])
		return boolObj(ok), true
	case "value?", "has_value?":
		for _, k := range h.Order {
			if i.equal(h.Pairs[k], args[0]) {
				return boolObj(true), true
			}
		}
		return boolObj(false), true
	case "key":
		for _, k := range h.Order {
			if i.equal(h.Pairs[k], args[0]) {
				return h.keyObj(k), true
			}
		}
		return &Nil{}, true
	case "[]":
		return i.indexGet(h, args[0]), true
	case "[]=", "store":
		return i.indexSet(h, args[0], args[1]), true
	case "fetch":
		if v, ok := lookup(args[0]); ok {
			return v, true
		}
		if blk != nil {
			return i.yield(blk, args[0]), true
		}
		if len(args) > 1 {
			return args[1], true
		}
		return newErrorClass("KeyError", "key not found: %s", i.inspectValue(args[0])), true
	case "dig":
		cur := i.indexGet(h, args[0])
		if len(args) == 1 || isError(cur) {
			return cur, true
		}
		if _, isNil := cur.(*Nil); isNil {
			return cur, true
		}
		return i.callMember(cur, "dig", args[1:], nil, callMode{isCall: true}), true
	case "delete":
		if e := mut(); e != nil {
			return e, true
		}
		k := hashKey(args[0])
		v, ok := h.Pairs[k]
		if !ok {
			if blk != nil {
				return i.yield(blk, args[0]), true
			}
			return &Nil{}, true
		}
		h.remove(k)
		return v, true
	case "merge", "merge!", "update":
		if name != "merge" {
			if e := mut(); e != nil {
				return e, true
			}
		}
		target := h
		if name == "merge" {
			target = hashMerge(h, &Hash{})
		}
		for _, o := range args {
			oh, ok := o.(*Hash)
			if !ok {
				return newError("no implicit conversion of %s into Hash", i.className(o)), true
			}
			for _, k := range oh.Order {
				nv := oh.Pairs[k]
				if old, exists := target.Pairs[k]; exists && blk != nil {
					nv = i.yield(blk, oh.keyObj(k), old, nv)
					if isError(nv) {
						return nv, true
					}
				}
				target.set(k, oh.keyObj(k), nv)
			}
		}
		return target, true
	case "each", "each_pair":
		if blk == nil {
			return arr(i.hashPairs(h)), true
		}
		for _, p := range i.hashPairs(h) {
			if r := i.yield(blk, p); isError(r) {
				return r, true
			}
		}
		return h, true
	case "each_key", "each_value":
		for _, k := range append([]string{}, h.Order...) {
			v := h.keyObj(k)
			if name == "each_value" {
				v = h.Pairs[k]
			}
			if r := i.yield(blk, v); isError(r) {
				return r, true
			}
		}
		return h, true
	case "select", "filter", "reject", "find_all":
		if blk == nil {
			return h, true
		}
		if name == "find_all" {
			return i.enumMethod(i.hashPairs(h), name, args, blk, h)
		}
		out, err := keep(name != "reject")
		if err != nil {
			return err, true
		}
		return out, true
	case "select!", "filter!", "keep_if", "reject!", "delete_if":
		if e := mut(); e != nil {
			return e, true
		}
		out, err := keep(name == "select!" || name == "filter!" || name == "keep_if")
		if err != nil {
			return err, true
		}
		oh := out.(*Hash)
		changed := len(oh.Order) != len(h.Order)
		h.Pairs, h.Order, h.Keys = oh.Pairs, oh.Order, oh.Keys
		if !changed && (name == "select!" || name == "filter!" || name == "reject!") {
			return &Nil{}, true
		}
		return h, true
	case "partition":
		var a, b []Object
		for _, p := range i.hashPairs(h) {
			r := i.yield(blk, p)
			if isError(r) {
				return r, true
			}
			if isTruthy(r) {
				a = append(a, p)
			} else {
				b = append(b, p)
			}
		}
		return arr([]Object{arr(a), arr(b)}), true
	case "transform_values", "transform_values!":
		out := i.newHashLike(h)
		for _, k := range h.Order {
			r := i.yield(blk, h.Pairs[k])
			if isError(r) {
				return r, true
			}
			out.set(k, h.keyObj(k), r)
		}
		if name == "transform_values!" {
			h.Pairs = out.Pairs
			return h, true
		}
		return out, true
	case "transform_keys", "transform_keys!":
		out := i.newHashLike(h)
		for _, k := range h.Order {
			var nk Object
			if len(args) > 0 {
				if m, ok := args[0].(*Hash); ok {
					if v, found := m.Pairs[k]; found {
						nk = v
					}
				}
			}
			if nk == nil && blk != nil {
				nk = i.yield(blk, h.keyObj(k))
				if isError(nk) {
					return nk, true
				}
			}
			if nk == nil {
				nk = h.keyObj(k)
			}
			out.set(hashKey(nk), nk, h.Pairs[k])
		}
		if name == "transform_keys!" {
			h.Pairs, h.Order, h.Keys = out.Pairs, out.Order, out.Keys
			return h, true
		}
		return out, true
	case "map", "collect", "flat_map", "filter_map", "sum", "min_by", "max_by", "sort_by", "group_by", "each_with_index",
		"each_with_object", "reduce", "inject", "find", "detect", "any?", "all?", "none?", "zip", "take", "drop",
		"take_while", "drop_while", "first", "sort", "min", "max", "uniq", "tally", "each_slice", "each_cons", "to_a", "entries",
		"find_index", "cycle", "reverse_each", "one?", "minmax":
		return i.enumMethod(i.hashPairs(h), name, args, blk, h)
	case "to_h":
		if blk == nil {
			return h, true
		}
		return i.enumMethod(i.hashPairs(h), name, args, blk, h)
	case "invert":
		out := &Hash{Pairs: map[string]Object{}}
		for _, k := range h.Order {
			out.set(hashKey(h.Pairs[k]), h.Pairs[k], h.keyObj(k))
		}
		return out, true
	case "slice", "except":
		out := i.newHashLike(h)
		want := map[string]bool{}
		for _, a := range args {
			want[hashKey(a)] = true
		}
		for _, k := range h.Order {
			if want[k] == (name == "slice") {
				out.set(k, h.keyObj(k), h.Pairs[k])
			}
		}
		return out, true
	case "compact":
		out := i.newHashLike(h)
		for _, k := range h.Order {
			if _, isNil := h.Pairs[k].(*Nil); !isNil {
				out.set(k, h.keyObj(k), h.Pairs[k])
			}
		}
		return out, true
	case "clear":
		if e := mut(); e != nil {
			return e, true
		}
		h.Pairs, h.Order, h.Keys = map[string]Object{}, nil, nil
		return h, true
	case "default":
		if h.Default == nil {
			return &Nil{}, true
		}
		return h.Default, true
	case "default=":
		h.Default = args[0]
		return args[0], true
	case "assoc":
		if v, ok := lookup(args[0]); ok {
			return arr([]Object{args[0], v}), true
		}
		return &Nil{}, true
	case "to_s", "inspect":
		return &String{Value: i.inspectValue(h)}, true
	case "==", "!=":
		eq := i.equal(h, args[0])
		return boolObj(eq == (name == "==")), true
	case "compare_by_identity":
		return h, true
	}
	return nil, false
}

func (i *Interpreter) rangeMethod(r *Range, name string, args []Object, blk Object) (Object, bool) {
	switch name {
	case "begin", "first", "min":
		if name == "first" && len(args) > 0 {
			break
		}
		if name == "min" && (blk != nil || len(args) > 0) {
			break
		}
		if r.IsStr {
			return &String{Value: r.StrStart}, true
		}
		if r.BeginLess {
			return &Nil{}, true
		}
		if name == "min" && !r.Endless && (r.Start > r.End || (r.Exclusive && r.Start == r.End)) {
			return &Nil{}, true
		}
		return &Integer{Value: r.Start}, true
	case "end", "last", "max":
		if (name == "last" && len(args) > 0) || (name == "max" && (blk != nil || len(args) > 0)) {
			break
		}
		if r.IsStr {
			return &String{Value: r.StrEnd}, true
		}
		if r.Endless {
			return &Nil{}, true
		}
		if name != "end" && r.Exclusive {
			if r.Start >= r.End {
				return &Nil{}, true
			}
			return &Integer{Value: r.End - 1}, true
		}
		if name == "max" && r.Start > r.End {
			return &Nil{}, true
		}
		return &Integer{Value: r.End}, true
	case "exclude_end?":
		return boolObj(r.Exclusive), true
	case "size", "count", "length":
		if name == "count" && (blk != nil || len(args) > 0) {
			break
		}
		if r.Endless {
			return &Float{Value: posInf()}, true
		}
		if r.IsStr {
			return &Integer{Value: int64(len(r.ToArray().Elements))}, true
		}
		if r.Start > r.End {
			return &Integer{Value: 0}, true
		}
		sz := r.End - r.Start + 1
		if r.Exclusive {
			sz--
		}
		return &Integer{Value: sz}, true
	case "include?", "member?", "cover?", "===":
		if len(args) == 1 {
			return i.caseEqual(r, args[0]), true
		}
	case "to_s", "inspect":
		return &String{Value: r.Inspect()}, true
	case "step", "%":
		if len(args) < 1 {
			break
		}
		if r.Endless && blk == nil {
			return newError("cannot convert endless range to an array"), true
		}
		step, _ := toFloat(args[0])
		if step <= 0 {
			return newError("step can't be 0 or negative"), true
		}
		_, intStep := args[0].(*Integer)
		var out []Object
		end := float64(r.End)
		for x := float64(r.Start); r.Endless || x < end || (!r.Exclusive && x == end); x += step {
			var it Object
			if intStep {
				it = &Integer{Value: int64(x)}
			} else {
				it = &Float{Value: x}
			}
			if blk != nil {
				if res := i.yield(blk, it); isError(res) {
					return res, true
				}
			} else {
				out = append(out, it)
			}
		}
		if blk != nil {
			return r, true
		}
		return &Array{Elements: out}, true
	case "each":
		if r.Endless && blk != nil && !r.IsStr {
			for x := r.Start; ; x++ {
				if res := i.yield(blk, &Integer{Value: x}); isError(res) {
					return res, true
				}
			}
		}
	case "to_a", "to_ary", "entries", "force":
		if r.Endless || r.BeginLess {
			return newError("cannot convert endless range to an array"), true
		}
		return r.ToArray(), true
	case "dup", "clone", "freeze":
		return r, true
	case "sum":
		if blk == nil && len(args) == 0 && !r.IsStr && !r.Endless && !r.BeginLess {
			if r.Start > r.End {
				return &Integer{Value: 0}, true
			}
			end := r.End
			if r.Exclusive {
				end--
			}
			if end < r.Start {
				return &Integer{Value: 0}, true
			}
			return &Integer{Value: (r.Start + end) * (end - r.Start + 1) / 2}, true
		}
	case "take", "first_n":
		if r.Endless && len(args) > 0 {
			n, _ := toInt64Strict(args[0])
			out := make([]Object, 0, n)
			for k := int64(0); k < n; k++ {
				out = append(out, &Integer{Value: r.Start + k})
			}
			return &Array{Elements: out}, true
		}
	case "lazy":
		return r, true
	case "==", "eql?":
		return boolObj(i.equal(r, args[0])), true
	case "hash":
		return &Integer{Value: int64(hashString(r.Inspect()))}, true
	}
	if name == "first" && r.Endless && len(args) > 0 {
		return i.rangeMethod(r, "take", args, blk)
	}
	if r.Endless || r.BeginLess {
		switch name {
		case "find", "detect", "take_while", "each_with_index", "find_index":
			if r.Endless && blk != nil {
				for x := r.Start; ; x++ {
					var res Object
					if name == "each_with_index" {
						res = i.yield(blk, &Integer{Value: x}, &Integer{Value: x - r.Start})
					} else {
						res = i.yield(blk, &Integer{Value: x})
					}
					if isError(res) {
						return res, true
					}
					if name != "each_with_index" && isTruthy(res) == (name != "take_while") {
						if name == "find_index" {
							return &Integer{Value: x - r.Start}, true
						}
						return &Integer{Value: x}, true
					}
					if name == "take_while" && !isTruthy(res) {
						return &Nil{}, true
					}
				}
			}
		}
		return nil, false
	}
	return i.enumMethod(r.ToArray().Elements, name, args, blk, r)
}

func (i *Interpreter) procMethod(recv Object, name string, args []Object, blk Object) (Object, bool) {
	switch name {
	case "call", "()", "yield", "===", "[]":
		return i.callFunction(recv, args, blk), true
	case "to_proc":
		return recv, true
	case "lambda?":
		f, ok := recv.(*Function)
		return boolObj(ok && f.IsLambda), true
	case "arity":
		f, ok := recv.(*Function)
		if !ok {
			return &Integer{Value: -1}, true
		}
		req, opt := 0, 0
		for k := range f.Parameters {
			switch {
			case k < len(f.Keyword) && f.Keyword[k]:
			case k < len(f.Defaults) && f.Defaults[k] != nil:
				opt++
			default:
				req++
			}
		}
		if f.Variadic || (opt > 0 && f.IsLambda) {
			return &Integer{Value: int64(-req - 1)}, true
		}
		return &Integer{Value: int64(req)}, true
	case "parameters":
		f, ok := recv.(*Function)
		if !ok {
			return &Array{}, true
		}
		var out []Object
		for k, p := range f.Parameters {
			kind := "req"
			if k < len(f.Keyword) && f.Keyword[k] {
				kind = "key"
			} else if k < len(f.Defaults) && f.Defaults[k] != nil {
				kind = "opt"
			}
			out = append(out, &Array{Elements: []Object{&Symbol{Name: kind}, &Symbol{Name: p.Value}}})
		}
		return &Array{Elements: out}, true
	case "curry":
		f, ok := recv.(*Function)
		arity := int64(0)
		if ok {
			for k := range f.Parameters {
				if !(k < len(f.Keyword) && f.Keyword[k]) && !(k < len(f.Defaults) && f.Defaults[k] != nil) {
					arity++
				}
			}
		}
		if len(args) > 0 {
			arity, _ = toInt64Strict(args[0])
		}
		return i.curry(recv, int(arity), nil), true
	case ">>", "<<":
		g := args[0]
		first, second := recv, g
		if name == "<<" {
			first, second = g, recv
		}
		return &Builtin{Fn: func(a ...Object) Object {
			r := i.callFunction(first, a, nil)
			if isError(r) {
				return r
			}
			return i.callFunction(second, []Object{r}, nil)
		}}, true
	case "name":
		if b, ok := recv.(*BoundMethod); ok {
			return &Symbol{Name: b.Name}, true
		}
		if f, ok := recv.(*Function); ok && f.Name != "" {
			return &Symbol{Name: f.Name}, true
		}
	case "receiver":
		if b, ok := recv.(*BoundMethod); ok && b.Recv != nil {
			return b.Recv, true
		}
	case "unbind", "bind":
		return recv, true
	}
	return nil, false
}

func (i *Interpreter) curry(fn Object, arity int, got []Object) Object {
	return &Builtin{Fn: func(a ...Object) Object {
		all := append(append([]Object{}, got...), a...)
		if len(all) >= arity {
			return i.callFunction(fn, all, nil)
		}
		return i.curry(fn, arity, all)
	}}
}

func (i *Interpreter) structClassMethod(s *Struct, name string, args []Object, blk Object) (Object, bool) {
	switch name {
	case "new", "[]":
		return i.constructStruct(s, args), true
	case "members":
		order := s.fieldOrder()
		out := make([]Object, len(order))
		for k, f := range order {
			out[k] = &Symbol{Name: f}
		}
		return &Array{Elements: out}, true
	case "name", "to_s":
		return &String{Value: s.Name}, true
	}
	return nil, false
}

func (i *Interpreter) callStructMethod(r *StructInstance, name string, args []Object, blk Object) (Object, bool) {
	order := r.Struct.fieldOrder()
	switch name {
	case "to_h":
		h := &Hash{Pairs: map[string]Object{}}
		for _, f := range order {
			h.set(f, &Symbol{Name: f}, r.Fields[f])
		}
		return h, true
	case "to_a", "values", "deconstruct":
		out := make([]Object, len(order))
		for k, f := range order {
			out[k] = r.Fields[f]
		}
		return &Array{Elements: out}, true
	case "members":
		out := make([]Object, len(order))
		for k, f := range order {
			out[k] = &Symbol{Name: f}
		}
		return &Array{Elements: out}, true
	case "with", "merge":
		c := &StructInstance{Struct: r.Struct, TypeArgs: r.TypeArgs, Fields: map[string]Object{}}
		for k, v := range r.Fields {
			c.Fields[k] = v
		}
		for _, a := range args {
			if h, ok := a.(*Hash); ok {
				for _, k := range h.Order {
					if _, exists := r.Struct.Fields[k]; !exists {
						return newError("undefined field '%s' in struct '%s'", k, r.Struct.Name), true
					}
					c.Fields[k] = h.Pairs[k]
				}
			}
		}
		return c, true
	case "[]":
		if len(args) == 1 {
			return i.indexGet(r, args[0]), true
		}
	case "[]=":
		if len(args) == 2 {
			return i.indexSet(r, args[0], args[1]), true
		}
	case "each":
		for _, f := range order {
			if res := i.yield(blk, r.Fields[f]); isError(res) {
				return res, true
			}
		}
		return r, true
	case "each_pair":
		for _, f := range order {
			if res := i.yield(blk, &Symbol{Name: f}, r.Fields[f]); isError(res) {
				return res, true
			}
		}
		return r, true
	case "to_s", "inspect":
		return &String{Value: i.inspectValue(r)}, true
	case "==":
		return boolObj(i.equal(r, args[0])), true
	case "dig":
		cur := i.indexGet(r, args[0])
		if len(args) == 1 || isError(cur) {
			return cur, true
		}
		if _, isNil := cur.(*Nil); isNil {
			return cur, true
		}
		return i.callMember(cur, "dig", args[1:], nil, callMode{isCall: true}), true
	}
	if strings.HasSuffix(name, "=") && len(args) == 1 {
		f := strings.TrimSuffix(name, "=")
		if _, ok := r.Struct.Fields[f]; ok {
			return i.indexSet(r, &String{Value: f}, args[0]), true
		}
	}
	return nil, false
}
