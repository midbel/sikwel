package lint

import (
	"fmt"

	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/rules"
)

func checkDuplicateCte(stmt ast.Statement) ([]LintMessage, error) {
	with, ok := stmt.(ast.WithStatement)
	if !ok {
		return nil, ErrNa
	}
	var (
		seen = make(map[string]struct{})
		list []LintMessage
	)
	for _, q := range with.Queries {
		c, ok := q.(ast.CteStatement)
		if !ok {
			return nil, fmt.Errorf("cte expected! got %T", q)
		}
		if _, ok := seen[c.Ident]; ok {
			list = append(list, cteDuplicate(c.Ident))
			continue
		}
		seen[c.Ident] = struct{}{}
	}
	return list, nil
}

func checkUnusedCte(stmt ast.Statement) ([]LintMessage, error) {
	with, ok := stmt.(ast.WithStatement)
	if !ok {
		return nil, ErrNa
	}
	var (
		all  = make(map[string]struct{})
		list []LintMessage
	)
	for _, q := range with.Queries {
		c, ok := q.(ast.CteStatement)
		if !ok {
			return nil, fmt.Errorf("cte expected! got %T", q)
		}
		all[c.Ident] = struct{}{}
	}
	for _, n := range with.GetNames() {
		delete(all, n)
	}
	for n := range all {
		list = append(list, cteUnused(n))
	}

	return list, nil
}

func checkColumnsMissingCte(stmt ast.Statement) ([]LintMessage, error) {
	with, ok := stmt.(ast.WithStatement)
	if !ok {
		return nil, ErrNa
	}
	var list []LintMessage
	for _, q := range with.Queries {
		c, ok := q.(ast.CteStatement)
		if !ok {
			return nil, fmt.Errorf("cte expected! got %T", q)
		}
		if len(c.Columns) == 0 {
			list = append(list, cteColumnsMissing(c.Ident))
		}
	}
	return list, nil
}

func checkColumnsUnsedCte(stmt ast.Statement) ([]LintMessage, error) {
	return nil, nil
}

func checkColumnsMismatchedCte(stmt ast.Statement) ([]LintMessage, error) {
	with, ok := stmt.(ast.WithStatement)
	if !ok {
		return nil, ErrNa
	}
	var list []LintMessage
	for _, q := range with.Queries {
		c, ok := q.(ast.CteStatement)
		if !ok {
			return nil, fmt.Errorf("cte expected! got %T", q)
		}
		q, ok := c.Statement.(ast.SelectStatement)
		if !ok {
			return nil, fmt.Errorf("select expected! got %T", q)
		}
		if len(c.Columns) != len(q.Columns) {
			list = append(list, cteColumnsMismatched(c.Ident))
		}
	}
	return list, nil
}

func cteColumnsMismatched(cte string) LintMessage {
	return LintMessage{
		Severity: rules.Error,
		Message:  fmt.Sprintf("%s: columns count mismatched", cte),
		Rule:     ruleCteColsMismatched,
	}
}

func cteColumnsMissing(cte string) LintMessage {
	return LintMessage{
		Severity: rules.Error,
		Message:  fmt.Sprintf("%s: no columns defined for cte", cte),
		Rule:     ruleCteColsMissing,
	}
}

func cteDuplicate(cte string) LintMessage {
	return LintMessage{
		Severity: rules.Error,
		Message:  fmt.Sprintf("%s: cte already defined", cte),
		Rule:     ruleCteDuplicated,
	}
}

func cteUnused(cte string) LintMessage {
	return LintMessage{
		Severity: rules.Error,
		Message:  fmt.Sprintf("%s: cte declared but not used", cte),
		Rule:     ruleCteUnused,
	}
}
