package lint

import (
	"fmt"
	"slices"

	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/rules"
)

func checkEnforcedAlias(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectEnforcedAlias(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkEnforcedAlias)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkEnforcedAlias)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkEnforcedAlias)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkEnforcedAlias)
	case ast.CteStatement:
		return checkEnforcedAlias(stmt.Statement)
	case ast.Join:
		return checkEnforcedAlias(stmt.Table)
	case ast.Group:
		return checkEnforcedAlias(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectEnforcedAlias(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	var list []rules.LintMessage
	if cs := ast.GetAliasFromStmt(stmt.Columns); len(cs) == 0 {
		list = append(list, enforcedAlias())
	}
	if ts := ast.GetAliasFromStmt(stmt.Tables); len(ts) == 0 {
		list = append(list, enforcedAlias())
	}
	others, err := handleSelectStatement(stmt, checkEnforcedAlias)
	return slices.Concat(list, others), err
}

func checkUniqueAlias(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectUniqueAlias(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkUniqueAlias)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkUniqueAlias)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkUniqueAlias)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkUniqueAlias)
	case ast.CteStatement:
		return checkUniqueAlias(stmt.Statement)
	case ast.Join:
		return checkUniqueAlias(stmt.Table)
	case ast.Group:
		return checkUniqueAlias(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectUniqueAlias(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	var (
		columns  = ast.GetAliasFromStmt(stmt.Columns)
		tables   = ast.GetAliasFromStmt(stmt.Tables)
		contains = func(list []string, str string) bool {
			return slices.Contains(list, str)
		}
		list []rules.LintMessage
	)
	for i := range columns {
		if ok := contains(columns[i+1:], columns[i]); ok {
			list = append(list, duplicatedAlias(columns[i]))
		}
	}
	for i := range tables {
		if ok := contains(tables[i+1:], tables[i]); ok {
			list = append(list, duplicatedAlias(tables[i]))
		}
	}
	others, err := handleSelectStatement(stmt, checkUniqueAlias)
	return slices.Concat(list, others), err
}

func checkUndefinedAlias(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectUndefinedAlias(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkUndefinedAlias)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkUndefinedAlias)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkUndefinedAlias)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkUndefinedAlias)
	case ast.CteStatement:
		return checkUndefinedAlias(stmt.Statement)
	case ast.Join:
		return checkUndefinedAlias(stmt.Table)
	case ast.Group:
		return checkUndefinedAlias(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectUndefinedAlias(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	var (
		alias  = ast.GetAliasFromStmt(stmt.Tables)
		names  = ast.GetNamesFromStmt(stmt.Tables)
		values = slices.Concat(alias, names)
		list   []rules.LintMessage
	)
	for _, c := range stmt.Columns {
		if a, ok := c.(ast.Alias); ok {
			c = a.Statement
		}
		n, ok := c.(ast.Name)
		if !ok {
			continue
		}
		if schema := n.Schema(); schema != "" && !slices.Contains(values, schema) {
			list = append(list, undefinedAlias(schema))
		}
	}
	others, err := handleSelectStatement(stmt, checkUndefinedAlias)
	return slices.Concat(list, others), err
}

func checkMissingAlias(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectMissingAlias(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkMissingAlias)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkMissingAlias)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkMissingAlias)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkMissingAlias)
	case ast.CteStatement:
		return checkMissingAlias(stmt.Statement)
	case ast.Join:
		return checkMissingAlias(stmt.Table)
	case ast.Group:
		return checkMissingAlias(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectMissingAlias(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	var list []rules.LintMessage
	for _, s := range stmt.Columns {
		if g, ok := s.(ast.Group); ok {
			s = g.Statement
		}
		if _, ok := s.(ast.SelectStatement); ok {
			list = append(list, missingAlias())
		}
	}
	for _, s := range stmt.Tables {
		if j, ok := s.(ast.Join); ok {
			if g, ok := j.Table.(ast.Group); ok {
				s = g.Statement
			} else {
				s = j.Table
			}
		}
		if _, ok := s.(ast.SelectStatement); ok {
			list = append(list, missingAlias())
		}
	}
	others, err := handleSelectStatement(stmt, checkMissingAlias)
	return slices.Concat(list, others), err
}

func checkMisusedAlias(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectMisusedAlias(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkMisusedAlias)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkMisusedAlias)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkMisusedAlias)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkMisusedAlias)
	case ast.CteStatement:
		return checkMisusedAlias(stmt.Statement)
	case ast.Join:
		return checkMisusedAlias(stmt.Table)
	case ast.Group:
		return checkMisusedAlias(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectMisusedAlias(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	var (
		names = ast.GetNamesFromStmt([]ast.Statement{stmt.Where, stmt.Having})
		list  []rules.LintMessage
	)
	for _, a := range ast.GetAliasFromStmt(stmt.Columns) {
		ok := slices.Contains(names, a)
		if ok {
			list = append(list, unexpectedAlias(a))
		}
	}
	others, err := handleSelectStatement(stmt, checkMisusedAlias)
	return slices.Concat(list, others), err
}

func enforcedAlias() rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  "alias expected",
		Rule:     ruleAliasExpected,
	}
}

func unexpectedAlias(alias string) rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  fmt.Sprintf("%s: alias not allowed in where clause", alias),
		Rule:     ruleAliasUnexpected,
	}
}

func undefinedAlias(alias string) rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  fmt.Sprintf("%s: alias not defined", alias),
		Rule:     ruleAliasUndefined,
	}
}

func missingAlias() rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  "alias needed but missing",
		Rule:     ruleAliasMissing,
	}
}

func duplicatedAlias(alias string) rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Error,
		Message:  fmt.Sprintf("%s: alias already defined", alias),
		Rule:     ruleAliasDuplicate,
	}
}
