package lint

import (
	"fmt"
	"strings"

	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/rules"
)

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

type RuleFunc = rules.RuleFunc[ast.Statement]

type registeredRule = rules.RegisteredRule[ast.Statement]

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
	var ns []string
	for n := range allRules {
		ns = append(ns, n)
	}
	return ns
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

func getDefaultRules() rules.Map[ast.Statement] {
	list := []string{
		ruleAliasUnexpected,
		ruleAliasUndefined,
		ruleAliasDuplicate,
		ruleAliasMissing,
		ruleCteUnused,
		ruleCteDuplicated,
		ruleCteColsMissing,
		ruleCteColsMismatched,
		ruleConstExprJoin,
		ruleConstExprBin,
		ruleConstExprBin,
		ruleSubqueryColsMismatched,
		ruleInconsistentUseAs,
		ruleInconsistentUseOrder,
	}
	all := make(rules.Map[ast.Statement])
	for _, n := range list {
		all.Register(n, defaultPriority, allRules[n])
	}
	return all
}

func customizeRule(fn RuleFunc, enabled bool, level rules.Level) RuleFunc {
	return func(stmt ast.Statement) ([]rules.LintMessage, error) {
		if !enabled {
			return nil, nil
		}
		ms, err := fn(stmt)
		if level == rules.Default {
			return ms, err
		}
		for i := range ms {
			ms[i].Severity = level
		}
		return ms, err
	}
}
