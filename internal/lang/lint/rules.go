package lint

const (
	ruleAliasUnexpected      = "alias.unexpected"
	ruleAliasUndefined       = "alias.undefined"
	ruleAliasDuplicate       = "alias.duplicate"
	ruleAliasMissing         = "alias.missing"
	ruleAliasExpected        = "alias.require"
	ruleCteUnused            = "cte.unused"
	ruleCteDuplicated        = "cte.duplicate"
	ruleCteColsMissing       = "cte.columns.missing"
	ruleCteColsMismatched    = "cte.columns.mismatched"
	ruleCteColsUnused        = "cte.columns.unused"
	ruleSubqueryNotAllow     = "subquery.disallow"
	ruleSubqueryTooMany      = "subquery.result.mismatched"
	ruleExprUnqualified      = "expr.unqalified"
	ruleExprAggregate        = "expr.aggregate"
	ruleExprInvalid          = "expr.invalid"
	ruleExprJoinConst        = "expr.join.constant"
	ruleExprBinConst         = "expr.bin.constant"
	ruleRewriteExpr          = "rewrite.expr."
	ruleRewriteExprIn        = "rewrite.expr.in"
	ruleRewriteExprNot       = "rewrite.expr.not"
	ruleInconsistentUseAs    = "inconsistent.use.as"
	ruleInconsistentUseOrder = "inconsistent.use.order"
)

type Level int

func (e Level) String() string {
	switch e {
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "other"
	}
}

const (
	Default Level = iota
	Debug
	Info
	Warning
	Error
)

type LintMessage struct {
	Severity Level
	Rule     string
	Message  string
}
