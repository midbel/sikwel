package token

import (
	"fmt"
	"strings"
)

type Position struct {
	Line   int
	Column int
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

type Token struct {
	Symbol

	Offset int
	Position
}

func (t Token) IsJoin() bool {
	kw := strings.ToUpper(t.Literal)
	return t.Type == Keyword && strings.HasSuffix(kw, "JOIN")
}

func (t Token) IsValue() bool {
	return t.Type == Ident || t.Type == Literal || t.Type == Number
}

func (t Token) AsSymbol() Symbol {
	sym := Symbol{
		Type: t.Type,
	}
	if t.Type == Keyword {
		sym.Literal = t.Literal
	}
	return sym
}

func (t Token) Length() int {
	if t.Literal != "" {
		return len(t.Literal)
	}
	if t.Type == Concat {
		return 2
	}
	return 1
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	case EOF:
		return "<eof>"
	case EOL:
		return "<eol>"
	case Dot:
		return "<dot>"
	case Comma:
		return "<comma>"
	case Lparen:
		return "<lparen>"
	case Rparen:
		return "<rparen>"
	case Plus:
		return "<plus>"
	case Minus:
		return "<minus>"
	case Slash:
		return "<slash>"
	case Star:
		return "<star>"
	case Mod:
		return "<modulo>"
	case BitAnd:
		return "<bit-and>"
	case BitOr:
		return "<bit-or>"
	case BitXor:
		return "<bit-xor>"
	case Lshift:
		return "<left-shift>"
	case Rshift:
		return "<right-shift>"
	case AddAssign:
		return "<add-assign>"
	case MinAssign:
		return "<min-assign>"
	case MulAssign:
		return "<mul-assign>"
	case DivAssign:
		return "<div-assign>"
	case ModAssign:
		return "<mod-assign>"
	case Concat:
		return "<concat>"
	case Arrow:
		return "<arrow>"
	case Eq:
		return "<equal>"
	case Ne:
		return "<not-equal>"
	case Lt:
		return "<lesser-than>"
	case Le:
		return "<lesser-eq>"
	case Gt:
		return "<greater-than>"
	case Ge:
		return "<greater-eq>"
	case Macro:
		prefix = "macro"
	case Ident:
		prefix = "identifier"
	case Literal:
		prefix = "literal"
	case Keyword:
		prefix = "keyword"
	case Number:
		prefix = "number"
	case Comment:
		prefix = "comment"
	case Invalid:
		return "<invalid>"
	default:
		prefix = "unknown"
	}
	return fmt.Sprintf("%s(%s)", prefix, t.Literal)
}

type Symbol struct {
	Type    rune
	Literal string
}

func SymbolFor(kind rune, literal string) Symbol {
	return Symbol{
		Type:    kind,
		Literal: literal,
	}
}
