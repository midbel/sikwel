package token

const (
	EOL rune = -(iota + 1)
	EOF
	Dot
	Comment
	Ident
	Literal
	Keyword
	Macro
	Number
	Comma
	Lparen
	Rparen
	Plus
	Minus
	Slash
	Star
	Mod
	BitAnd
	BitOr
	BitXor
	Lshift
	Rshift
	Eq
	Ne
	Lt
	Le
	Gt
	Ge
	AddAssign
	MinAssign
	MulAssign
	DivAssign
	ModAssign
	Concat
	Arrow
	Invalid
)
