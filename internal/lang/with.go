package lang

import (
	"fmt"
	"strings"
)

func (p *Parser) parseWith() (Statement, error) {
	p.Next()
	var (
		stmt WithStatement
		err  error
	)
	if p.IsKeyword("RECURSIVE") {
		stmt.Recursive = true
		p.Next()
	}
	for !p.Done() && !p.Is(Keyword) {
		cte, err := p.parseSubquery()
		if err = wrapError("subquery", err); err != nil {
			return nil, err
		}
		stmt.Queries = append(stmt.Queries, cte)
		if err = p.EnsureEnd("with", Comma, Keyword); err != nil {
			return nil, err
		}
	}
	stmt.Statement, err = p.ParseStatement()
	if p.inlineCte {
		return p.inlineWith(stmt)
	}
	return stmt, wrapError("with", err)
}

func (p *Parser) inlineWith(with WithStatement) (Statement, error) {
	list := make(map[string]Statement)
	for _, c := range with.Queries {
		q, ok := c.(CteStatement)
		if !ok {
			return nil, fmt.Errorf("unexpected cte type")
		}
		list[q.Ident] = q.Statement
	}
	var replace func(Statement, bool) Statement

	replace = func(stmt Statement, withAlias bool) Statement {
		switch s := stmt.(type) {
		case Name:
			if len(s.Parts) != 1 {
				break
			}
			n := s.Parts[0]
			if q, ok := list[n]; ok {
				if withAlias {
					q = Alias{
						Statement: q,
						Alias:     n,
					}
				}
				return q
			}
			return stmt
		case Join:
			return replace(s.Table, withAlias)
		case Alias:
		default:
		}
		return stmt
	}
	switch stmt := with.Statement.(type) {
	case SelectStatement:
		for i := range stmt.Columns {
			stmt.Columns[i] = replace(stmt.Columns[i], false)
		}
		for i := range stmt.Tables {
			stmt.Tables[i] = replace(stmt.Tables[i], true)
		}
		stmt.Where = replace(stmt.Where, false)
		return stmt, nil
	case UpdateStatement:
		return stmt, nil
	case InsertStatement:
		return stmt, nil
	case DeleteStatement:
		return stmt, nil
	default:
		return nil, fmt.Errorf("unsupported query type")
	}
}

func (p *Parser) parseSubquery() (Statement, error) {
	var (
		cte CteStatement
		err error
	)
	if !p.Is(Ident) {
		return nil, p.Unexpected("subquery")
	}
	cte.Ident = p.GetCurrLiteral()
	p.Next()

	cte.Columns, err = p.parseColumnsList()
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("AS") {
		return nil, p.Unexpected("subquery")
	}
	p.Next()
	if p.IsKeyword("MATERIALIZED") {
		p.Next()
		cte.Materialized = MaterializedCte
	} else if p.IsKeyword("NOT") {
		p.Next()
		if !p.IsKeyword("MATERIALIZED") {
			return nil, p.Unexpected("subquery")
		}
		p.Next()
		cte.Materialized = NotMaterializedCte
	}
	if !p.Is(Lparen) {
		return nil, p.Unexpected("subquery")
	}
	p.Next()

	cte.Statement, err = p.ParseStatement()
	if err = wrapError("subquery", err); err != nil {
		return nil, err
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("subquery")
	}
	p.Next()
	return cte, nil
}

func (w *Writer) FormatWith(stmt WithStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Recursive {
		w.WriteBlank()
		w.WriteString("RECURSIVE")
	}
	w.WriteNL()

	for i, q := range stmt.Queries {
		if i > 0 {
			w.WriteString(",")
			w.WriteNL()
		}
		if err := w.FormatStatement(q); err != nil {
			return err
		}
	}
	w.WriteNL()
	return w.FormatStatement(stmt.Statement)
}

func (w *Writer) FormatCte(stmt CteStatement) error {
	w.Enter()
	defer w.Leave()

	w.WritePrefix()
	ident := stmt.Ident
	if w.Upperize {
		ident = strings.ToUpper(ident)
	}
	w.WriteString(ident)
	if len(stmt.Columns) == 0 && w.UseNames {
		if q, ok := stmt.Statement.(interface{ GetNames() []string }); ok {
			stmt.Columns = q.GetNames()
		}
	}
	if len(stmt.Columns) > 0 {
		w.WriteString("(")
		for i, s := range stmt.Columns {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			if w.Upperize {
				s = strings.ToUpper(s)
			}
			w.WriteString(s)
		}
		w.WriteString(")")
	}
	w.WriteBlank()
	w.WriteKeyword("AS")
	w.WriteBlank()
	w.WriteString("(")
	w.WriteNL()

	if err := w.FormatStatement(stmt.Statement); err != nil {
		return err
	}
	w.WriteNL()
	w.WriteString(")")
	return nil
}
