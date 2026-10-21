package interpreter

import (
	"sort"
	"strings"
)

var arrayNames = strings.Fields("length size push pop shift unshift first last each map select reject reduce inject sum min max sort sort_by reverse join include? index uniq flatten compact zip take drop count any? all? none? find group_by partition each_with_index each_slice each_cons tally flat_map filter_map empty? concat delete insert slice sample shuffle rotate product combination permutation transpose dig fetch values_at to_a to_h + - * & | <=> << []")

func (i *Interpreter) cmpFail(a, b Object) Object {
	return newError("comparison of %s with %s failed", i.className(a), i.describeCmp(b))
}

func (i *Interpreter) describeCmp(o Object) string {
	switch o.(type) {
	case *Integer, *Float, *Nil, *Boolean:
		return i.inspectValue(o)
	}
	return i.className(o)
}

// sortObjects sorts items using <=> or a comparison block.
func (i *Interpreter) sortObjects(items []Object, blk Object) ([]Object, Object) {
	out := append([]Object{}, items...)
	var fail Object
	sort.SliceStable(out, func(a, b int) bool {
		if fail != nil {
			return false
		}
		if blk != nil {
			r := i.yield(blk, out[a], out[b])
			if isError(r) {
				fail = r
				return false
			}
			n, ok := toInt64Strict(r)
			if !ok {
				fail = i.cmpFail(out[a], out[b])
				return false
			}
			return n < 0
		}
		c, ok := i.compare(out[a], out[b])
		if !ok {
			fail = i.cmpFail(out[a], out[b])
			return false
		}
		return c < 0
	})
	return out, fail
}

// sortByKeys sorts items by the keys computed from the block.
func (i *Interpreter) sortByKeys(items []Object, blk Object) ([]Object, Object) {
	keys := make([]Object, len(items))
	for k, it := range items {
		r := i.yield(blk, it)
		if isError(r) {
			return nil, r
		}
		keys[k] = r
	}
	idx := make([]int, len(items))
	for k := range idx {
		idx[k] = k
	}
	var fail Object
	sort.SliceStable(idx, func(a, b int) bool {
		if fail != nil {
			return false
		}
		c, ok := i.compare(keys[idx[a]], keys[idx[b]])
		if !ok {
			fail = i.cmpFail(keys[idx[a]], keys[idx[b]])
			return false
		}
		return c < 0
	})
	if fail != nil {
		return nil, fail
	}
	out := make([]Object, len(items))
	for k, ix := range idx {
		out[k] = items[ix]
	}
	return out, nil
}

func (i *Interpreter) flatten(items []Object, depth int) []Object {
	var out []Object
	for _, it := range items {
		if a, ok := it.(*Array); ok && depth != 0 {
			out = append(out, i.flatten(a.Elements, depth-1)...)
		} else {
			out = append(out, it)
		}
	}
	return out
}

func (i *Interpreter) uniqObjects(items []Object, blk Object) ([]Object, Object) {
	seen := map[string]bool{}
	var out []Object
	for _, it := range items {
		k := it
		if blk != nil {
			r := i.yield(blk, it)
			if isError(r) {
				return nil, r
			}
			k = r
		}
		ks := i.keyString(k)
		if !seen[ks] {
			seen[ks] = true
			out = append(out, it)
		}
	}
	return out, nil
}

// predicate evaluates the "matcher" of any?/all?/count/etc: a pattern argument, a block, or truthiness.
func (i *Interpreter) predicate(args []Object, blk Object) func(Object) (bool, Object) {
	return func(it Object) (bool, Object) {
		switch {
		case len(args) > 0:
			return isTruthy(i.caseEqual(args[0], it)), nil
		case blk != nil:
			r := i.yield(blk, it)
			if isError(r) {
				return false, r
			}
			return isTruthy(r), nil
		}
		return isTruthy(it), nil
	}
}

