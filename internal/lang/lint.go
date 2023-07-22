package lang

import (
	"fmt"
	"io"
)

type Linter struct {
	parser *Parser
}

func Lint(r io.Reader) (*Linter, error) {
	p, err := NewParser(r)
	if err != nil {
		return nil, err
	}
	i := Linter{
		parser: p,
	}
	return &i, nil
}

func (i *Linter) Lint() ([]error, error) {
	stmt, err := i.parser.Parse()
	if err != nil {
		return nil, err
	}
	return i.LintStatement(stmt)
}

func (i *Linter) LintStatement(stmt Statement) ([]error, error) {
	var errs []error
	switch s := stmt.(type) {
	case SelectStatement:
		errs = i.lintSelect(s)
	case InsertStatement:
		errs = i.lintInsert(s)
	case DeleteStatement:
		errs = i.lintDelete(s)
	case UpdateStatement:
		errs = i.lintUpdate(s)
	case WithStatement:
		errs = i.lintWith(s)
	default:
		return nil, fmt.Errorf("unsupport statement type %T", stmt)
	}
	return errs, nil
}

func (i *Linter) lintSelect(stmt SelectStatement) []error {
	return nil
}

func (i *Linter) lintInsert(stmt InsertStatement) []error {
	return nil
}

func (i *Linter) lintUpdate(stmt UpdateStatement) []error {
	return nil
}

func (i *Linter) lintDelete(stmt DeleteStatement) []error {
	return nil
}

func (i *Linter) lintWith(stmt WithStatement) []error {
	return nil
}
