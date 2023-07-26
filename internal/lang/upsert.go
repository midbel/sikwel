package lang

import (
	"fmt"
)

func (p *Parser) parseDelete() (Statement, error) {
	p.Next()
	var (
		stmt DeleteStatement
		err  error
	)
	if !p.Is(Ident) {
		return nil, p.Unexpected("delete")
	}
	stmt.Table = p.curr.Literal
	p.Next()

	if stmt.Where, err = p.ParseWhere(); err != nil {
		return nil, wrapError("delete", err)
	}
	if stmt.Return, err = p.ParseReturning(); err != nil {
		return nil, wrapError("delete", err)
	}
	return stmt, nil
}

func (p *Parser) ParseUpdate() (Statement, error) {
	p.Next()
	var (
		stmt UpdateStatement
		err  error
	)
	stmt.Table, err = p.ParseIdent()
	if err != nil {
		return nil, err
	}

	if !p.IsKeyword("SET") {
		return nil, p.Unexpected("update")
	}
	p.Next()

	if stmt.List, err = p.ParseUpdateList(); err != nil {
		return nil, err
	}

	if p.IsKeyword("FROM") {
		_, err = p.ParseFrom()
		if err != nil {
			return nil, err
		}
	}
	if stmt.Where, err = p.ParseWhere(); err != nil {
		return nil, err
	}
	stmt.Return, err = p.ParseReturning()
	return stmt, wrapError("update", err)
}

func (p *Parser) ParseUpdateList() ([]Statement, error) {
	var list []Statement
	for !p.Done() && !p.Is(EOL) && !p.IsKeyword("WHERE") && !p.IsKeyword("FROM") && !p.IsKeyword("RETURNING") {
		stmt, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}
		if p.Is(EOL) {
			break
		}
		if err := p.EnsureEnd("update", Comma, Keyword); err != nil {
			return nil, err
		}
		list = append(list, stmt)
	}
	return list, nil
}

func (p *Parser) parseAssignment() (Statement, error) {
	var (
		ass Assignment
		err error
	)
	switch {
	case p.Is(Ident):
		ass.Field, err = p.ParseIdent()
		if err != nil {
			return nil, err
		}
	case p.Is(Lparen):
		p.Next()
		var list List
		for !p.Done() && !p.Is(Rparen) {
			stmt, err := p.ParseIdent()
			if err != nil {
				return nil, err
			}
			list.Values = append(list.Values, stmt)
			if err = p.EnsureEnd("update", Comma, Rparen); err != nil {
				return nil, err
			}
		}
		if !p.Is(Rparen) {
			return nil, err
		}
		p.Next()
		ass.Field = list
	default:
		return nil, p.Unexpected("update")
	}
	if !p.Is(Eq) {
		return nil, p.Unexpected("update")
	}
	p.Next()
	if p.Is(Lparen) {
		p.Next()
		var list List
		for !p.Done() && !p.Is(Rparen) {
			expr, err := p.StartExpression()
			if err != nil {
				return nil, err
			}
			if err = p.EnsureEnd("update", Comma, Rparen); err != nil {
				return nil, err
			}
			list.Values = append(list.Values, expr)
		}
		if !p.Is(Rparen) {
			return nil, p.Unexpected("update")
		}
		p.Next()
	} else {
		ass.Value, err = p.StartExpression()
		if err != nil {
			return nil, p.Unexpected("update")
		}
	}
	return ass, nil
}

func (p *Parser) ParseInsert() (Statement, error) {
	p.Next()
	var (
		stmt InsertStatement
		err  error
	)
	stmt.Table, err = p.ParseIdent()
	if err != nil {
		return nil, err
	}

	stmt.Columns, err = p.parseColumnsList()
	if err = wrapError("insert", err); err != nil {
		return nil, err
	}

	switch {
	case p.IsKeyword("SELECT"):
		stmt.Values, err = p.ParseStatement()
	case p.IsKeyword("VALUES"):
		p.Next()
		var all List
		for !p.Done() && !p.IsKeyword("RETURNING") && !p.IsKeyword("ON CONFLICT") && !p.Is(EOL) {
			if !p.Is(Lparen) {
				return nil, p.Unexpected("values")
			}
			p.Next()

			list, err := p.parseListValues()
			if err = wrapError("insert", err); err != nil {
				return nil, err
			}
			all.Values = append(all.Values, list)

			switch {
			case p.Is(Comma):
				p.Next()
			case p.Is(EOL):
			case p.Is(Keyword):
			default:
				return nil, p.Unexpected("values")
			}
		}
		stmt.Values = all
	default:
		return nil, p.Unexpected("values")
	}
	if err = wrapError("insert", err); err != nil {
		return nil, err
	}
	if stmt.Upsert, err = p.ParseUpsert(); err != nil {
		return nil, err
	}
	stmt.Return, err = p.ParseReturning()
	return stmt, wrapError("insert", err)
}

func (p *Parser) ParseUpsert() (Statement, error) {
	if !p.IsKeyword("ON CONFLICT") {
		return nil, nil
	}
	p.Next()

	var (
		stmt UpsertStatement
		err  error
	)

	if !p.IsKeyword("DO") {
		stmt.Columns, err = p.parseColumnsList()
		if err != nil {
			return nil, err
		}
	}
	if !p.IsKeyword("DO") {
		return nil, p.Unexpected("upsert")
	}
	p.Next()
	if p.IsKeyword("NOTHING") {
		p.Next()
		return stmt, nil
	}
	if !p.IsKeyword("UPDATE") {
		return nil, p.Unexpected("upsert")
	}
	p.Next()
	if !p.IsKeyword("SET") {
		return nil, p.Unexpected("upsert")
	}
	p.Next()
	if stmt.List, err = p.ParseUpsertList(); err != nil {
		return nil, err
	}
	stmt.Where, err = p.ParseWhere()
	return stmt, wrapError("upsert", err)
}

