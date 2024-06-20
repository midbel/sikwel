package format

type RewriteRule uint8

const (
	RewriteStdExpr = 1 << iota
	RewriteStdOp
	RewriteMissCteAlias
	RewriteMissViewAlias
	RewriteWithCte
	RewriteWithSubqueries
	RewriteJoinWithLiteral
)

func (r RewriteRule) All() bool {
	return false
}

func (r RewriteRule) UseStdExpr() bool {
	return r&RewriteStdExpr != 0
}

func (r RewriteRule) UseStdOp() bool {
	return r&RewriteStdOp != 0
}

func (r RewriteRule) SetMissingCteAlias() bool {
	return r&RewriteMissCteAlias != 0
}

func (r RewriteRule) SetMissingViewAlias() bool {
	return r&RewriteMissViewAlias != 0
}

func (r RewriteRule) ReplaceCteWithSubquery() bool {
	return r&RewriteWithSubqueries != 0
}

func (r RewriteRule) ReplaceSubqueryWithCte() bool {
	return r&RewriteWithCte != 0
}

func (r RewriteRule) JoinWithLiteralAsSubquery() bool {
	return r&RewriteJoinWithLiteral != 0
}

func (r RewriteRule) KeepAsIs() bool {
	return r == 0
}

type UpperMode uint8

const (
	UpperNone UpperMode = 1 << iota
	UpperKw
	UpperFn
	UpperId
	UpperType
)

func (u UpperMode) All() bool {
	return u.Identifier() && u.Function() && u.Keyword() && u.Type()
}

func (u UpperMode) Identifier() bool {
	return (u & UpperId) != 0
}

func (u UpperMode) Function() bool {
	return (u & UpperFn) != 0
}

func (u UpperMode) Keyword() bool {
	return (u & UpperKw) != 0
}

func (u UpperMode) Type() bool {
	return (u & UpperType) != 0
}
