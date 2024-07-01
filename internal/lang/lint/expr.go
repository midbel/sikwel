package lint

import (
	"fmt"

	"github.com/midbel/sweet/internal/lang/ast"
)

func checkForUnqualifiedNames(stmt ast.Statement) ([]LintMessage, error) {
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
	default:
		return nil, ErrNa
	}
}

func selectUnqualifiedNames(stmt ast.SelectStatement) ([]LintMessage, error) {
	var (
		names = ast.GetAliasFromStmt(stmt.Columns)
		list  []LintMessage
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
	return list, nil
}

func unqualifiedName(name string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: expr is not qualified", name),
		Rule:     ruleExprUnqualified,
	}
}
