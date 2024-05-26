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
	case IntersectStatement:
	case ExceptStatement:
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
	case List:
		for j := range stmt.Values {
			others, err := i.LintStatement(stmt.Values[j])
			if err != nil {
				return nil, err
			}
			list = append(list, others...)
		}
	case All:
		list, err = i.LintStatement(stmt.Statement)
	case Any:
		list, err = i.LintStatement(stmt.Statement)
	default:
	}
	return list, err
}

func (i Linter) lintInsert(stmt InsertStatement) ([]LintMessage, error) {
	return nil, nil
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
					list = append(list, notAggregateFunction(call.GetIdent()))
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
				list = append(list, notAggregateFunction(c.GetIdent()))
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
		columns = getAliasFromStmt(stmt.Columns)
		tables  = getAliasFromStmt(stmt.Tables)
		list    []LintMessage
	)
	contains := func(list []string, str string) bool {
		return slices.Contains(list, str)
	}
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
	return nil
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
	for _, a := range stmt.GetAlias() {
		ok := slices.Contains(names, a)
		if ok {
			list = append(list, aliasFoundInWhere(a))
		}
	}
	return list
}

func missingAlias() LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  "missing alias",
		Rule:     "alias.missing",
	}
}

func duplicatedAlias(alias string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s: duplicate alias found", alias),
		Rule:     "alias.duplicate",
	}
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
		Message:  fmt.Sprintf("field %s not used in group by closed nor in an aggregate function"),
		Rule:     "expression.group",
	}
}

func notAggregateFunction(ident string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s not an aggregation function"),
		Rule:     "aggregate.function",
	}
}

func unexpectedExprType(field, ctx string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("unexpected expression type in %s", ctx),
		Rule:     "expression.invalid",
	}
}

func aliasFoundInWhere(field string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("alias %s found in predicate", field),
		Rule:     "alias.unexpected",
	}
}
