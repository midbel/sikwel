package lint

import (
	"errors"
	"slices"

	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/rules"
)

func checkRewriteIn(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectRewriteIn(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkRewriteIn)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkRewriteIn)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkRewriteIn)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkRewriteIn)
	case ast.CteStatement:
		return checkRewriteIn(stmt.Statement)
	case ast.Join:
		return joinRewriteIn(stmt)
	case ast.Group:
		return checkRewriteIn(stmt.Statement)
	case ast.Binary:
		return lintIn(stmt)
	case ast.In:
		return lintIn(stmt)
	default:
		return nil, ErrNa
	}
}

func selectRewriteIn(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	list, err := lintIn(stmt.Where)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	others, err := handleSelectStatement(stmt, checkRewriteIn)
	return slices.Concat(list, others), err
}

func joinRewriteIn(stmt ast.Join) ([]rules.LintMessage, error) {
	list, err := checkRewriteIn(stmt.Where)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	others, err := checkRewriteIn(stmt.Table)
	return slices.Concat(list, others), err
}

func lintIn(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.In:
		vs, ok := stmt.Value.(ast.List)
		if !ok {
			return nil, nil
		}
		if len(vs.Values) == 1 {
			return makeArray(rewriteIn()), nil
		}
		return nil, nil
	case ast.Binary:
		if !stmt.IsRelation() {
			return nil, ErrNa
		}
		l1, err1 := lintIn(stmt.Left)
		if err1 != nil && !errors.Is(err1, ErrNa) {
			return nil, err1
		}
		l2, err2 := lintIn(stmt.Right)
		if err2 != nil && !errors.Is(err2, ErrNa) {
			return nil, err2
		}
		return slices.Concat(l1, l2), nil
	default:
		return nil, ErrNa
	}
}

func checkRewriteBinary(stmt ast.Statement) ([]rules.LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectRewriteBinary(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkRewriteBinary)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkRewriteBinary)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkRewriteBinary)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkRewriteBinary)
	case ast.CteStatement:
		return checkRewriteBinary(stmt.Statement)
	case ast.Join:
		return joinRewriteBinary(stmt)
	case ast.Group:
		return checkRewriteBinary(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectRewriteBinary(stmt ast.SelectStatement) ([]rules.LintMessage, error) {
	list, err := lintBinary(stmt.Where)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	others, err := handleSelectStatement(stmt, checkRewriteBinary)
	return slices.Concat(list, others), err
}

func joinRewriteBinary(stmt ast.Join) ([]rules.LintMessage, error) {
	list, err := checkRewriteBinary(stmt.Where)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	others, err := checkRewriteBinary(stmt.Table)
	return slices.Concat(list, others), err
}

func lintBinary(stmt ast.Statement) ([]rules.LintMessage, error) {
	bin, ok := stmt.(ast.Binary)
	if !ok {
		return nil, ErrNa
	}
	if bin.IsRelation() {
		l1, err1 := lintBinary(bin.Left)
		if err1 != nil && !errors.Is(err1, ErrNa) {
			return nil, err1
		}
		l2, err2 := lintBinary(bin.Right)
		if err2 != nil && !errors.Is(err2, ErrNa) {
			return nil, err2
		}
		return slices.Concat(l1, l2), nil
	}
	if bin.Op == "=" || bin.Op == "<>" {
		if v, ok := bin.Right.(ast.Value); ok && v.Constant() {
			return makeArray(rewriteBinary()), nil
		}
		if v, ok := bin.Left.(ast.Value); ok && v.Constant() {
			return makeArray(rewriteBinary()), nil
		}
	}
	return nil, ErrNa
}

func rewriteIn() rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Warning,
		Message:  "in predicate should be rewritten",
		Rule:     ruleRewriteExprIn,
	}
}

func rewriteBinary() rules.LintMessage {
	return rules.LintMessage{
		Severity: rules.Warning,
		Message:  "expression should be rewritten",
		Rule:     ruleRewriteExpr,
	}
}
