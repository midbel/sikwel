package lint

import (
	"errors"
	"io"
	"slices"

	"github.com/midbel/sweet/internal/config"
	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/lang/parser"
)

var ErrNa = errors.New("not applicable")

type RuleFunc func(ast.Statement) ([]LintMessage, error)

type Linter struct {
	MinLevel Level
	Max      int
	Rules    []RuleFunc
}

func NewLinter() *Linter {
	var i Linter
	i.prepareRules()
	return &i
}

func (i *Linter) Lint(r io.Reader) ([]LintMessage, error) {
	p, err := parser.NewParser(r)
	if err != nil {
		return nil, err
	}
	var list []LintMessage
	for {
		stmt, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		others, err := i.LintStatement(stmt)
		if err != nil {
			return nil, err
		}
		list = append(list, others...)
		if i.Max > 0 && len(list) >= i.Max {
			list = list[:i.Max]
			break
		}
	}
	return list, nil
}

func (i *Linter) prepareRules() {
	i.Rules = append(i.Rules, checkEnforcedAlias)
	i.Rules = append(i.Rules, checkUniqueAlias)
	i.Rules = append(i.Rules, checkUndefinedAlias)
	i.Rules = append(i.Rules, checkMissingAlias)
	i.Rules = append(i.Rules, checkMisusedAlias)
	i.Rules = append(i.Rules, checkUnusedCte)
	i.Rules = append(i.Rules, checkDuplicateCte)
	i.Rules = append(i.Rules, checkColumnsMissingCte)
	i.Rules = append(i.Rules, checkColumnsMismatchedCte)
	i.Rules = append(i.Rules, checkForSubqueries)
	i.Rules = append(i.Rules, checkForUnqualifiedNames)
	i.Rules = append(i.Rules, checkAsUsage)
	i.Rules = append(i.Rules, checkDirectionUsage)
	i.Rules = append(i.Rules, checkGroupBy)
	i.Rules = append(i.Rules, checkResultSubquery)
	i.Rules = append(i.Rules, checkRewriteIn)
	i.Rules = append(i.Rules, checkRewriteBinary)
	i.Rules = append(i.Rules, checkConstantBinary)
	i.Rules = append(i.Rules, checkJoin)
	i.Rules = append(i.Rules, checkUnusedColumns)
}

func (i *Linter) configure(cfg *config.Config) {

}

func (i *Linter) LintStatement(stmt ast.Statement) ([]LintMessage, error) {
	var list []LintMessage
	for _, r := range i.Rules {
		res, err := r(stmt)
		if err != nil {
			if errors.Is(err, ErrNa) {
				continue
			}
			return nil, err
		}
		list = append(list, res...)
	}
	return list, nil
}

func checkUnusedColumns(stmt ast.Statement) ([]LintMessage, error) {
	return nil, nil
}

