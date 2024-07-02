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
		return selectJoin(stmt)
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
	default:
		return nil, ErrNa
	}
}

func selectJoin(stmt ast.SelectStatement) ([]LintMessage, error) {
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
	default:
		return nil, ErrNa
	}
}

func selectRewriteIn(stmt ast.SelectStatement) ([]LintMessage, error) {
	if stmt.Where == nil {
		return nil, nil
	}
	var check func(ast.Statement) ([]LintMessage, error)

	check = func(stmt ast.Statement) ([]LintMessage, error) {
		switch w := stmt.(type) {
		case ast.In:
			vs, ok := w.Value.(ast.List)
			if !ok {
				return nil, nil
			}
			if len(vs.Values) == 1 {
				return []LintMessage{rewriteIn()}, nil
			}
			return nil, nil
		case ast.Binary:
			if !w.IsRelation() {
				return nil, nil
			}
			l1, err1 := check(w.Left)
			if err1 != nil && !errors.Is(err1, ErrNa) {
				return nil, err1
			}
			l2, err2 := check(w.Right)
			if err2 != nil && !errors.Is(err2, ErrNa) {
				return nil, err2
			}
			return slices.Concat(l1, l2), nil
		default:
			return nil, ErrNa
		}

	}
	return check(stmt.Where)
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
	default:
		return nil, ErrNa
	}
}

func selectRewriteBinary(stmt ast.SelectStatement) ([]LintMessage, error) {
	if stmt.Where == nil {
		return nil, nil
	}

	var check func(ast.Statement) ([]LintMessage, error)

	check = func(stmt ast.Statement) ([]LintMessage, error) {
		b, ok := stmt.(ast.Binary)
		if !ok {
			return nil, ErrNa
		}
		if b.IsRelation() {
			l1, err1 := check(b.Left)
			if err1 != nil && !errors.Is(err1, ErrNa) {
				return nil, err1
			}
			l2, err2 := check(b.Right)
			if err2 != nil && !errors.Is(err2, ErrNa) {
				return nil, err2
			}
			return slices.Concat(l1, l2), nil
		}
		if b.Op == "=" || b.Op == "<>" {
			if v, ok := b.Right.(ast.Value); ok && v.Constant() {
				return []LintMessage{rewriteBinary()}, nil
			}
			if v, ok := b.Left.(ast.Value); ok && v.Constant() {
				return []LintMessage{rewriteBinary()}, nil
			}
		}
		return nil, ErrNa
	}
	return check(stmt.Where)
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
	return list, nil
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
		Message:  "in predicate could be rewritten",
		Rule:     ruleExprRewriteIn,
	}
}

func rewriteBinary() LintMessage {
	return LintMessage{
		Severity: Warning,
		Message:  "expression can be rewritten",
		Rule:     ruleExprRewrite,
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
