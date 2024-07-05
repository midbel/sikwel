package lint

import (
	"fmt"
	"slices"

	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/rules"
)

func checkResultSubquery(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectResultSubquery(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkResultSubquery)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkResultSubquery)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkResultSubquery)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkResultSubquery)
	case ast.CteStatement:
		return checkResultSubquery(stmt.Statement)
	case ast.Join:
		return checkResultSubquery(stmt.Table)
	case ast.Group:
		return checkResultSubquery(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectResultSubquery(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	var list []rules.LintMessage
	for _, c := range stmt.Columns {
		q, ok := c.(ast.SelectStatement)
		if !ok {
			continue
		}
		if len(q.Columns) != 1 {
			list = append(list, subqueryTooManyResult())
		}
	}
	others, err := handleSelectStatement(stmt, checkResultSubquery)
	return slices.Concat(list, others), err
}

func checkGroupBy(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectGroupBy(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkGroupBy)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkGroupBy)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkGroupBy)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkGroupBy)
	case ast.CteStatement:
		return checkGroupBy(stmt.Statement)
	case ast.Join:
		return checkGroupBy(stmt.Table)
	case ast.Group:
		return checkGroupBy(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectGroupBy(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	if len(stmt.Groups) == 0 {
		return nil, nil
	}
	var (
		list   []rules.LintMessage
		groups = ast.GetNamesFromStmt(stmt.Groups)
	)
	for _, c := range stmt.Columns {
		if a, ok := c.(ast.Alias); ok {
			c = a.Statement
		}
		switch c := c.(type) {
		case ast.Value:
		case ast.Name:
			if !slices.Contains(groups, c.Ident()) {
				list = append(list, exprNotInGroupBy(c.Ident()))
			}
		case ast.Call:
			if !c.IsAggregate() {
				list = append(list, aggregateExpected(c.GetIdent()))
			}
		default:
			list = append(list, unexpectedExpr(""))
		}
	}
	others, err := handleSelectStatement(stmt, checkGroupBy)
	return slices.Concat(list, others), err
}

func checkAsUsage(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectInconsistentAs(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkAsUsage)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkAsUsage)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkAsUsage)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkAsUsage)
	case ast.CteStatement:
		return checkAsUsage(stmt.Statement)
	case ast.Join:
		return checkAsUsage(stmt.Table)
	case ast.Group:
		return checkAsUsage(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectInconsistentAs(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	var (
		list []rules.LintMessage
		used bool
	)
	for _, c := range stmt.Columns {
		a, ok := c.(ast.Alias)
		if !ok {
			continue
		}
		if !used && a.As {
			used = true
		}
	}
	if used && len(stmt.Columns) > 1 {
		list = append(list, inconsistentAs("select"))
	}
	used = false
	for _, s := range stmt.Tables {
		if j, ok := s.(ast.Join); ok {
			s = j.Table
		}
		a, ok := s.(ast.Alias)
		if !ok {
			continue
		}
		if !used && a.As {
			used = true
			continue
		}
	}
	if used && len(stmt.Tables) > 1 {
		list = append(list, inconsistentAs("from"))
	}
	others, err := handleSelectStatement(stmt, checkAsUsage)
	return slices.Concat(list, others), err
}

func checkDirectionUsage(stmt ast.Statement) ([]rules.LintMessage, error) {
	return nil, nil
}

func checkForUnqualifiedNames(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectUnqualifiedNames(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkForUnqualifiedNames)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkForUnqualifiedNames)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkForUnqualifiedNames)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkForUnqualifiedNames)
	case ast.CteStatement:
		return checkForUnqualifiedNames(stmt.Statement)
	case ast.Join:
		return checkForUnqualifiedNames(stmt.Table)
	case ast.Group:
		return checkForUnqualifiedNames(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectUnqualifiedNames(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	var (
		names = ast.GetAliasFromStmt(stmt.Columns)
		list  []rules.LintMessage
	)
	for _, c := range stmt.Columns {
		if a, ok := c.(ast.Alias); ok {
			c = a.Statement
		}
		n, ok := c.(ast.Name)
		if !ok {
			continue
		}
		if len(n.Parts) == 1 && len(names) > 0 {
			list = append(list, unqualifiedName(n.Ident()))
		}
	}
	others, err := handleSelectStatement(stmt, checkForUnqualifiedNames)
	return slices.Concat(list, others), err
}

func unqualifiedName(name string) rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  fmt.Sprintf("%s should be fully qualified", name),
		Rule:     ruleExprUnqualified,
	}
}

func inconsistentAs(clause string) rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Warning,
		Message:  fmt.Sprintf("%s: inconsistent use of AS", clause),
		Rule:     ruleInconsistentUseAs,
	}
}

func inconsistentOrder() rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Warning,
		Message:  "inconsistent use of ASC/DESC",
		Rule:     ruleInconsistentUseOrder,
	}
}

func aggregateExpected(ident string) rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  fmt.Sprintf("%s should be an aggregate function", ident),
		Rule:     ruleExprAggregate,
	}
}

func exprNotInGroupBy(ident string) rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  fmt.Sprintf("%s should be used in group by clause", ident),
		Rule:     ruleExprInvalid,
	}
}

func unexpectedExpr(ident string) rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  "%s: unexpected expression",
		Rule:     ruleExprInvalid,
	}
}

func subqueryTooManyResult() rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  "too many result returned by subquery",
		Rule:     ruleSubqueryColsMismatched,
	}
}
