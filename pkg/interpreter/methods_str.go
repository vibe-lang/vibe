package interpreter

import (
	"math/rand"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var stringNames = strings.Fields("length size upcase downcase capitalize swapcase strip lstrip rstrip chomp chop chars bytes lines split reverse include? start_with? end_with? index rindex sub gsub scan match match? tr delete squeeze count center ljust rjust each_char each_line each_with_index to_i to_f to_s to_sym to_str empty? replace insert prepend concat freeze * + % <=> ord chr hex succ next slice [] []= format unpack encoding times between? casecmp casecmp? eql? upto")

func (i *Interpreter) strArg(args []Object, idx int) (string, bool) {
	if idx >= len(args) {
		return "", false
	}
	switch v := args[idx].(type) {
	case *String:
		return v.Value, true
	case *Symbol:
		return v.Name, true
	}
	return "", false
}

func strObjs(ss []string) []Object {
	out := make([]Object, len(ss))
	for k, s := range ss {
		out[k] = &String{Value: s}
	}
	return out
}

func (i *Interpreter) toRegex(o Object) (*RegexObject, bool) {
	switch v := o.(type) {
	case *RegexObject:
		return v, true
	case *String:
		if re, err := newRegex(regexpQuote(v.Value), ""); err == nil {
			return re, true
		}
	}
	return nil, false
}

func regexpQuote(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(`\.+*?()|[]{}^$`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// expandReplacement handles \1, \0 and \k<name> in sub/gsub replacement strings.
func expandReplacement(repl string, re *RegexObject, m []string) string {
	if !strings.Contains(repl, "\\") {
		return repl
	}
	var b strings.Builder
	for k := 0; k < len(repl); k++ {
		c := repl[k]
		if c == '\\' && k+1 < len(repl) {
			n := repl[k+1]
			if n >= '0' && n <= '9' {
				idx := int(n - '0')
				if idx < len(m) {
					b.WriteString(m[idx])
				}
				k++
				continue
			}
			if n == 'k' && k+2 < len(repl) && repl[k+2] == '<' {
				if end := strings.IndexByte(repl[k+3:], '>'); end >= 0 {
					name := repl[k+3 : k+3+end]
					for gi, gn := range re.Re.SubexpNames() {
						if gn == name && gi < len(m) {
							b.WriteString(m[gi])
						}
					}
					k += 3 + end
					continue
				}
			}
			if n == '\\' {
				b.WriteByte('\\')
				k++
				continue
			}
		}
		b.WriteByte(c)
	}
	return b.String()
}

func (i *Interpreter) subst(s string, args []Object, blk Object, global bool) (string, Object) {
	if len(args) == 0 {
		return "", newError("wrong number of arguments (given 0, expected 1..2)")
	}
	re, ok := i.toRegex(args[0])
	if !ok {
		return "", newError("wrong argument type %s (expected Regexp)", i.className(args[0]))
	}
	var repl *String
	var rh *Hash
	if len(args) > 1 {
		switch r := args[1].(type) {
		case *String:
			repl = r
		case *Hash:
			rh = r
		default:
			return "", newError("no implicit conversion of %s into String", i.className(args[1]))
		}
	} else if blk == nil {
		return "", newError("wrong number of arguments (given 1, expected 2)")
	}
	matches := re.Re.FindAllStringSubmatchIndex(s, -1)
	if !global && len(matches) > 1 {
		matches = matches[:1]
	}
	var b strings.Builder
	last := 0
	for _, loc := range matches {
		b.WriteString(s[last:loc[0]])
		m := make([]string, len(loc)/2)
		for g := range m {
			if loc[2*g] >= 0 {
				m[g] = s[loc[2*g]:loc[2*g+1]]
			}
		}
		switch {
		case repl != nil:
			b.WriteString(expandReplacement(repl.Value, re, m))
		case rh != nil:
			if v, ok := rh.Pairs[m[0]]; ok {
				b.WriteString(i.display(v))
			}
		default:
			i.setMatchGlobals(re, m[0])
			r := i.yield(blk, &String{Value: m[0]})
			if isError(r) {
				return "", r
			}
			b.WriteString(i.display(r))
		}
		last = loc[1]
	}
	b.WriteString(s[last:])
	return b.String(), nil
}

func expandTrSet(set string) ([]rune, bool) {
	neg := false
	rs := []rune(set)
	if len(rs) > 1 && rs[0] == '^' {
		neg = true
		rs = rs[1:]
	}
	var out []rune
	for k := 0; k < len(rs); k++ {
		if k+2 < len(rs) && rs[k+1] == '-' {
			for c := rs[k]; c <= rs[k+2]; c++ {
				out = append(out, c)
			}
			k += 2
			continue
		}
		if rs[k] == '\\' && k+1 < len(rs) {
			k++
		}
		out = append(out, rs[k])
	}
	return out, neg
}

func trString(s, from, to string, del bool) string {
	fs, neg := expandTrSet(from)
	ts, _ := expandTrSet(to)
	var b strings.Builder
	for _, r := range s {
		idx := -1
		for k, c := range fs {
			if c == r {
				idx = k
				break
			}
		}
		hit := idx >= 0
		if neg {
			hit = !hit
		}
		if !hit {
			b.WriteRune(r)
			continue
		}
		if del {
			continue
		}
		if len(ts) == 0 {
			continue
		}
		if neg || idx >= len(ts) {
			b.WriteRune(ts[len(ts)-1])
		} else {
			b.WriteRune(ts[idx])
		}
	}
	return b.String()
}

func inSets(r rune, sets []string) bool {
	for _, set := range sets {
		cs, neg := expandTrSet(set)
		found := false
		for _, c := range cs {
			if c == r {
				found = true
				break
			}
		}
		if found == neg {
			return false
		}
	}
	return true
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + strings.ToLower(s[n:])
}

func swapcase(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsUpper(r) {
			return unicode.ToLower(r)
		}
		return unicode.ToUpper(r)
	}, s)
}

func reverseString(s string) string {
	rs := []rune(s)
	for a, b := 0, len(rs)-1; a < b; a, b = a+1, b-1 {
		rs[a], rs[b] = rs[b], rs[a]
	}
	return string(rs)
}

func padString(s string, width int, pad string, mode string) string {
	n := utf8.RuneCountInString(s)
	if width <= n || pad == "" {
		return s
	}
	total := width - n
	mk := func(k int) string {
		var b strings.Builder
		pr := []rune(pad)
		for c := 0; c < k; c++ {
			b.WriteRune(pr[c%len(pr)])
		}
		return b.String()
	}
	switch mode {
	case "l":
		return s + mk(total)
	case "r":
		return mk(total) + s
	}
	left := total / 2
	return mk(left) + s + mk(total-left)
}

func parseIntPrefix(s string, base int) int64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "_", "")
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	if base == 16 {
		s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	} else if base == 2 {
		s = strings.TrimPrefix(strings.TrimPrefix(s, "0b"), "0B")
	}
	end := 0
	for end < len(s) {
		d := digitVal(s[end])
		if d < 0 || d >= base {
			break
		}
		end++
	}
	if end == 0 {
		return 0
	}
	n, _ := strconv.ParseInt(s[:end], base, 64)
	if neg {
		return -n
	}
	return n
}

func digitVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	}
	return -1
}

func parseFloatPrefix(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, "_", ""))
	end := 0
	seenDot, seenExp, seenDigit := false, false, false
	for end < len(s) {
		c := s[end]
		switch {
		case c >= '0' && c <= '9':
			seenDigit = true
		case (c == '+' || c == '-') && (end == 0 || s[end-1] == 'e' || s[end-1] == 'E'):
		case c == '.' && !seenDot && !seenExp && end+1 < len(s) && s[end+1] >= '0' && s[end+1] <= '9':
			seenDot = true
		case (c == 'e' || c == 'E') && seenDigit && !seenExp && end+1 < len(s):
			seenExp = true
		default:
			goto done
		}
		end++
	}
done:
	for end > 0 {
		if f, err := strconv.ParseFloat(s[:end], 64); err == nil {
			return f
		}
		end--
	}
	return 0
}

func (i *Interpreter) stringMethod(s *String, name string, args []Object, blk Object) (Object, bool) {
	v := s.Value
	mutate := func(nv string) Object {
		if s.Frozen {
			return i.frozenErr(s)
		}
		s.Value = nv
		return s
	}
	bang := func(nv string) Object {
		if s.Frozen {
			return i.frozenErr(s)
		}
		if nv == v {
			return &Nil{}
		}
		s.Value = nv
		return s
	}
	str := func(x string) (Object, bool) { return &String{Value: x}, true }
	switch name {
	case "length", "size":
		return &Integer{Value: int64(utf8.RuneCountInString(v))}, true
	case "bytesize":
		return &Integer{Value: int64(len(v))}, true
	case "to_s", "to_str":
		return s, true
	case "inspect", "dump":
		return &String{Value: strconv.Quote(v)}, true
	case "to_sym", "intern":
		return &Symbol{Name: v}, true
	case "to_i":
		base := 10
		if len(args) > 0 {
			b, _ := toInt64Strict(args[0])
			base = int(b)
		}
		return &Integer{Value: parseIntPrefix(v, base)}, true
	case "hex":
		return &Integer{Value: parseIntPrefix(v, 16)}, true
	case "oct":
		return &Integer{Value: parseIntPrefix(v, 8)}, true
	case "to_f":
		return &Float{Value: parseFloatPrefix(v)}, true
	case "to_r", "to_c":
		return &Float{Value: parseFloatPrefix(v)}, true
	case "upcase":
		return str(strings.ToUpper(v))
	case "downcase":
		return str(strings.ToLower(v))
	case "capitalize":
		return str(capitalize(v))
	case "swapcase":
		return str(swapcase(v))
	case "upcase!":
		return bang(strings.ToUpper(v)), true
	case "downcase!":
		return bang(strings.ToLower(v)), true
	case "capitalize!":
		return bang(capitalize(v)), true
	case "swapcase!":
		return bang(swapcase(v)), true
	case "strip":
		return str(strings.TrimSpace(v))
	case "lstrip":
		return str(strings.TrimLeftFunc(v, unicode.IsSpace))
	case "rstrip":
		return str(strings.TrimRightFunc(v, unicode.IsSpace))
	case "strip!":
		return bang(strings.TrimSpace(v)), true
	case "lstrip!":
		return bang(strings.TrimLeftFunc(v, unicode.IsSpace)), true
	case "rstrip!":
		return bang(strings.TrimRightFunc(v, unicode.IsSpace)), true
	case "chomp", "chomp!":
		var nv string
		if sfx, ok := i.strArg(args, 0); ok {
			nv = strings.TrimSuffix(v, sfx)
		} else if strings.HasSuffix(v, "\r\n") {
			nv = v[:len(v)-2]
		} else {
			nv = strings.TrimSuffix(strings.TrimSuffix(v, "\n"), "\r")
		}
		if name == "chomp!" {
			return bang(nv), true
		}
		return str(nv)
	case "chop", "chop!":
		nv := v
		if v != "" {
			_, n := utf8.DecodeLastRuneInString(v)
			nv = v[:len(v)-n]
		}
		if name == "chop!" {
			return bang(nv), true
		}
		return str(nv)
	case "chr":
		if v == "" {
			return str("")
		}
		r, _ := utf8.DecodeRuneInString(v)
		return str(string(r))
	case "chars":
		out := strObjs(nil)
		for _, r := range v {
			out = append(out, &String{Value: string(r)})
		}
		return &Array{Elements: out}, true
	case "bytes":
		var out []Object
		for k := 0; k < len(v); k++ {
			out = append(out, &Integer{Value: int64(v[k])})
		}
		return &Array{Elements: out}, true
	case "codepoints":
		var out []Object
		for _, r := range v {
			out = append(out, &Integer{Value: int64(r)})
		}
		return &Array{Elements: out}, true
	case "lines":
		var out []string
		rest := v
		for rest != "" {
			idx := strings.IndexByte(rest, '\n')
			if idx < 0 {
				out = append(out, rest)
				break
			}
			out = append(out, rest[:idx+1])
			rest = rest[idx+1:]
		}
		return &Array{Elements: strObjs(out)}, true
	case "each_char", "each_line", "each_byte":
		var items []Object
		switch name {
		case "each_char":
			items = func() []Object { r, _ := i.stringMethod(s, "chars", nil, nil); return r.(*Array).Elements }()
		case "each_line":
			items = func() []Object { r, _ := i.stringMethod(s, "lines", nil, nil); return r.(*Array).Elements }()
		default:
			items = func() []Object { r, _ := i.stringMethod(s, "bytes", nil, nil); return r.(*Array).Elements }()
		}
		if blk == nil {
			return &Array{Elements: items}, true
		}
		for _, it := range items {
			if r := i.yield(blk, it); isError(r) {
				return r, true
			}
		}
		return s, true
	case "split":
		var parts []string
		limit := -1
		if len(args) > 1 {
			l, _ := toInt64Strict(args[1])
			limit = int(l)
		}
		if len(args) == 0 {
			parts = strings.Fields(v)
		} else if re, ok := args[0].(*RegexObject); ok {
			parts = re.Re.Split(v, limit)
			if len(parts) > 0 && parts[0] == "" && re.Pattern == "" {
				parts = parts[1:]
			}
		} else if sep, ok := i.strArg(args, 0); ok {
			switch {
			case sep == " ":
				parts = strings.Fields(v)
			case sep == "":
				for _, r := range v {
					parts = append(parts, string(r))
				}
			case limit > 0:
				parts = strings.SplitN(v, sep, limit)
			default:
				parts = strings.Split(v, sep)
			}
		} else {
			return newError("wrong argument type %s (expected Regexp)", i.className(args[0])), true
		}
		if limit < 0 {
			for len(parts) > 0 && parts[len(parts)-1] == "" {
				parts = parts[:len(parts)-1]
			}
		}
		return &Array{Elements: strObjs(parts)}, true
	case "reverse":
		return str(reverseString(v))
	case "reverse!":
		return mutate(reverseString(v)), true
	case "include?", "member?":
		sub, ok := i.strArg(args, 0)
		if !ok {
			if re, isRe := args[0].(*RegexObject); isRe {
				return boolObj(re.Re.MatchString(v)), true
			}
			return newError("no implicit conversion of %s into String", i.className(args[0])), true
		}
		return boolObj(strings.Contains(v, sub)), true
	case "start_with?":
		for _, a := range args {
			if re, ok := a.(*RegexObject); ok {
				if loc := re.Re.FindStringIndex(v); loc != nil && loc[0] == 0 {
					return boolObj(true), true
				}
			} else if p, ok := a.(*String); ok && strings.HasPrefix(v, p.Value) {
				return boolObj(true), true
			}
		}
		return boolObj(false), true
	case "end_with?":
		for _, a := range args {
			if p, ok := a.(*String); ok && strings.HasSuffix(v, p.Value) {
				return boolObj(true), true
			}
		}
		return boolObj(false), true
	case "delete_prefix":
		p, _ := i.strArg(args, 0)
		return str(strings.TrimPrefix(v, p))
	case "delete_suffix":
		p, _ := i.strArg(args, 0)
		return str(strings.TrimSuffix(v, p))
	case "empty?":
		return boolObj(v == ""), true
	case "index", "rindex":
		start := 0
		if len(args) > 1 {
			st, _ := toInt64Strict(args[1])
			if st < 0 {
				st += int64(utf8.RuneCountInString(v))
			}
			start = int(st)
		}
		rs := []rune(v)
		if start < 0 || start > len(rs) {
			return &Nil{}, true
		}
		off := len(string(rs[:start]))
		var byteIdx int
		if re, ok := args[0].(*RegexObject); ok {
			loc := re.Re.FindStringIndex(v[off:])
			if loc == nil {
				return &Nil{}, true
			}
			byteIdx = off + loc[0]
		} else {
			sub, _ := i.strArg(args, 0)
			if name == "index" {
				idx := strings.Index(v[off:], sub)
				if idx < 0 {
					return &Nil{}, true
				}
				byteIdx = off + idx
			} else {
				idx := strings.LastIndex(v, sub)
				if idx < 0 {
					return &Nil{}, true
				}
				byteIdx = idx
			}
		}
		return &Integer{Value: int64(utf8.RuneCountInString(v[:byteIdx]))}, true
	case "sub", "gsub", "sub!", "gsub!":
		nv, err := i.subst(v, args, blk, strings.HasPrefix(name, "g"))
		if err != nil {
			return err, true
		}
		if strings.HasSuffix(name, "!") {
			return bang(nv), true
		}
		return str(nv)
	case "scan":
		re, ok := i.toRegex(args[0])
		if !ok {
			return newError("wrong argument type (expected Regexp)"), true
		}
		var out []Object
		for _, m := range re.Re.FindAllStringSubmatch(v, -1) {
			var item Object
			if len(m) == 1 {
				item = &String{Value: m[0]}
			} else {
				item = &Array{Elements: strObjs(m[1:])}
			}
			if blk != nil {
				if r := i.yield(blk, item); isError(r) {
					return r, true
				}
			} else {
				out = append(out, item)
			}
		}
		if blk != nil {
			return s, true
		}
		return &Array{Elements: out}, true
	case "match":
		re, ok := i.toRegex(args[0])
		if !ok {
			return newError("wrong argument type (expected Regexp)"), true
		}
		return i.regexMatch(re, v), true
	case "match?":
		re, ok := i.toRegex(args[0])
		if !ok {
			return newError("wrong argument type (expected Regexp)"), true
		}
		return boolObj(re.Re.MatchString(v)), true
	case "=~":
		return i.matchOp(s, args[0]), true
	case "tr", "tr!", "tr_s":
		from, _ := i.strArg(args, 0)
		to, _ := i.strArg(args, 1)
		nv := trString(v, from, to, false)
		if name == "tr!" {
			return bang(nv), true
		}
		return str(nv)
	case "delete", "delete!":
		var sets []string
		for k := range args {
			x, _ := i.strArg(args, k)
			sets = append(sets, x)
		}
		nv := strings.Map(func(r rune) rune {
			if inSets(r, sets) {
				return -1
			}
			return r
		}, v)
		if name == "delete!" {
			return bang(nv), true
		}
		return str(nv)
	case "squeeze", "squeeze!":
		var sets []string
		for k := range args {
			x, _ := i.strArg(args, k)
			sets = append(sets, x)
		}
		var b strings.Builder
		var prev rune = -1
		for _, r := range v {
			if r == prev && (len(sets) == 0 || inSets(r, sets)) {
				continue
			}
			b.WriteRune(r)
			prev = r
		}
		if name == "squeeze!" {
			return bang(b.String()), true
		}
		return str(b.String())
	case "count":
		var sets []string
		for k := range args {
			x, _ := i.strArg(args, k)
			sets = append(sets, x)
		}
		n := 0
		for _, r := range v {
			if inSets(r, sets) {
				n++
			}
		}
		return &Integer{Value: int64(n)}, true
	case "center", "ljust", "rjust":
		w, ok := argInt(args, 0, 0)
		if !ok {
			return newError("no implicit conversion into Integer"), true
		}
		pad := " "
		if p, ok := i.strArg(args, 1); ok {
			pad = p
		}
		mode := map[string]string{"center": "c", "ljust": "l", "rjust": "r"}[name]
		return str(padString(v, int(w), pad, mode))
	case "ord":
		if v == "" {
			return newError("empty string"), true
		}
		r, _ := utf8.DecodeRuneInString(v)
		return &Integer{Value: int64(r)}, true
	case "sum":
		var t int64
		for k := 0; k < len(v); k++ {
			t += int64(v[k])
		}
		return &Integer{Value: t}, true
	case "succ", "next":
		return str(strSucc(v))
	case "replace":
		if len(args) == 2 {
			o, _ := i.strArg(args, 0)
			n, _ := i.strArg(args, 1)
			return str(strings.ReplaceAll(v, o, n))
		}
		o, _ := i.strArg(args, 0)
		return mutate(o), true
	case "clear":
		return mutate(""), true
	case "concat", "<<":
		nv := v
		for _, a := range args {
			nv += i.display(a)
		}
		return mutate(nv), true
	case "prepend":
		nv := v
		for k := len(args) - 1; k >= 0; k-- {
			nv = i.display(args[k]) + nv
		}
		return mutate(nv), true
	case "insert":
		idx, _ := toInt64Strict(args[0])
		ins, _ := i.strArg(args, 1)
		rs := []rune(v)
		if idx < 0 {
			idx += int64(len(rs)) + 1
		}
		if idx < 0 || int(idx) > len(rs) {
			return newError("index %d out of string", idx), true
		}
		return mutate(string(rs[:idx]) + ins + string(rs[idx:])), true
	case "slice", "[]":
		return i.indexGet(s, firstOrIndex(args)), true
	case "slice!":
		r := i.indexGet(s, firstOrIndex(args))
		if rs, ok := r.(*String); ok {
			if idx := strings.Index(v, rs.Value); idx >= 0 {
				mutate(v[:idx] + v[idx+len(rs.Value):])
			}
		}
		return r, true
	case "[]=":
		if len(args) >= 2 {
			return i.indexSet(s, args[0], args[len(args)-1]), true
		}
	case "freeze":
		s.Frozen = true
		return s, true
	case "dup", "+@", "clone":
		return &String{Value: v}, true
	case "-@":
		s.Frozen = true
		return s, true
	case "<=>":
		if o, ok := args[0].(*String); ok {
			return &Integer{Value: int64(strings.Compare(v, o.Value))}, true
		}
		return &Nil{}, true
	case "casecmp":
		o, _ := i.strArg(args, 0)
		return &Integer{Value: int64(strings.Compare(strings.ToLower(v), strings.ToLower(o)))}, true
	case "casecmp?":
		o, _ := i.strArg(args, 0)
		return boolObj(strings.EqualFold(v, o)), true
	case "between?":
		return numClamp(i, s, name, args)
	case "upto":
		if o, ok := i.strArg(args, 0); ok {
			items := strObjs(strRange(v, o, false))
			if blk == nil {
				return &Array{Elements: items}, true
			}
			for _, it := range items {
				if r := i.yield(blk, it); isError(r) {
					return r, true
				}
			}
			return s, true
		}
	case "*", "+", "%", "<", ">", "<=", ">=":
		if len(args) == 1 {
			if r, ok := i.stringInfix(name, s, args[0]); ok {
				return r, true
			}
			return newError("no implicit conversion of %s into String", i.className(args[0])), true
		}
	case "unicode_normalize", "encode", "force_encoding", "scrub", "b":
		return s, true
	case "encoding":
		return str("UTF-8")
	case "valid_encoding?", "ascii_only?":
		return boolObj(utf8.ValidString(v)), true
	case "format":
		return i.sprintf(v, args), true
	case "partition", "rpartition":
		sep, ok := i.strArg(args, 0)
		if !ok {
			if re, isRe := args[0].(*RegexObject); isRe {
				loc := re.Re.FindStringIndex(v)
				if loc == nil {
					return &Array{Elements: strObjs([]string{v, "", ""})}, true
				}
				return &Array{Elements: strObjs([]string{v[:loc[0]], v[loc[0]:loc[1]], v[loc[1]:]})}, true
			}
		}
		idx := strings.Index(v, sep)
		if name == "rpartition" {
			idx = strings.LastIndex(v, sep)
		}
		if idx < 0 {
			if name == "rpartition" {
				return &Array{Elements: strObjs([]string{"", "", v})}, true
			}
			return &Array{Elements: strObjs([]string{v, "", ""})}, true
		}
		return &Array{Elements: strObjs([]string{v[:idx], sep, v[idx+len(sep):]})}, true
	case "shuffle":
		rs := []rune(v)
		rand.Shuffle(len(rs), func(a, b int) { rs[a], rs[b] = rs[b], rs[a] })
		return str(string(rs))
	case "each_with_index", "each_char_with_index":
		r, _ := i.stringMethod(s, "chars", nil, nil)
		return i.enumMethod(r.(*Array).Elements, "each_with_index", args, blk, s)
	}
	return nil, false
}

func firstOrIndex(args []Object) Object {
	if len(args) == 2 {
		a, ok1 := toInt64Strict(args[0])
		b, ok2 := toInt64Strict(args[1])
		if ok1 && ok2 {
			return &Range{Start: a, End: a + b, Exclusive: true}
		}
	}
	if len(args) == 0 {
		return &Nil{}
	}
	return args[0]
}
