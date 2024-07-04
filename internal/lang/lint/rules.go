package lint

const (
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

var allRules = map[string][]RuleFunc{
	"alias": {
		checkUniqueAlias,
		checkUndefinedAlias,
		checkMissingAlias,
		checkMisusedAlias,
	},
	ruleAliasUnexpected: {},
	ruleAliasUndefined:  {},
	ruleAliasDuplicate:  {},
	ruleAliasMissing:    {},
	ruleAliasExpected:   {},
	"cte": {
		checkUnusedCte,
		checkDuplicateCte,
	},
	ruleCteUnused:     {},
	ruleCteDuplicated: {},
	"cte.columns": {
		checkColumnsMissingCte,
		checkColumnsMismatchedCte,
	},
	ruleCteColsMissing:         {},
	ruleCteColsMismatched:      {},
	ruleCteColsUnused:          {},
	"subquery":                 {},
	ruleSubqueryNotAllow:       {},
	"subquery.columns":         {},
	ruleSubqueryColsMismatched: {},
	"expr":                     {},
	ruleExprUnqualified:        {},
	ruleExprAggregate:          {},
	ruleExprInvalid:            {},
	"const":                    {},
	ruleConstExprJoin:          {},
	ruleConstExprBin:           {},
	"rewrite":                  {},
	"rewrite.expr":             {},
	ruleRewriteExpr:            {},
	ruleRewriteExprIn:          {},
	ruleRewriteExprNot:         {},
	"inconsistent": {
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
