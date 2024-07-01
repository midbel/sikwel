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

func (i *Linter) lintIn(stmt ast.In) ([]LintMessage, error) {
	if vs, ok := stmt.Value.(ast.List); ok && len(vs.Values) == 1 {
		return []LintMessage{oneValueWithInPredicate()}, nil
	}
	return i.LintStatement(stmt.Value)
}

func (i *Linter) lintBinary(stmt ast.Binary) ([]LintMessage, error) {
	l1, err1 := i.LintStatement(stmt.Left)
	if err1 != nil {
		return nil, err1
	}
	l2, err2 := i.LintStatement(stmt.Right)
	if err2 != nil {
		return nil, err2
	}
	list := slices.Concat(l1, l2)

	if stmt.IsRelation() {
		return list, nil
	}
	if stmt.Op == "!=" {
		list = append(list, notStandardOperator())
	}
	if stmt.Op == "=" || stmt.Op == "<>" {
		v, ok := stmt.Right.(ast.Value)
		if ok && v.Constant() {
			list = append(list, rewritableBinaryExpr())
		}
	}
	return list, nil
}

func (i *Linter) lintList(stmt ast.List) ([]LintMessage, error) {
	var list []LintMessage
	for _, v := range stmt.Values {
		others, err := i.LintStatement(v)
		if err != nil {
			return nil, err
		}
		list = append(list, others...)
	}
	return list, nil
}

func (i *Linter) lintSets(s1, s2 ast.SelectStatement) ([]LintMessage, error) {
	var (
		list   []LintMessage
		others []LintMessage
		err    error
	)
	if c1, c2 := s1.ColumnsCount(), s2.ColumnsCount(); c1 != c2 || c1 < 0 || c2 < 0 {
		list = append(list, columnsCountMismatched())
	}
	others, err = i.LintStatement(s1)
	if err != nil {
		return nil, err
	}
	list = append(list, others...)
	others, err = i.LintStatement(s2)
	if err != nil {
		return nil, err
	}
	list = append(list, others...)
	return list, nil
}

func (i *Linter) lintCte(stmt ast.CteStatement) ([]LintMessage, error) {
	var list []LintMessage
	if z := len(stmt.Columns); z != 0 {
		if c, ok := stmt.Statement.(interface{ ColumnsCount() int }); ok {
			n := c.ColumnsCount()
			if n != z {
				list = append(list, columnsCountMismatched())
			}
		}
	}
	others, err := i.LintStatement(stmt.Statement)
	if err != nil {
		return nil, err
	}
	list = append(list, others...)
	return list, nil
}

func (i *Linter) lintValues(stmt ast.ValuesStatement) ([]LintMessage, error) {
	if len(stmt.List) <= 1 {
		return nil, nil
	}
	var (
		list  []LintMessage
		count = 1
	)
	others, err := i.LintStatement(stmt.List[0])
	if err != nil {
		return nil, err
	}
	list = append(list, others...)
	if vs, ok := stmt.List[0].(ast.List); ok {
		count = len(vs.Values)
	}
	for _, vs := range stmt.List[1:] {
		others, err := i.LintStatement(vs)
		if err != nil {
			return nil, err
		}
		list = append(list, others...)

		n := 1
		if vs, ok := vs.(ast.List); ok {
			n = len(vs.Values)
		}
		if count != n {
			list = append(list, columnsCountMismatched())
		}
	}
	return list, nil
}

func checkFieldsFromSubqueries(stmt ast.SelectStatement) []LintMessage {
	var list []LintMessage
	for _, c := range stmt.Columns {
		s, ok := c.(ast.SelectStatement)
		if !ok {
			continue
		}
		if len(s.Columns) != 1 {
			list = append(list, countMultipleFields())
		}
	}
	return nil
}

func checkColumnUsedInGroup(stmt ast.SelectStatement) []LintMessage {
	if len(stmt.Groups) == 0 {
		return nil
	}
	var (
		groups = ast.GetNamesFromStmt(stmt.Groups)
		list   []LintMessage
	)
	for _, c := range stmt.Columns {
		switch c := c.(type) {
		case ast.Alias:
			call, ok := c.Statement.(ast.Call)
			if ok {
				if ok = call.IsAggregate(); !ok {
					list = append(list, aggregateFunctionExpected(call.GetIdent()))
				}
			}
			name, ok := c.Statement.(ast.Name)
			if !ok {
				list = append(list, unexpectedExprType("", "GROUP BY"))
			}
			if ok = slices.Contains(groups, name.Ident()); !ok {
				list = append(list, fieldNotInGroup(name.Ident()))
			}
		case ast.Call:
			if ok := c.IsAggregate(); !ok {
				list = append(list, aggregateFunctionExpected(c.GetIdent()))
			}
		case ast.Name:
			ok := slices.Contains(groups, c.Name())
			if !ok {
				list = append(list, fieldNotInGroup(c.Name()))
			}
		default:
			list = append(list, unexpectedExprType("", "GROUP BY"))
		}
	}
	return list
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
