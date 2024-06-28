package lint

import (
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/lang/parser"
)

type MultiError []error

func (_ MultiError) Error() string {
	return "multiple error detected"
}

func (e LintMessage) Error() string {
	return ""
}

type Linter struct {
	MinLevel Level
}

func NewLinter() *Linter {
	return &Linter{}
}

func (i Linter) Lint(r io.Reader) ([]LintMessage, error) {
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
	}
	return list, nil
}

func (i Linter) LintStatement(stmt ast.Statement) ([]LintMessage, error) {
	var (
		list []LintMessage
		err  error
	)
	switch stmt := stmt.(type) {
	case ast.CreateViewStatement:
	case ast.WithStatement:
		list, err = i.lintWith(stmt)
	case ast.CteStatement:
		list, err = i.lintCte(stmt)
	case ast.ValuesStatement:
		list, err = i.lintValues(stmt)
	case ast.SelectStatement:
		list, err = i.lintSelect(stmt)
	case ast.InsertStatement:
		list, err = i.lintInsert(stmt)
	case ast.UpdateStatement:
		list, err = i.lintUpdate(stmt)
	case ast.DeleteStatement:
		list, err = i.lintDelete(stmt)
	case ast.MergeStatement:
		list, err = i.lintMerge(stmt)
	case ast.UnionStatement:
		list, err = i.lintUnion(stmt)
	case ast.IntersectStatement:
		list, err = i.lintIntersect(stmt)
	case ast.ExceptStatement:
		list, err = i.lintExcept(stmt)
	case ast.List:
		list, err = i.lintList(stmt)
	case ast.Unary:
		list, err = i.LintStatement(stmt.Right)
	case ast.Binary:
		list, err = i.lintBinary(stmt)
	case ast.Between:
		l1, err1 := i.LintStatement(stmt.Lower)
		l2, err2 := i.LintStatement(stmt.Upper)
		list = append(list, l1...)
		list = append(list, l2...)
		if err1 != nil {
			err = err1
		}
		if err2 != nil {
			err = err2
		}
	case ast.In:
		list, err = i.lintIn(stmt)
	case ast.Is:
		list, err = i.LintStatement(stmt.Value)
	case ast.Not:
		list, err = i.LintStatement(stmt.Statement)
	case ast.Exists:
		list, err = i.LintStatement(stmt.Statement)
	case ast.All:
		list, err = i.LintStatement(stmt.Statement)
	case ast.Any:
		list, err = i.LintStatement(stmt.Statement)
	case ast.Group:
		list, err = i.LintStatement(stmt.Statement)
	default:
	}
	return list, err
}

func (i Linter) lintIn(stmt ast.In) ([]LintMessage, error) {
	if vs, ok := stmt.Value.(ast.List); ok && len(vs.Values) == 1 {
		return []LintMessage{oneValueWithInPredicate()}, nil
	}
	return i.LintStatement(stmt.Value)
}

