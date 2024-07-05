package lint

import (
	"github.com/midbel/sweet/internal/lang/ast"
)

const (
	ruleAlias                  = "alias"
	ruleCte                    = "cte"
	ruleSubquery               = "subquery"
	ruleExpr                   = "expr"
	ruleConst                  = "const"
	ruleRewrite                = "rewrite"
	ruleInconsistent           = "inconsistent"
	ruleAliasUnexpected        = "alias.unexpected"
	ruleAliasUndefined         = "alias.undefined"
	ruleAliasDuplicate         = "alias.duplicate"
	ruleAliasMissing           = "alias.missing"
	ruleAliasExpected          = "alias.require"
	ruleCteUnused              = "cte.unused"
	ruleCteDuplicated          = "cte.duplicate"
	ruleCteColsMissing         = "cte.columns.missing"
	ruleCteColsMismatched      = "cte.columns.mismatched"
	ruleCteColsUnused          = "cte.columns.unused"
	ruleSubqueryNotAllow       = "subquery.disallow"
	ruleSubqueryColsMismatched = "subquery.columns.mismatched"
	ruleExprUnqualified        = "expr.unqalified"
	ruleExprAggregate          = "expr.aggregate"
	ruleExprInvalid            = "expr.invalid"
	ruleConstExprJoin          = "const.expr.join"
	ruleConstExprBin           = "const.expr.bin"
	ruleRewriteExpr            = "rewrite.expr"
	ruleRewriteExprIn          = "rewrite.expr.in"
	ruleRewriteExprNot         = "rewrite.expr.not"
	ruleInconsistentUseAs      = "inconsistent.use.as"
	ruleInconsistentUseOrder   = "inconsistent.use.order"
)

func customizeRule(fn RuleFunc, enabled bool, level Level) RuleFunc {
	return func(stmt ast.Statement) ([]LintMessage, error) {
		if !enabled {
			return nil, nil
		}
		ms, err := fn(stmt)
		if level == Default {
			return ms, err
		}
		for i := range ms {
			ms[i].Severity = level
		}
		return ms, err
	}
}

type rulesMap map[string][]RuleFunc

func (r rulesMap) Get(rule string) ([]RuleFunc, error) {
	return nil, nil
}

var allRules = rulesMap{
	ruleAlias: {
		checkUniqueAlias,
		checkUndefinedAlias,
		checkMissingAlias,
		checkMisusedAlias,
	},
	ruleAliasUnexpected: {
		checkMisusedAlias,
	},
	ruleAliasUndefined: {
		checkUndefinedAlias,
	},
	ruleAliasDuplicate: {
		checkUniqueAlias,
	},
	ruleAliasMissing: {
		checkMissingAlias,
	},
	ruleAliasExpected: {
		checkEnforcedAlias,
	},
	ruleCte: {
		checkUnusedCte,
		checkDuplicateCte,
	},
	ruleCteUnused: {
		checkUnusedCte,
	},
	ruleCteDuplicated: {
		checkDuplicateCte,
	},
	"cte.columns": {
		checkColumnsMissingCte,
		checkColumnsMismatchedCte,
	},
	ruleCteColsMissing: {
		checkColumnsMissingCte,
	},
	ruleCteColsMismatched: {
		checkColumnsMismatchedCte,
	},
	ruleCteColsUnused: {},
	ruleSubquery: {
		checkForSubqueries,
		checkResultSubquery,
	},
	ruleSubqueryNotAllow: {
		checkForSubqueries,
	},
	"subquery.columns": {
		checkResultSubquery,
	},
	ruleSubqueryColsMismatched: {
		checkResultSubquery,
	},
	ruleExpr: {
		checkForUnqualifiedNames,
	},
	ruleExprUnqualified: {
		checkForUnqualifiedNames,
	},
	ruleExprAggregate: {},
	ruleExprInvalid:   {},
	ruleConst:         {},
	ruleConstExprJoin: {},
	ruleConstExprBin:  {},
	ruleRewrite: {
		checkRewriteBinary,
		checkRewriteIn,
	},
	ruleRewriteExpr: {
		checkRewriteBinary,
		checkRewriteIn,
	},
	ruleRewriteExprIn:  {},
	ruleRewriteExprNot: {},
	ruleInconsistent: {
		checkAsUsage,
	},
	ruleInconsistentUseAs:    {},
	ruleInconsistentUseOrder: {},
}

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

func getLevelFromName(name string) Level {
	switch name {
	case "", "default":
		return Default
	case "info":
		return Info
	case "warning":
		return Warning
	case "error":
		return Error
	default:
		return -1
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
