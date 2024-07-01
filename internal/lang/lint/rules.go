package lint

import "fmt"

const (
	ruleCountMulti        = "count.multiple"
	ruleCountInvalid      = "count.invalid"
	ruleExprRewrite       = "expression.rewrite"
	ruleExprRewriteNot    = "expression.rewrite.not"
	ruleExprRewriteIn     = "expression.rewrite.in"
	ruleExprGroup         = "expression.group"
	ruleExprAggregate     = "expression.aggregate"
	ruleExprInvalid       = "expression.invalid"
	ruleExprUnused        = "expression.unused"
	ruleJoinCondition     = "join.condition"
	ruleAliasUnexpected   = "alias.unexpected"
	ruleAliasUndefined    = "alias.undefined"
	ruleAliasDuplicate    = "alias.duplicate"
	ruleAliasMissing      = "alias.missing"
	ruleAliasExpected     = "alias.expected"
	ruleCteUnused         = "cte.unused"
	ruleCteDuplicated     = "cte.duplicate"
	ruleCteColsMissing    = "cte.columns.missing"
	ruleCteColsMismatched = "cte.columns.mismatched"
	ruleSubqueryNotAllow  = "subquery.disallow"
	ruleExprUnqualified   = "expression.unqalified"
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

func oneValueWithInPredicate() LintMessage {
	return LintMessage{
		Severity: Warning,
		Message:  "only one value used with in can be rewritten with comparison operator",
		Rule:     ruleExprRewrite,
	}
}

func rewritableBinaryExpr() LintMessage {
	return LintMessage{
		Severity: Warning,
		Message:  "expression can be rewritten",
		Rule:     ruleExprRewrite,
	}
}

func notStandardOperator() LintMessage {
	return LintMessage{
		Severity: Warning,
		Message:  "non standard operator found",
		Rule:     ruleExprRewrite,
	}
}

func countMultipleFields() LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  "subquery should only return one field",
		Rule:     ruleCountMulti,
	}
}

func columnsCountMismatched() LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  "columns count mismatched",
		Rule:     ruleCountInvalid,
	}
}

func fieldNotInGroup(field string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: field should be used in a 'group by' clause or with an aggregate function", field),
		Rule:     ruleExprGroup,
	}
}

func aggregateFunctionExpected(ident string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: aggregate function expected", ident),
		Rule:     ruleExprAggregate,
	}
}

func unexpectedExprType(field, ctx string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: unexpected expression type", ctx),
		Rule:     ruleExprInvalid,
	}
}
