package ast

import (
	"bytes"
	"strings"

	"github.com/vibe-lang/vibe/pkg/lexer"
)

// HashEntry is one element of a hash literal in source order. A splat entry
// ({**other}) carries only a Value.
type HashEntry struct {
	Key   Expression
	Value Expression
	Splat bool
}

// RescueClause is one `rescue Type, Type => name` clause of a begin block.
type RescueClause struct {
	Token lexer.Token
	Types []Expression // Exception classes to match; empty matches any error
	Var   *Identifier  // Optional binding for the caught value
	Body  *BlockStatement
}

func (rc *RescueClause) TokenLiteral() string { return rc.Token.Literal }
func (rc *RescueClause) String() string {
	var out bytes.Buffer
	out.WriteString("rescue")
	types := []string{}
	for _, t := range rc.Types {
		types = append(types, t.String())
	}
	if len(types) > 0 {
		out.WriteString(" " + strings.Join(types, ", "))
	}
	if rc.Var != nil {
		out.WriteString(" => " + rc.Var.String())
	}
	out.WriteString(" " + rc.Body.String())
	return out.String()
}

// SymbolLiteral is :name or :"quoted name".
type SymbolLiteral struct {
	Token lexer.Token
	Value string
}

func (sl *SymbolLiteral) expressionNode()      {}
func (sl *SymbolLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *SymbolLiteral) String() string       { return ":" + sl.Value }

// RegexLiteral is /pattern/flags.
type RegexLiteral struct {
	Token   lexer.Token
	Pattern string
	Flags   string
}

func (rl *RegexLiteral) expressionNode()      {}
func (rl *RegexLiteral) TokenLiteral() string { return rl.Token.Literal }
func (rl *RegexLiteral) String() string       { return "/" + rl.Pattern + "/" + rl.Flags }

// SplatExpression is *value in call arguments, array literals and destructuring targets.
type SplatExpression struct {
	Token lexer.Token
	Value Expression // nil for a bare `*` target
}

func (se *SplatExpression) expressionNode()      {}
func (se *SplatExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SplatExpression) String() string {
	if se.Value == nil {
		return "*"
	}
	return "*" + se.Value.String()
}

// DoubleSplatExpression is **value in call arguments.
type DoubleSplatExpression struct {
	Token lexer.Token
	Value Expression
}

func (ds *DoubleSplatExpression) expressionNode()      {}
func (ds *DoubleSplatExpression) TokenLiteral() string { return ds.Token.Literal }
func (ds *DoubleSplatExpression) String() string       { return "**" + ds.Value.String() }

// BlockPass is &value in call arguments: the value is passed as the call's block.
type BlockPass struct {
	Token lexer.Token
	Value Expression
}

func (bp *BlockPass) expressionNode()      {}
func (bp *BlockPass) TokenLiteral() string { return bp.Token.Literal }
func (bp *BlockPass) String() string       { return "&" + bp.Value.String() }

// KeywordArgument is `name: value` in a call's argument list.
type KeywordArgument struct {
	Token lexer.Token
	Name  string
	Value Expression
}

func (ka *KeywordArgument) expressionNode()      {}
func (ka *KeywordArgument) TokenLiteral() string { return ka.Token.Literal }
func (ka *KeywordArgument) String() string       { return ka.Name + ": " + ka.Value.String() }

// YieldExpression calls the block passed to the current function.
type YieldExpression struct {
	Token     lexer.Token
	Arguments []Expression
}

func (ye *YieldExpression) expressionNode()      {}
func (ye *YieldExpression) TokenLiteral() string { return ye.Token.Literal }
func (ye *YieldExpression) String() string {
	args := []string{}
	for _, a := range ye.Arguments {
		args = append(args, a.String())
	}
	return "yield(" + strings.Join(args, ", ") + ")"
}

// LogicalAssignment is `target ||= value`, `target &&= value` or `target ??= value`.
// The value is only evaluated, and the target only written, when the operator requires it.
type LogicalAssignment struct {
	Token    lexer.Token
	Target   Expression // Identifier, IndexExpression or DotExpression
	Operator string     // "||", "&&" or "??"
	Value    Expression
}

func (la *LogicalAssignment) expressionNode()      {}
func (la *LogicalAssignment) TokenLiteral() string { return la.Token.Literal }
func (la *LogicalAssignment) String() string {
	return la.Target.String() + " " + la.Operator + "= " + la.Value.String()
}

// PostfixLoop is `statement while cond` / `statement until cond`. With a begin/end
// body the statement runs once before the condition is first checked.
type PostfixLoop struct {
	Token     lexer.Token
	Statement Statement
	Condition Expression
	Until     bool
	DoWhile   bool
}

func (pl *PostfixLoop) statementNode()       {}
func (pl *PostfixLoop) TokenLiteral() string { return pl.Token.Literal }
func (pl *PostfixLoop) String() string {
	kw := "while"
	if pl.Until {
		kw = "until"
	}
	return pl.Statement.String() + " " + kw + " " + pl.Condition.String()
}

// AliasStatement is `alias new_name old_name`.
type AliasStatement struct {
	Token   lexer.Token
	NewName string
	OldName string
}

func (as *AliasStatement) statementNode()       {}
func (as *AliasStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AliasStatement) String() string       { return "alias " + as.NewName + " " + as.OldName }

// RetryStatement re-runs the begin block that is handling the current rescue.
type RetryStatement struct {
	Token lexer.Token
}

