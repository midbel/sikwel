package format

type RewriteRule uint16

const (
	RewriteStdExpr = 1 << iota
	RewriteStdOp
	RewriteMissCteAlias
	RewriteMissViewAlias
	RewriteWithCte
	RewriteWithSubqueries
	RewriteJoinSubquery
	RewriteJoinPredicate
	RewriteGroupByGroup
	RewriteGroupByAggr

	RewriteAll = RewriteStdExpr |
		RewriteStdOp |
		RewriteMissCteAlias |
		RewriteMissViewAlias |
		RewriteWithCte |
		RewriteWithSubqueries |
		RewriteJoinSubquery |
		RewriteJoinPredicate |
		RewriteGroupByGroup |
		RewriteGroupByAggr
)

func (r RewriteRule) All() bool {
	return r == RewriteAll
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

func (r RewriteRule) JoinAsSubquery() bool {
	return r&RewriteJoinSubquery != 0
}

func (r RewriteRule) JoinPredicate() bool {
	return r&RewriteJoinPredicate != 0
}

func (r RewriteRule) SetRewriteGroupBy() bool {
	return r.SetRewriteGroupByGroup() || r.SetRewriteGroupByAggr()
}

func (r RewriteRule) SetRewriteGroupByGroup() bool {
	return r&RewriteGroupByGroup != 0
}

func (r RewriteRule) SetRewriteGroupByAggr() bool {
	return r&RewriteGroupByAggr != 0
}

func (r RewriteRule) KeepAsIs() bool {
	return r == 0
}

type CompactMode uint8

const (
	CompactNL CompactMode = 1 << iota
	CompactColumns
	CompactValues
	CompactSpacesAround
	CompactAll = CompactNL | CompactColumns | CompactValues
)

func (c CompactMode) None() bool {
	return c == 0
}

func (c CompactMode) KeepSpacesAround() bool {
	return c&CompactSpacesAround == 0
}

func (c CompactMode) ColumnsStacked() bool {
	return c&CompactColumns == 0
}

func (c CompactMode) ValuesStacked() bool {
	return c&CompactValues == 0
}

func (c CompactMode) All() bool {
	return c&CompactAll != 0
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
