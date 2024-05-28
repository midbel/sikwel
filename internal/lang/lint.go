package lang

import (
	"errors"
	"fmt"
	"io"
	"slices"
)

type Level int

func (e Level) String() string {
	switch e {
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "other"
	}
}

const (
	Default Level = iota
	Debug
	Info
	Warning
	Error
)

type MultiError []error

func (_ MultiError) Error() string {
	return "multiple error detected"
}

type LintMessage struct {
	Severity Level
	Rule     string
	Message  string
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
	p, err := NewParser(r)
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

func (i Linter) LintStatement(stmt Statement) ([]LintMessage, error) {
	var (
		list []LintMessage
		err  error
	)
	switch stmt := stmt.(type) {
	case CreateViewStatement:
	case WithStatement:
		list, err = i.lintWith(stmt)
	case CteStatement:
		list, err = i.lintCte(stmt)
	case SelectStatement:
		list, err = i.lintSelect(stmt)
	case InsertStatement:
		list, err = i.lintInsert(stmt)
	case UpdateStatement:
		list, err = i.lintUpdate(stmt)
	case DeleteStatement:
		list, err = i.lintDelete(stmt)
	case MergeStatement:
		list, err = i.lintMerge(stmt)
	case UnionStatement:
		list, err = i.lintUnion(stmt)
	case IntersectStatement:
		list, err = i.lintIntersect(stmt)
	case ExceptStatement:
		list, err = i.lintExcept(stmt)
	case List:
		list, err = i.lintList(stmt)
	case Unary:
		list, err = i.LintStatement(stmt.Right)
	case Binary:
		l1, err1 := i.LintStatement(stmt.Left)
		l2, err2 := i.LintStatement(stmt.Right)
		list = append(list, l1...)
		list = append(list, l2...)
		if err1 != nil {
			err = err1
		}
		if err2 != nil {
			err = err2
		}
	case Between:
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
	case In:
		list, err = i.LintStatement(stmt.Value)
	case Is:
		list, err = i.LintStatement(stmt.Value)
	case Not:
		list, err = i.LintStatement(stmt.Statement)
	case Exists:
		list, err = i.LintStatement(stmt.Statement)
	case All:
		list, err = i.LintStatement(stmt.Statement)
	case Any:
		list, err = i.LintStatement(stmt.Statement)
	default:
	}
	return list, err
}

func (i Linter) lintList(stmt List) ([]LintMessage, error) {
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

func (i Linter) lintInsert(stmt InsertStatement) ([]LintMessage, error) {
	var (
		list  []LintMessage
		count = len(stmt.Columns)
	)
	if count > 0 {
		switch stmt := stmt.Values.(type) {
		case SelectStatement:
			if stmt.ColumnsCount() != count {
				list = append(list, columnsCountMismatched())
			}
		case List:
			for i := range stmt.Values {
				vs, ok := stmt.Values[i].(List)
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

func (i Linter) lintMerge(stmt MergeStatement) ([]LintMessage, error) {
	return nil, nil
}

func (i Linter) lintUpdate(stmt UpdateStatement) ([]LintMessage, error) {
	return nil, nil
}

func (i Linter) lintDelete(stmt DeleteStatement) ([]LintMessage, error) {
	return nil, nil
}

func (i Linter) lintUnion(stmt UnionStatement) ([]LintMessage, error) {
	s1, ok := stmt.Left.(SelectStatement)
	if !ok {
		return nil, fmt.Errorf("select expected on left side of union")
	}
	s2, ok := stmt.Right.(SelectStatement)
	if !ok {
		return nil, fmt.Errorf("select expected on right side of union")
	}
	return i.lintSets(s1, s2)
}

func (i Linter) lintIntersect(stmt IntersectStatement) ([]LintMessage, error) {
	s1, ok := stmt.Left.(SelectStatement)
	if !ok {
		return nil, fmt.Errorf("select expected on left side of union")
	}
	s2, ok := stmt.Right.(SelectStatement)
	if !ok {
		return nil, fmt.Errorf("select expected on right side of union")
	}
	return i.lintSets(s1, s2)
}

func (i Linter) lintExcept(stmt ExceptStatement) ([]LintMessage, error) {
	s1, ok := stmt.Left.(SelectStatement)
	if !ok {
		return nil, fmt.Errorf("select expected on left side of union")
	}
	s2, ok := stmt.Right.(SelectStatement)
	if !ok {
		return nil, fmt.Errorf("select expected on right side of union")
	}
	return i.lintSets(s1, s2)
}

func (i Linter) lintSets(s1, s2 SelectStatement) ([]LintMessage, error) {
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

func (i Linter) lintCte(stmt CteStatement) ([]LintMessage, error) {
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

func (i Linter) lintWith(stmt WithStatement) ([]LintMessage, error) {
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

func (i Linter) lintSelect(stmt SelectStatement) ([]LintMessage, error) {
	// check subqueries
	var (
		list    []LintMessage
		queries []Statement
	)
	queries = slices.Concat(slices.Clone(stmt.Columns), slices.Clone(stmt.Tables))
	for _, c := range queries {
		if c, ok := c.(SelectStatement); ok {
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
	return list, nil
}

func checkColumnUsedInGroup(stmt SelectStatement) []LintMessage {
	if len(stmt.Groups) == 0 {
		return nil
	}
	var (
		groups = getNamesFromStmt(stmt.Groups)
		list   []LintMessage
	)
	for _, c := range stmt.Columns {
		switch c := c.(type) {
		case Alias:
			call, ok := c.Statement.(Call)
			if ok {
				if ok = call.IsAggregate(); !ok {
					list = append(list, aggregateFunctionExpected(call.GetIdent()))
				}
			}
			name, ok := c.Statement.(Name)
			if !ok {
				list = append(list, unexpectedExprType("", "GROUP BY"))
			}
			if ok = slices.Contains(groups, name.Ident()); !ok {
				list = append(list, fieldNotInGroup(name.Ident()))
			}
		case Call:
			if ok := c.IsAggregate(); !ok {
				list = append(list, aggregateFunctionExpected(c.GetIdent()))
			}
		case Name:
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

func checkUniqueAlias(stmt SelectStatement) []LintMessage {
	var (
		columns  = getAliasFromStmt(stmt.Columns)
		tables   = getAliasFromStmt(stmt.Tables)
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

func checkUndefinedAlias(stmt SelectStatement) []LintMessage {
	var (
		alias  = getAliasFromStmt(stmt.Tables)
		names  = getNamesFromStmt(stmt.Tables)
		values = slices.Concat(alias, names)
		list   []LintMessage
	)
	for _, c := range stmt.Columns {
		if a, ok := c.(Alias); ok {
			c = a.Statement
		}
		n, ok := c.(Name)
		if !ok {
			continue
		}
		if schema := n.Schema(); schema != "" && !slices.Contains(values, schema) {
			list = append(list, undefinedAlias(schema))
		}
	}
	return list
}

func checkMissingAlias(stmt SelectStatement) []LintMessage {
	var list []LintMessage
	for _, s := range stmt.Columns {
		if _, ok := s.(SelectStatement); ok {
			list = append(list, missingAlias())
		}
	}
	for _, s := range stmt.Tables {
		switch s := s.(type) {
		case SelectStatement:
			list = append(list, missingAlias())
		case Join:
			if _, ok := s.Table.(SelectStatement); ok {
				list = append(list, missingAlias())
			}
		default:
		}
	}
	return nil
}

func checkAliasUsedInWhere(stmt SelectStatement) []LintMessage {
	var (
		names = getNamesFromStmt([]Statement{stmt.Where, stmt.Having})
		list  []LintMessage
	)
	for _, a := range getAliasFromStmt(stmt.Columns) {
		ok := slices.Contains(names, a)
		if ok {
			list = append(list, unexpectedAlias(a))
		}
	}
	return list
}

func columnsCountMismatched() LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  "columns count mismatched",
		Rule:     "count.invalid",
	}
}

func fieldNotInGroup(field string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: field should be used in a 'group by' clause or with an aggregate function", field),
		Rule:     "expression.group",
	}
}

func aggregateFunctionExpected(ident string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: aggregate function expected"),
		Rule:     "aggregate.function",
	}
}

func unexpectedExprType(field, ctx string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: unexpected expression type", ctx),
		Rule:     "expression.invalid",
	}
}

func unexpectedAlias(alias string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: alias found in predicate", alias),
		Rule:     "alias.unexpected",
	}
}

func undefinedAlias(alias string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: alias not defined", alias),
		Rule:     "alias.missing",
	}
}

func missingAlias() LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  "expression needs to be used with an alias",
		Rule:     "alias.missing",
	}
}

func duplicatedAlias(alias string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: alias already defined", alias),
		Rule:     "alias.duplicate",
	}
}
