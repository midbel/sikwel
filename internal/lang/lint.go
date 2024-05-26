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
	case UpdateStatement:
	case DeleteStatement:
	case MergeStatement:
	default:
		return nil, fmt.Errorf("statement type not supported for linting")
	}
	return list, err
}

func (i Linter) lintCte(stmt CteStatement) ([]LintMessage, error) {
	var list []LintMessage
	if z := len(stmt.Columns); z != 0 {
		if c, ok := stmt.Statement.(interface{ ColumnsCount() int }); ok {
			n := c.ColumnsCount()
			if n != z {

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
		list []LintMessage
		tmp  []LintMessage
	)
	tmp = checkAliasUsedInWhere(stmt)
	list = append(list, tmp...)

	tmp = checkColumnUsedInGroup(stmt)
	list = append(list, tmp...)
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

func checkAliasUsedInWhere(stmt SelectStatement) []LintMessage {
	var (
		names = getNamesFromStmt([]Statement{stmt.Where})
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

func fieldNotInGroup(field string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("field %s not used in group by closed nor in an aggregate function"),
		Rule:     "field-not-grouped",
	}
}

func notAggregateFunction(ident string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("%s not an aggregation function"),
		Rule:     "aggregate-function",
	}
}

func unexpectedExprType(field, ctx string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("unexpected expression type in %s", ctx),
		Rule:     "unexpected-expression",
	}
}

func aliasFoundInWhere(field string) LintMessage {
	return LintMessage{
		Severity: Error,
		Message:  fmt.Sprintf("alias %s found in predicate", field),
		Rule:     "alias-in-predicate",
	}
}