func (p *Parser) ParseUpsertList() ([]Statement, error) {
	var list []Statement
	for !p.Done() && !p.Is(EOL) && !p.IsKeyword("WHERE") && !p.IsKeyword("RETURNING") {
		stmt, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}
		if p.Is(EOL) {
			break
		}
		if err := p.EnsureEnd("update", Comma, Keyword); err != nil {
			return nil, err
		}
		list = append(list, stmt)
	}
	return list, nil
}

func (p *Parser) parseListValues() (Statement, error) {
	var list List
	for !p.Done() && !p.Is(Rparen) {
		expr, err := p.StartExpression()
		if err = wrapError("values", err); err != nil {
			return nil, err
		}
		if err := p.EnsureEnd("values", Comma, Rparen); err != nil {
			return nil, err
		}
		list.Values = append(list.Values, expr)
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("values")
	}
	p.Next()
	return list, nil
}

func (w *Writer) FormatDelete(stmt DeleteStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	w.WriteString(stmt.Table)
	if stmt.Where != nil {
		w.WriteNL()
		if err := w.FormatWhere(stmt.Where); err != nil {
			return err
		}
	}
	if stmt.Return != nil {
		w.WriteNL()
		if err := w.FormatReturning(stmt.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatUpdate(stmt UpdateStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	switch stmt := stmt.Table.(type) {
	case Name:
		w.FormatName(stmt)
	case Alias:
		if err := w.FormatAlias(stmt); err != nil {
			return err
		}
	default:
		return w.CanNotUse("update", stmt)
	}
	w.WriteBlank()
	w.WriteKeyword("SET")
	w.WriteNL()

	if err := w.FormatAssignment(stmt.List); err != nil {
		return err
	}

	if len(stmt.Tables) > 0 {
		w.WriteNL()
		if err := w.FormatFrom(stmt.Tables); err != nil {
			return err
		}
	}
	if stmt.Where != nil {
		w.WriteNL()
		if err := w.FormatWhere(stmt.Where); err != nil {
			return err
		}
	}
	if stmt.Return != nil {
		w.WriteNL()
		if err := w.FormatReturning(stmt.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatInsert(stmt InsertStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Table, false); err != nil {
		return err
	}
	if len(stmt.Columns) > 0 {
		w.WriteBlank()
		w.WriteString("(")
		for i, c := range stmt.Columns {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteString(c)
		}
		w.WriteString(")")
	}
	w.WriteBlank()
	if err := w.FormatInsertValues(stmt.Values); err != nil {
		return err
	}
	if stmt.Upsert != nil {
		w.WriteNL()
		if err := w.FormatUpsert(stmt.Upsert); err != nil {
			return err
		}
	}
	if stmt.Return != nil {
		w.WriteNL()
		if err := w.FormatReturning(stmt.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatInsertValues(values Statement) error {
	if values == nil {
		return nil
	}
	var err error
	switch stmt := values.(type) {
	case List:
		w.WriteKeyword("VALUES")
		w.WriteNL()

		w.Enter()
		defer w.Leave()
		for i, v := range stmt.Values {
			if i > 0 {
				w.WriteString(",")
				w.WriteNL()
			}
			w.WritePrefix()
			if err = w.FormatExpr(v, false); err != nil {
				break
			}
		}
	case SelectStatement:
		w.WriteNL()
		err = w.FormatSelect(stmt)
	default:
		err = fmt.Errorf("values: unexpected statement type(%T)", values)
	}
	return err
}

func (w *Writer) FormatUpsert(stmt Statement) error {
	if stmt == nil {
		return nil
	}
	upsert, ok := stmt.(UpsertStatement)
	if !ok {
		return w.CanNotUse("insert(upsert)", stmt)
	}
	w.WriteStatement("ON CONFLICT")
	w.WriteBlank()

	if len(upsert.Columns) > 0 {
		w.WriteString("(")
		for i, s := range upsert.Columns {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteString(s)
		}
		w.WriteString(")")
	}
	w.WriteBlank()
	if len(upsert.List) == 0 {
		w.WriteKeyword("DO NOTHING")
		return nil
	}
	w.WriteKeyword("UPDATE SET")
	w.WriteNL()
	if err := w.FormatAssignment(upsert.List); err != nil {
		return err
	}
	return w.FormatWhere(upsert.Where)
}

func (w *Writer) FormatAssignment(list []Statement) error {
	w.Enter()
	defer w.Leave()

	var err error
	for i, s := range list {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		ass, ok := s.(Assignment)
		if !ok {
			return w.CanNotUse("assignment", s)
		}
		w.WritePrefix()
		switch field := ass.Field.(type) {
		case Name:
			w.FormatName(field)
		case List:
			err = w.formatList(field)
		default:
			return w.CanNotUse("assignment", s)
		}
		if err != nil {
			return err
		}
		w.WriteString("=")
		switch value := ass.Value.(type) {
		case List:
			err = w.formatList(value)
		default:
			err = w.FormatExpr(value, false)
		}
		if err != nil {
			return err
		}
	}
	return err
}

func (w *Writer) FormatReturning(stmt Statement) error {
	if stmt == nil {
		return nil
	}
	w.WriteStatement("RETURNING")
	w.WriteBlank()

	list, ok := stmt.(List)
	if !ok {
		return w.FormatExpr(stmt, false)
	}
	return w.formatStmtSlice(list.Values)
}
