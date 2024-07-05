package lint

import (
	"fmt"
	"slices"
	"strings"

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

type RuleFunc func(ast.Statement) ([]LintMessage, error)

var allRules = map[string]RuleFunc{
	ruleAliasUnexpected:        checkMisusedAlias,
	ruleAliasUndefined:         checkUndefinedAlias,
	ruleAliasDuplicate:         checkUniqueAlias,
	ruleAliasMissing:           checkMissingAlias,
	ruleAliasExpected:          checkEnforcedAlias,
	ruleCteUnused:              checkUnusedCte,
	ruleCteDuplicated:          checkDuplicateCte,
	ruleCteColsMissing:         checkColumnsMissingCte,
	ruleCteColsMismatched:      checkColumnsMismatchedCte,
	ruleCteColsUnused:          checkColumnsUnsedCte,
	ruleSubqueryNotAllow:       checkSubqueriesNotAllow,
	ruleSubqueryColsMismatched: checkResultSubquery,
	ruleExprUnqualified:        checkForUnqualifiedNames,
	ruleExprAggregate:          nil,
	ruleExprInvalid:            nil,
	ruleConstExprJoin:          checkJoin,
	ruleConstExprBin:           checkConstantBinary,
	ruleRewriteExpr:            nil,
	ruleRewriteExprIn:          checkRewriteIn,
	ruleRewriteExprNot:         nil,
	ruleInconsistentUseAs:      checkAsUsage,
	ruleInconsistentUseOrder:   checkDirectionUsage,
}

func GetRuleNames() []string {
	return []string{
		ruleAlias,
		ruleCte,
		ruleSubquery,
		ruleExpr,
		ruleConst,
		ruleRewrite,
		ruleInconsistent,
		ruleAliasUnexpected,
		ruleAliasUndefined,
		ruleAliasDuplicate,
		ruleAliasMissing,
		ruleAliasExpected,
		ruleCteUnused,
		ruleCteDuplicated,
		ruleCteColsMissing,
		ruleCteColsMismatched,
		ruleCteColsUnused,
		ruleSubqueryNotAllow,
		ruleSubqueryColsMismatched,
		ruleExprUnqualified,
		ruleExprAggregate,
		ruleExprInvalid,
		ruleConstExprJoin,
		ruleConstExprBin,
		ruleRewriteExpr,
		ruleRewriteExprIn,
		ruleRewriteExprNot,
		ruleInconsistentUseAs,
		ruleInconsistentUseOrder,
	}
}

func getRulesByName(rule string) ([]registeredRule, error) {
	var (
		set   []registeredRule
		group string
		parts = strings.Split(rule, ".")
	)
	if len(parts) == 3 {
		fn, ok := allRules[rule]
		if !ok {
			return nil, fmt.Errorf("no such rule %s", rule)
		}
		r := registeredRule{
			Name: rule,
			Func: fn,
		}
		return append(set, r), nil
	}
	group = strings.Join(parts[:len(parts)-1], ".")
	for k, fn := range allRules {
		if !strings.HasPrefix(k, group) {
			continue
		}
		r := registeredRule{
			Name: k,
			Func: fn,
		}
		set = append(set, r)
	}
	return set, nil
}

const defaultPriority = 100

type registeredRule struct {
	Name     string
	Func     RuleFunc
	Priority int
}

type rulesMap map[string]registeredRule

func getDefaultRules() rulesMap {
	all := make(rulesMap)
	all.register(ruleAliasUnexpected, checkMisusedAlias)
	all.register(ruleAliasUndefined, checkUndefinedAlias)
	all.register(ruleAliasDuplicate, checkUniqueAlias)
	all.register(ruleAliasMissing, checkMissingAlias)
	return all
}

func (r rulesMap) Get() []RuleFunc {
	var (
		tmp []registeredRule
		all []RuleFunc
	)
	for _, fn := range r {
		tmp = append(tmp, fn)
	}
	slices.SortFunc(tmp, func(a, b registeredRule) int {
		return a.Priority - b.Priority
	})
	for i := range tmp {
		all = append(all, tmp[i].Func)
	}
	return all
}

func (r rulesMap) Register(name string, priority int, fn RuleFunc) {
	r[name] = registeredRule{
		Name:     name,
		Func:     fn,
		Priority: priority,
	}
}

func (r rulesMap) register(name string, fn RuleFunc) {
	r.Register(name, defaultPriority, fn)
}

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