func (rs *RetryStatement) statementNode()       {}
func (rs *RetryStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *RetryStatement) String() string       { return "retry" }

// TypeAliasStatement is `type Name = int | string`.
type TypeAliasStatement struct {
	Token lexer.Token
	Name  *Identifier
	Type  Expression
}

func (ta *TypeAliasStatement) statementNode()       {}
func (ta *TypeAliasStatement) TokenLiteral() string { return ta.Token.Literal }
func (ta *TypeAliasStatement) String() string {
	return "type " + ta.Name.String() + " = " + ta.Type.String()
}

// MethodSignature is a method declaration inside an interface.
type MethodSignature struct {
	Name       string
	Parameters []*Identifier
	ParamTypes []Expression
	ReturnType Expression
}

// InterfaceStatement is `interface Name ... end`, a set of required method signatures.
type InterfaceStatement struct {
	Token   lexer.Token
	Name    *Identifier
	Methods []*MethodSignature
}

func (is *InterfaceStatement) statementNode()       {}
func (is *InterfaceStatement) TokenLiteral() string { return is.Token.Literal }
func (is *InterfaceStatement) String() string {
	names := []string{}
	for _, m := range is.Methods {
		names = append(names, m.Name)
	}
	return "interface " + is.Name.String() + " { " + strings.Join(names, "; ") + " }"
}

// MatchPattern is `value in pattern` (a boolean test) or `value => pattern` (raises on mismatch).
type MatchPattern struct {
	Token   lexer.Token
	Value   Expression
	Pattern Expression
	Strict  bool
}

func (mp *MatchPattern) expressionNode()      {}
func (mp *MatchPattern) TokenLiteral() string { return mp.Token.Literal }
func (mp *MatchPattern) String() string {
	op := " in "
	if mp.Strict {
		op = " => "
	}
	return mp.Value.String() + op + mp.Pattern.String()
}

// InClause is `in pattern [if guard] then body` inside a case expression.
type InClause struct {
	Token   lexer.Token
	Pattern Expression
	Guard   Expression
	Unless  bool
	Body    *BlockStatement
}

// ValuePattern matches with ===: literals, ranges, classes and constants.
type ValuePattern struct {
	Token lexer.Token
	Value Expression
}

func (vp *ValuePattern) expressionNode()      {}
func (vp *ValuePattern) TokenLiteral() string { return vp.Token.Literal }
func (vp *ValuePattern) String() string       { return vp.Value.String() }

// BindPattern matches anything and binds it to a name.
type BindPattern struct {
	Token lexer.Token
	Name  string
}

func (bp *BindPattern) expressionNode()      {}
func (bp *BindPattern) TokenLiteral() string { return bp.Token.Literal }
func (bp *BindPattern) String() string       { return bp.Name }

// PinPattern matches against the current value of an expression: ^name or ^(expr).
type PinPattern struct {
	Token lexer.Token
	Value Expression
}

func (pp *PinPattern) expressionNode()      {}
func (pp *PinPattern) TokenLiteral() string { return pp.Token.Literal }
func (pp *PinPattern) String() string       { return "^" + pp.Value.String() }

// AltPattern matches when any alternative matches: a | b | c.
type AltPattern struct {
	Token lexer.Token
	Alts  []Expression
}

func (ap *AltPattern) expressionNode()      {}
func (ap *AltPattern) TokenLiteral() string { return ap.Token.Literal }
func (ap *AltPattern) String() string {
	parts := []string{}
	for _, a := range ap.Alts {
		parts = append(parts, a.String())
	}
	return strings.Join(parts, " | ")
}

// CapturePattern matches the inner pattern and also binds the value: pattern => name.
type CapturePattern struct {
	Token   lexer.Token
	Pattern Expression
	Name    string
}

func (cp *CapturePattern) expressionNode()      {}
func (cp *CapturePattern) TokenLiteral() string { return cp.Token.Literal }
func (cp *CapturePattern) String() string       { return cp.Pattern.String() + " => " + cp.Name }

// ArrayPattern matches an array element by element: [a, 1, *rest, b].
type ArrayPattern struct {
	Token    lexer.Token
	Class    Expression // optional Const(...) / Const[...] constraint
	Elements []Expression
	RestIdx  int    // index in Elements of the splat, or -1
	RestName string // binding for the splat, "" if anonymous
}

func (ap *ArrayPattern) expressionNode()      {}
func (ap *ArrayPattern) TokenLiteral() string { return ap.Token.Literal }
func (ap *ArrayPattern) String() string {
	parts := []string{}
	for idx, e := range ap.Elements {
		if idx == ap.RestIdx {
			parts = append(parts, "*"+ap.RestName)
		} else {
			parts = append(parts, e.String())
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// HashPattern matches hash/struct/object keys: {name:, age: Integer => n, **rest}.
type HashPattern struct {
	Token    lexer.Token
	Class    Expression // optional Const(...) constraint
	Keys     []string
	Patterns []Expression // nil means "bind the key name"
	HasRest  bool
	RestName string // binding for **rest, "" if anonymous
	Exact    bool   // **nil: no other keys allowed
}

func (hp *HashPattern) expressionNode()      {}
func (hp *HashPattern) TokenLiteral() string { return hp.Token.Literal }
func (hp *HashPattern) String() string {
	parts := []string{}
	for idx, k := range hp.Keys {
		if hp.Patterns[idx] == nil {
			parts = append(parts, k+":")
		} else {
			parts = append(parts, k+": "+hp.Patterns[idx].String())
		}
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