func (i Linter) lintBinary(stmt ast.Binary) ([]LintMessage, error) {
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

func (i Linter) lintList(stmt ast.List) ([]LintMessage, error) {
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

func (i Linter) lintInsert(stmt ast.InsertStatement) ([]LintMessage, error) {
	var (
		list  []LintMessage
		count = len(stmt.Columns)
	)
	if count > 0 {
		switch stmt := stmt.Values.(type) {
		case ast.SelectStatement:
			if stmt.ColumnsCount() != count {
				list = append(list, columnsCountMismatched())
			}
		case ast.List:
			for i := range stmt.Values {
				vs, ok := stmt.Values[i].(ast.List)
				if !ok {
					return nil, fmt.Errorf("values expected with insert statement")
				}
				if len(vs.Values) != count {
					list = append(list, columnsCountMismatched())
				}
			}
		default:
			return nil, fmt.Errorf("select/values expected with insert statement")
		}
	}
	others, err := i.LintStatement(stmt.Values)
	if err != nil {
		return nil, err
	}
	list = append(list, others...)
	return list, nil
}

func (i Linter) lintMerge(stmt ast.MergeStatement) ([]LintMessage, error) {
	return nil, nil
}

func (i Linter) lintUpdate(stmt ast.UpdateStatement) ([]LintMessage, error) {
	return nil, nil
}

func (i Linter) lintDelete(stmt ast.DeleteStatement) ([]LintMessage, error) {
	return nil, nil
}

func (i Linter) lintUnion(stmt ast.UnionStatement) ([]LintMessage, error) {
	s1, ok := stmt.Left.(ast.SelectStatement)
	if !ok {
		return nil, fmt.Errorf("select expected on left side of union")
	}
	s2, ok := stmt.Right.(ast.SelectStatement)
	if !ok {
		return nil, fmt.Errorf("select expected on right side of union")
	}
	return i.lintSets(s1, s2)
}

func (i Linter) lintIntersect(stmt ast.IntersectStatement) ([]LintMessage, error) {
	s1, ok := stmt.Left.(ast.SelectStatement)
	if !ok {
		return nil, fmt.Errorf("select expected on left side of union")
	}
	s2, ok := stmt.Right.(ast.SelectStatement)
	if !ok {
		return nil, fmt.Errorf("select expected on right side of union")
	}
	return i.lintSets(s1, s2)
}

func (i Linter) lintExcept(stmt ast.ExceptStatement) ([]LintMessage, error) {
	s1, ok := stmt.Left.(ast.SelectStatement)
	if !ok {
		return nil, fmt.Errorf("select expected on left side of union")
	}
	s2, ok := stmt.Right.(ast.SelectStatement)
	if !ok {
		return nil, fmt.Errorf("select expected on right side of union")
	}
	return i.lintSets(s1, s2)
}

func (i Linter) lintSets(s1, s2 ast.SelectStatement) ([]LintMessage, error) {
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

func (i Linter) lintCte(stmt ast.CteStatement) ([]LintMessage, error) {
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

func (i Linter) lintWith(stmt ast.WithStatement) ([]LintMessage, error) {
	var list []LintMessage
	for _, q := range stmt.Queries {
		others, err := i.LintStatement(q)
		if err != nil {
			return nil, err
		}
		list = append(list, others...)
	}
	others, err := i.LintStatement(stmt.Statement)
	if err != nil {
		return nil, err
	}
	list = append(list, others...)
	return list, nil
}

func (i Linter) lintValues(stmt ast.ValuesStatement) ([]LintMessage, error) {
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

func (i Linter) lintSelect(stmt ast.SelectStatement) ([]LintMessage, error) {
	// check subqueries
	var (
		list    []LintMessage
		queries []ast.Statement
	)
	queries = slices.Concat(slices.Clone(stmt.Columns), slices.Clone(stmt.Tables))
	for _, c := range queries {
		if c, ok := c.(ast.SelectStatement); ok {
			others, err := i.lintSelect(c)
			if err != nil {
				return nil, err
			}
			list = append(list, others...)
		}
	}
	list = append(list, checkUniqueAlias(stmt)...)
	list = append(list, checkMissingAlias(stmt)...)
	list = append(list, checkUndefinedAlias(stmt)...)
	list = append(list, checkAliasUsedInWhere(stmt)...)
	list = append(list, checkColumnUsedInGroup(stmt)...)
	ls, err := i.LintStatement(stmt.Where)
	if err != nil {
		return nil, err
	}
	list = append(list, ls...)
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

func checkUniqueAlias(stmt ast.SelectStatement) []LintMessage {
	var (
		columns  = ast.GetAliasFromStmt(stmt.Columns)
		tables   = ast.GetAliasFromStmt(stmt.Tables)
		contains = func(list []string, str string) bool {
			return slices.Contains(list, str)
		}
		list []LintMessage
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
	return nil
}

func checkUndefinedAlias(stmt ast.SelectStatement) []LintMessage {
	var (
		alias  = ast.GetAliasFromStmt(stmt.Tables)
		names  = ast.GetNamesFromStmt(stmt.Tables)
		values = slices.Concat(alias, names)
		list   []LintMessage
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
	return list
}

func checkMissingAlias(stmt ast.SelectStatement) []LintMessage {
	var list []LintMessage
	for _, s := range stmt.Columns {
		if _, ok := s.(ast.SelectStatement); ok {
			list = append(list, missingAlias())
		}
	}
	for _, s := range stmt.Tables {
		switch s := s.(type) {
		case ast.SelectStatement:
			list = append(list, missingAlias())
		case ast.Join:
			if _, ok := s.Table.(ast.SelectStatement); ok {
				list = append(list, missingAlias())
			}
		default:
		}
	}
	return nil
}

func checkAliasUsedInWhere(stmt ast.SelectStatement) []LintMessage {
	var (
		names = ast.GetNamesFromStmt([]ast.Statement{stmt.Where, stmt.Having})
		list  []LintMessage
	)
	for _, a := range ast.GetAliasFromStmt(stmt.Columns) {
		ok := slices.Contains(names, a)
		if ok {
			list = append(list, unexpectedAlias(a))
		}
	}
	return list
}
