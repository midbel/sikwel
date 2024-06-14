package lint

import "fmt"

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

func unexpectedAlias(alias string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: alias found in predicate", alias),
		Rule:     ruleAliasUnexpected,
	}
}

func undefinedAlias(alias string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: alias not defined", alias),
		Rule:     ruleAliasUndefined,
	}
}

func missingAlias() LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  "expression needs to be used with an alias",
		Rule:     ruleAliasMissing,
	}
}

func duplicatedAlias(alias string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: alias already defined", alias),
		Rule:     ruleAliasDuplicate,
	}
}

const (
	ruleCountMulti      = "count.mulitple"
	ruleCountInvalid    = "count.invalid"
	ruleExprGroup       = "expression.group"
	ruleExprAggregate   = "expression.aggregate"
	ruleExprInvalid     = "expression.invalid"
	ruleAliasUnexpected = "alias.unexpected"
	ruleAliasUndefined  = "alias.undefined"
	ruleAliasDuplicate  = "alias.duplicate"
	ruleAliasMissing    = "alias.missing"
)