func checkJoin(stmt ast.Statement) ([]LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return handleSelectStatement(stmt, checkJoin)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkJoin)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkJoin)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkJoin)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkJoin)
	case ast.CteStatement:
		return checkJoin(stmt.Statement)
	case ast.Join:
		return joinWithConstant(stmt)
	case ast.Group:
		return checkJoin(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func joinWithConstant(stmt ast.Join) ([]LintMessage, error) {
	var check func(ast.Statement) bool

	check = func(stmt ast.Statement) bool {
		switch s := stmt.(type) {
		case ast.Binary:
			return check(s.Left) || check(s.Right)
		case ast.Value:
			return true
		case ast.In:
			return check(s.Ident)
		case ast.Is:
			return check(s.Ident)
		case ast.Between:
			return check(s.Ident)
		default:
			return false
		}
	}
	if check(stmt.Where) {
		return makeArray(constantJoin()), nil
	}
	return nil, nil
}

func checkRewriteIn(stmt ast.Statement) ([]LintMessage, error) {
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

func selectRewriteIn(stmt ast.SelectStatement) ([]LintMessage, error) {
	list, err := lintIn(stmt.Where)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	others, err := handleSelectStatement(stmt, checkRewriteIn)
	return slices.Concat(list, others), err
}

func joinRewriteIn(stmt ast.Join) ([]LintMessage, error) {
	list, err := checkRewriteIn(stmt.Where)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	others, err := checkRewriteIn(stmt.Table)
	return slices.Concat(list, others), err
}

func lintIn(stmt ast.Statement) ([]LintMessage, error) {
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

func checkConstantBinary(stmt ast.Statement) ([]LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectConstantBinary(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkConstantBinary)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkConstantBinary)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkConstantBinary)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkConstantBinary)
	case ast.CteStatement:
		return checkConstantBinary(stmt.Statement)
	case ast.Join:
		return joinConstantBinary(stmt)
	case ast.Group:
		return checkConstantBinary(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectConstantBinary(stmt ast.SelectStatement) ([]LintMessage, error) {
	var list []LintMessage
	if isConstant(stmt.Where) {
		list = append(list, constantOnlyExpr())
	}
	others, err := handleSelectStatement(stmt, checkConstantBinary)
	return slices.Concat(list, others), err
}

func joinConstantBinary(stmt ast.Join) ([]LintMessage, error) {
	var list []LintMessage
	if isConstant(stmt.Where) {
		list = append(list, constantOnlyExpr())
	}
	others, err := checkConstantBinary(stmt.Table)
	return slices.Concat(list, others), err
}

func isConstant(stmt ast.Statement) bool {
	switch s := stmt.(type) {
	case ast.Value:
		return true
	case ast.List:
		for _, v := range s.Values {
			if !isConstant(v) {
				return false
			}
		}
		return true
	case ast.Binary:
		if s.IsRelation() {
			return isConstant(s.Left) || isConstant(s.Right)
		}
		return isConstant(s.Left) && isConstant(s.Right)
	case ast.Is:
		return isConstant(s.Ident)
	case ast.In:
		return isConstant(s.Ident) && isConstant(s.Value)
	case ast.Between:
		return isConstant(s.Ident) && isConstant(s.Lower) && isConstant(s.Upper)
	default:
		return false
	}
}

func checkRewriteBinary(stmt ast.Statement) ([]LintMessage, error) {
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

func selectRewriteBinary(stmt ast.SelectStatement) ([]LintMessage, error) {
	list, err := lintBinary(stmt.Where)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	others, err := handleSelectStatement(stmt, checkRewriteBinary)
	return slices.Concat(list, others), err
}

func joinRewriteBinary(stmt ast.Join) ([]LintMessage, error) {
	list, err := checkRewriteBinary(stmt.Where)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	others, err := checkRewriteBinary(stmt.Table)
	return slices.Concat(list, others), err
}

func lintBinary(stmt ast.Statement) ([]LintMessage, error) {
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

func checkForSubqueries(stmt ast.Statement) ([]LintMessage, error) {
	switch stmt := stmt.(type) {
	case ast.SelectStatement:
		return selectSubqueries(stmt)
	case ast.UnionStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkForSubqueries)
	case ast.IntersectStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkForSubqueries)
	case ast.ExceptStatement:
		return handleCompoundStatement(stmt.Left, stmt.Right, checkForSubqueries)
	case ast.WithStatement:
		return handleWithStatement(stmt, checkForSubqueries)
	case ast.CteStatement:
		return checkForSubqueries(stmt.Statement)
	case ast.Join:
		return checkForSubqueries(stmt.Table)
	case ast.Group:
		return checkForSubqueries(stmt.Statement)
	default:
		return nil, ErrNa
	}
}

func selectSubqueries(stmt ast.SelectStatement) ([]LintMessage, error) {
	isSubquery := func(q ast.Statement) bool {
		if a, ok := q.(ast.Alias); ok {
			q = a.Statement
		}
		if g, ok := q.(ast.Group); ok {
			q = g.Statement
		}
		_, ok := q.(ast.SelectStatement)
		return ok
	}
	var list []LintMessage
	for _, c := range stmt.Columns {
		if isSubquery(c) {
			list = append(list, subqueryDisallow())
		}
	}
	for _, t := range stmt.Tables {
		j, ok := t.(ast.Join)
		if !ok {
			continue
		}
		if isSubquery(j.Table) {
			list = append(list, subqueryDisallow())
		}
	}
	others, err := handleSelectStatement(stmt, checkForSubqueries)
	return slices.Concat(list, others), err
}

func subqueryDisallow() LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  "subquery is not allowed",
		Rule:     ruleSubqueryNotAllow,
	}
}

func rewriteIn() LintMessage {
	return LintMessage{
		Severity: Warning,
		Message:  "in predicate should be rewritten",
		Rule:     ruleRewriteExprIn,
	}
}

func rewriteBinary() LintMessage {
	return LintMessage{
		Severity: Warning,
		Message:  "expression should be rewritten",
		Rule:     ruleRewriteExpr,
	}
}

func constantJoin() LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  "join expression composed of constant values",
		Rule:     ruleExprJoinConst,
	}
}

func constantOnlyExpr() LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  "expression composed of constant values",
		Rule:     ruleExprBinConst,
	}
}

func handleCompoundStatement(q1, q2 ast.Statement, check RuleFunc) ([]LintMessage, error) {
	list1, err := check(q1)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	list2, err := check(q2)
	if err != nil && !errors.Is(err, ErrNa) {
		return nil, err
	}
	return append(list1, list2...), nil
}

func handleWithStatement(with ast.WithStatement, check RuleFunc) ([]LintMessage, error) {
	var list []LintMessage
	for _, q := range with.Queries {
		ms, err := check(q)
		if err != nil {
			if !errors.Is(err, ErrNa) {
				return nil, err
			}
			continue
		}
		list = append(list, ms...)
	}
	ms, err := check(with.Statement)
	if err == nil {
		list = append(list, ms...)
	}
	return list, err
}

func handleSelectStatement(stmt ast.SelectStatement, check RuleFunc) ([]LintMessage, error) {
	var list []LintMessage
	for _, c := range stmt.Columns {
		msg, err := check(c)
		if err != nil && !errors.Is(err, ErrNa) {
			return nil, err
		}
		list = append(list, msg...)
	}
	for _, c := range stmt.Tables {
		msg, err := check(c)
		if err != nil && !errors.Is(err, ErrNa) {
			return nil, err
		}
		list = append(list, msg...)
	}
	return list, nil
}

func makeArray[T LintMessage](el T) []T {
	return []T{el}
}
