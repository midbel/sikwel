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

type LintMessage struct {
	Severity Level
	Rule     string
	Message  string
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
	case WithStatement:
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

func (i Linter) lintSelect(stmt SelectStatement) ([]LintMessage, error) {
	// check subqueries
	var list []LintMessage
	if err := checkAliasUsedInWhere(stmt); err != nil {
		msg := LintMessage{
			Severity: Error,
			Message:  err.Error(),
			Rule:     "alias-use-where",
		}
		list = append(list, msg)
	}
	if err := checkColumnUsedInGroup(stmt); err != nil {
		msg := LintMessage{
			Severity: Error,
			Message:  err.Error(),
			Rule:     "column-in-group",
		}
		list = append(list, msg)
	}
	return list, nil
}

func checkColumnUsedInGroup(stmt SelectStatement) error {
	if len(stmt.Groups) == 0 {
		return nil
	}
	var (
		groups = getNamesFromStmt(stmt.Groups)
		err    error
	)
	for _, c := range stmt.Columns {
		switch c := c.(type) {
		case Alias:
			call, ok := c.Statement.(Call)
			if ok {
				if ok = call.IsAggregate(); !ok {
					err = fmt.Errorf("%s not an aggregate function", call.GetIdent())
				}
			}
			name, ok := c.Statement.(Name)
			if !ok {
				err = fmt.Errorf("unexpected expression type")
			}
			if ok = slices.Contains(groups, name.Ident()); !ok {
				err = fmt.Errorf("field %s should be used in group by or with aggregate function", name.Name())
			}
		case Call:
			if ok := c.IsAggregate(); !ok {
				err = fmt.Errorf("%s not an aggregate function", c.GetIdent())
			}
		case Name:
			ok := slices.Contains(groups, c.Name())
			if !ok {
				err = fmt.Errorf("field %s should be used in group by or with aggregate function", c.Name())
			}
		default:
			err = fmt.Errorf("unexpected expression type")
		}
		if err != nil {
			break
		}
	}
	return err
}

func checkAliasUsedInWhere(stmt SelectStatement) error {
	names := getNamesFromStmt([]Statement{stmt.Where})
	for _, a := range stmt.GetAlias() {
		ok := slices.Contains(names, a)
		if ok {
			return fmt.Errorf("alias found in where clause")
		}
	}
	return nil
}