func minMax(i *Interpreter, items []Object, want int, blk Object) (Object, Object) {
	if len(items) == 0 {
		return &Nil{}, nil
	}
	best := items[0]
	for _, it := range items[1:] {
		var c int
		if blk != nil {
			r := i.yield(blk, it, best)
			if isError(r) {
				return nil, r
			}
			n, _ := toInt64Strict(r)
			c = int(n)
		} else {
			cc, ok := i.compare(it, best)
			if !ok {
				return nil, i.cmpFail(it, best)
			}
			c = cc
		}
		if c*want > 0 {
			best = it
		}
	}
	return best, nil
}

func (i *Interpreter) toPairs(items []Object) *Hash {
	h := &Hash{Pairs: map[string]Object{}}
	for _, it := range items {
		p, ok := it.(*Array)
		if !ok || len(p.Elements) != 2 {
			return nil
		}
		h.set(hashKey(p.Elements[0]), p.Elements[0], p.Elements[1])
	}
	return h
}

// enumMethod implements Enumerable over a snapshot of items. recv is returned by each-style methods.
func (i *Interpreter) enumMethod(items []Object, name string, args []Object, blk Object, recv Object) (Object, bool) {
	arr := func(o []Object) Object { return &Array{Elements: o} }
	needBlk := func() Object {
		if blk == nil {
			return newError("no block given (%s)", name)
		}
		return nil
	}
	switch name {
	case "each", "each_entry":
		if blk == nil {
			return arr(items), true
		}
		for _, it := range items {
			if r := i.yield(blk, it); isError(r) {
				return r, true
			}
		}
		return recv, true
	case "reverse_each":
		if blk == nil {
			rev := make([]Object, len(items))
			for k, it := range items {
				rev[len(items)-1-k] = it
			}
			return arr(rev), true
		}
		for k := len(items) - 1; k >= 0; k-- {
			if r := i.yield(blk, items[k]); isError(r) {
				return r, true
			}
		}
		return recv, true
	case "each_with_index", "each_with_index_map":
		if blk == nil {
			out := make([]Object, len(items))
			for k, it := range items {
				out[k] = arr([]Object{it, &Integer{Value: int64(k)}})
			}
			return arr(out), true
		}
		for k, it := range items {
			if r := i.yield(blk, it, &Integer{Value: int64(k)}); isError(r) {
				return r, true
			}
		}
		return recv, true
	case "each_with_object":
		memo := args[0]
		if e := needBlk(); e != nil {
			return e, true
		}
		for _, it := range items {
			if r := i.yield(blk, it, memo); isError(r) {
				return r, true
			}
		}
		return memo, true
	case "map", "collect", "flat_map", "collect_concat", "filter_map":
		if blk == nil {
			return arr(items), true
		}
		var out []Object
		for _, it := range items {
			r := i.yield(blk, it)
			if isError(r) {
				return r, true
			}
			switch name {
			case "flat_map", "collect_concat":
				if a, ok := r.(*Array); ok {
					out = append(out, a.Elements...)
					continue
				}
			case "filter_map":
				if !isTruthy(r) {
					continue
				}
			}
			out = append(out, r)
		}
		return arr(out), true
	case "with_index", "map_with_index", "each_with_index_map!":
		off := int64(0)
		if len(args) > 0 {
			off, _ = toInt64Strict(args[0])
		}
		if blk == nil {
			return arr(items), true
		}
		var out []Object
		for k, it := range items {
			r := i.yield(blk, it, &Integer{Value: int64(k) + off})
			if isError(r) {
				return r, true
			}
			out = append(out, r)
		}
		return arr(out), true
	case "select", "filter", "find_all", "reject":
		if blk == nil {
			return arr(items), true
		}
		var out []Object
		for _, it := range items {
			r := i.yield(blk, it)
			if isError(r) {
				return r, true
			}
			if isTruthy(r) == (name != "reject") {
				out = append(out, it)
			}
		}
		return arr(out), true
	case "partition":
		if e := needBlk(); e != nil {
			return e, true
		}
		var a, b []Object
		for _, it := range items {
			r := i.yield(blk, it)
			if isError(r) {
				return r, true
			}
			if isTruthy(r) {
				a = append(a, it)
			} else {
				b = append(b, it)
			}
		}
		return arr([]Object{arr(a), arr(b)}), true
	case "find", "detect":
		if e := needBlk(); e != nil {
			return e, true
		}
		for _, it := range items {
			r := i.yield(blk, it)
			if isError(r) {
				return r, true
			}
			if isTruthy(r) {
				return it, true
			}
		}
		return &Nil{}, true
	case "find_index", "index":
		for k, it := range items {
			var hit bool
			if len(args) > 0 {
				hit = i.equal(it, args[0])
			} else if blk != nil {
				r := i.yield(blk, it)
				if isError(r) {
					return r, true
				}
				hit = isTruthy(r)
			}
			if hit {
				return &Integer{Value: int64(k)}, true
			}
		}
		return &Nil{}, true
	case "group_by", "tally_by":
		if e := needBlk(); e != nil {
			return e, true
		}
		h := &Hash{Pairs: map[string]Object{}}
		for _, it := range items {
			r := i.yield(blk, it)
			if isError(r) {
				return r, true
			}
			k := hashKey(r)
			if name == "tally_by" {
				cur, _ := h.Pairs[k].(*Integer)
				n := int64(1)
				if cur != nil {
					n = cur.Value + 1
				}
				h.set(k, r, &Integer{Value: n})
				continue
			}
			g, ok := h.Pairs[k].(*Array)
			if !ok {
				g = &Array{}
				h.set(k, r, g)
			}
			g.Elements = append(g.Elements, it)
		}
		return h, true
	case "tally":
		h := &Hash{Pairs: map[string]Object{}}
		for _, it := range items {
			k := hashKey(it)
			n := int64(1)
			if cur, ok := h.Pairs[k].(*Integer); ok {
				n = cur.Value + 1
			}
			h.set(k, it, &Integer{Value: n})
		}
		return h, true
	case "chunk_while", "slice_when":
		if e := needBlk(); e != nil {
			return e, true
		}
		var out []Object
		var cur []Object
		for k, it := range items {
			if k > 0 {
				r := i.yield(blk, items[k-1], it)
				if isError(r) {
					return r, true
				}
				split := !isTruthy(r)
				if name == "slice_when" {
					split = isTruthy(r)
				}
				if split {
					out = append(out, arr(cur))
					cur = nil
				}
			}
			cur = append(cur, it)
		}
		if len(cur) > 0 {
			out = append(out, arr(cur))
		}
		return arr(out), true
	case "each_slice", "each_cons":
		n, ok := argInt(args, 0, 0)
		if !ok || n <= 0 {
			return newError("invalid slice size"), true
		}
		var out []Object
		if name == "each_slice" {
			for k := 0; k < len(items); k += int(n) {
				end := k + int(n)
				if end > len(items) {
					end = len(items)
				}
				out = append(out, arr(append([]Object{}, items[k:end]...)))
			}
		} else {
			for k := 0; k+int(n) <= len(items); k++ {
				out = append(out, arr(append([]Object{}, items[k:k+int(n)]...)))
			}
		}
		if blk == nil {
			return arr(out), true
		}
		for _, g := range out {
			if r := i.yield(blk, g); isError(r) {
				return r, true
			}
		}
		return recv, true
	case "reduce", "inject":
		var acc Object
		rest := items
		var op string
		switch len(args) {
		case 2:
			acc = args[0]
			op = symOrString(args[1])
		case 1:
			if _, isSym := args[0].(*Symbol); isSym && blk == nil {
				op = symOrString(args[0])
			} else {
				acc = args[0]
			}
		}
		if acc == nil {
			if len(rest) == 0 {
				return &Nil{}, true
			}
			acc, rest = rest[0], rest[1:]
		}
		for _, it := range rest {
			if op != "" {
				acc = i.callMember(acc, op, []Object{it}, nil, callMode{isCall: true})
			} else if blk != nil {
				acc = i.yield(blk, acc, it)
			} else {
				return newError("no block given (reduce)"), true
			}
			if isError(acc) {
				return acc, true
			}
		}
		return acc, true
	case "sum":
		var acc Object = &Integer{Value: 0}
		if len(args) > 0 {
			acc = args[0]
		}
		for _, it := range items {
			if blk != nil {
				it = i.yield(blk, it)
				if isError(it) {
					return it, true
				}
			}
			acc = i.infix("+", acc, it)
			if isError(acc) {
				return acc, true
			}
		}
		return acc, true
	case "count":
		if len(args) == 0 && blk == nil {
			return &Integer{Value: int64(len(items))}, true
		}
		n := 0
		if len(args) > 0 {
			for _, it := range items {
				if i.equal(it, args[0]) {
					n++
				}
			}
			return &Integer{Value: int64(n)}, true
		}
		pred := i.predicate(nil, blk)
		for _, it := range items {
			ok, err := pred(it)
			if err != nil {
				return err, true
			}
			if ok {
				n++
			}
		}
		return &Integer{Value: int64(n)}, true
	case "min", "max":
		want := 1
		if name == "min" {
			want = -1
		}
		if len(args) > 0 {
			n, _ := toInt64Strict(args[0])
			sorted, err := i.sortObjects(items, blk)
			if err != nil {
				return err, true
			}
			if want == 1 {
				for a, b := 0, len(sorted)-1; a < b; a, b = a+1, b-1 {
					sorted[a], sorted[b] = sorted[b], sorted[a]
				}
			}
			if int(n) < len(sorted) {
				sorted = sorted[:n]
			}
			return arr(sorted), true
		}
		r, err := minMax(i, items, want, blk)
		if err != nil {
			return err, true
		}
		return r, true
	case "min_by", "max_by":
		if e := needBlk(); e != nil {
			return e, true
		}
		if len(items) == 0 {
			return &Nil{}, true
		}
		want := 1
		if name == "min_by" {
			want = -1
		}
		var best, bestKey Object
		for k, it := range items {
			key := i.yield(blk, it)
			if isError(key) {
				return key, true
			}
			if k == 0 {
				best, bestKey = it, key
				continue
			}
			c, ok := i.compare(key, bestKey)
			if !ok {
				return i.cmpFail(key, bestKey), true
			}
			if c*want > 0 {
				best, bestKey = it, key
			}
		}
		return best, true
	case "minmax":
		lo, err := minMax(i, items, -1, blk)
		if err != nil {
			return err, true
		}
		hi, err := minMax(i, items, 1, blk)
		if err != nil {
			return err, true
		}
		return arr([]Object{lo, hi}), true
	case "sort":
		out, err := i.sortObjects(items, blk)
		if err != nil {
			return err, true
		}
		return arr(out), true
	case "sort_by":
		if e := needBlk(); e != nil {
			return e, true
		}
		out, err := i.sortByKeys(items, blk)
		if err != nil {
			return err, true
		}
		return arr(out), true
	case "take":
		n, ok := argInt(args, 0, 0)
		if !ok || n < 0 {
			return newError("attempt to take negative size"), true
		}
		if int(n) > len(items) {
			n = int64(len(items))
		}
		return arr(append([]Object{}, items[:n]...)), true
	case "drop":
		n, ok := argInt(args, 0, 0)
		if !ok || n < 0 {
			return newError("attempt to drop negative size"), true
		}
		if int(n) > len(items) {
			n = int64(len(items))
		}
		return arr(append([]Object{}, items[n:]...)), true
	case "take_while", "drop_while":
		if e := needBlk(); e != nil {
			return e, true
		}
		k := 0
		for ; k < len(items); k++ {
			r := i.yield(blk, items[k])
			if isError(r) {
				return r, true
			}
			if !isTruthy(r) {
				break
			}
		}
		if name == "take_while" {
			return arr(append([]Object{}, items[:k]...)), true
		}
		return arr(append([]Object{}, items[k:]...)), true
	case "first":
		if len(args) == 0 {
			if len(items) == 0 {
				return &Nil{}, true
			}
			return items[0], true
		}
		return i.enumMethod(items, "take", args, nil, recv)
	case "any?", "all?", "none?", "one?":
		pred := i.predicate(args, blk)
		n := 0
		for _, it := range items {
			ok, err := pred(it)
			if err != nil {
				return err, true
			}
			if ok {
				n++
				if name == "any?" {
					return boolObj(true), true
				}
				if name == "none?" {
					return boolObj(false), true
				}
			} else if name == "all?" {
				return boolObj(false), true
			}
		}
		switch name {
		case "any?":
			return boolObj(false), true
		case "one?":
			return boolObj(n == 1), true
		}
		return boolObj(true), true
	case "include?", "member?":
		for _, it := range items {
			if i.equal(it, args[0]) {
				return boolObj(true), true
			}
		}
		return boolObj(false), true
	case "to_a", "entries", "force", "lazy", "to_enum", "each_entry_array":
		return arr(append([]Object{}, items...)), true
	case "to_h":
		out := make([]Object, len(items))
		for k, it := range items {
			if blk != nil {
				r := i.yield(blk, it)
				if isError(r) {
					return r, true
				}
				it = r
			}
			out[k] = it
		}
		h := i.toPairs(out)
		if h == nil {
			return newError("wrong element type (expected array of [key, value] pairs)"), true
		}
		return h, true
	case "uniq":
		out, err := i.uniqObjects(items, blk)
		if err != nil {
			return err, true
		}
		return arr(out), true
	case "zip":
		others := make([][]Object, len(args))
		for k, a := range args {
			o, ok := i.iterItems(a)
			if !ok {
				return newError("wrong argument type %s (must respond to :each)", i.className(a)), true
			}
			others[k] = o
		}
		out := make([]Object, len(items))
		for k, it := range items {
			row := []Object{it}
			for _, o := range others {
				if k < len(o) {
					row = append(row, o[k])
				} else {
					row = append(row, &Nil{})
				}
			}
			out[k] = arr(row)
		}
		if blk != nil {
			for _, row := range out {
				if r := i.yield(blk, row); isError(r) {
					return r, true
				}
			}
			return &Nil{}, true
		}
		return arr(out), true
	case "cycle":
		if blk == nil || len(items) == 0 {
			return &Nil{}, true
		}
		if len(args) == 0 {
			for {
				for _, it := range items {
					if r := i.yield(blk, it); isError(r) {
						return r, true
					}
				}
			}
		}
		n, _ := toInt64Strict(args[0])
		for c := int64(0); c < n; c++ {
			for _, it := range items {
				if r := i.yield(blk, it); isError(r) {
					return r, true
				}
			}
		}
		return &Nil{}, true
	case "join":
		sep := ""
		if len(args) > 0 {
			if s, ok := args[0].(*String); ok {
				sep = s.Value
			}
		}
		parts := make([]string, len(items))
		for k, it := range items {
			if a, ok := it.(*Array); ok {
				r, _ := i.enumMethod(a.Elements, "join", args, nil, a)
				parts[k] = r.(*String).Value
			} else {
				parts[k] = i.display(it)
			}
		}
		return &String{Value: strings.Join(parts, sep)}, true
	case "step":
		n, ok := argInt(args, 0, 1)
		if !ok || n <= 0 {
			return newError("step can't be 0"), true
		}
		var out []Object
		for k := 0; k < len(items); k += int(n) {
			out = append(out, items[k])
		}
		if blk == nil {
			return arr(out), true
		}
		for _, it := range out {
			if r := i.yield(blk, it); isError(r) {
				return r, true
			}
		}
		return recv, true
	}
	return nil, false
}
