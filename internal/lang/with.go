package lang

import (
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
	return stmt, wrapError("with", err)
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
	// w.Enter()
	// defer w.Leave()

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
	w.Leave()
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
