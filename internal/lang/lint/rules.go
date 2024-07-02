package lint

const (
	ruleCountMulti           = "count.multiple"
	ruleExprRewriteNot       = "expression.rewrite.not"
	ruleJoinCondition        = "join.condition"
	ruleAliasUnexpected      = "alias.unexpected"
	ruleAliasUndefined       = "alias.undefined"
	ruleAliasDuplicate       = "alias.duplicate"
	ruleAliasMissing         = "alias.missing"
	ruleAliasExpected        = "alias.expected"
	ruleCteUnused            = "cte.unused"
	ruleCteDuplicated        = "cte.duplicate"
	ruleCteColsMissing       = "cte.columns.missing"
	ruleCteColsMismatched    = "cte.columns.mismatched"
	ruleSubqueryNotAllow     = "subquery.disallow"
	ruleSubqueryCountInvalid = "subquery.count.invalid"
	ruleExprUnqualified      = "expr.unqalified"
	ruleExprAggregate        = "expr.aggregate"
	ruleExprInvalid          = "expr.invalid"
	ruleExprRewrite          = "expr.rewrite"
	ruleExprRewriteIn        = "expr.rewrite.in"
	ruleInconsistentUseAs    = "use.as.inconsistent"
	ruleInconsistentUseOrder = "use.order.inconsistent"
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

func constantExprInJoin() LintMessage {
	return LintMessage{
		Severity: Warning,
		Message:  "constant expression used in join condition",
		Rule:     ruleJoinCondition,
	}
}

func notStandardOperator() LintMessage {
	return LintMessage{
		Severity: Warning,
		Message:  "non standard operator found",
		Rule:     ruleExprRewrite,
	}
}
