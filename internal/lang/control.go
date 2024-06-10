package lang

func (p *Parser) parseSet() (Statement, error) {
	p.Next()
	var (
		stmt SetStatement
		err  error
	)
	stmt.Ident = p.GetCurrLiteral()
	p.Next()
	if !p.Is(Eq) {
		return nil, p.Unexpected("set")
	}
	p.Next()

	stmt.Expr, err = p.StartExpression()
	return stmt, err
}

func (p *Parser) parseDeclare() (Statement, error) {
	p.Next()

	var (
		stmt Declare
		err  error
	)
	if !p.Is(Ident) {
		return nil, p.Unexpected("declare")
	}
	stmt.Ident = p.GetCurrLiteral()
	p.Next()

	stmt.Type, err = p.ParseType()
	if err != nil {
		return nil, err
	}

	if p.IsKeyword("DEFAULT") {
		p.Next()
		stmt.Value, err = p.StartExpression()
		if err != nil {
			return nil, err
		}
	}
	return stmt, nil
}

func (p *Parser) parseIf() (Statement, error) {
	p.Next()

	var (
		stmt IfStatement
		err  error
	)
	if stmt.Cdt, err = p.StartExpression(); err != nil {
		return nil, err
	}
	if !p.IsKeyword("THEN") {
		return nil, p.Unexpected("if")
	}
	p.Next()
	stmt.Csq, err = p.ParseBody(p.KwCheck("ELSE", "ELSIF", "END IF"))
	if err != nil {
		return nil, err
	}
	switch {
	case p.IsKeyword("ELSE"):
		p.Next()
		stmt.Alt, err = p.ParseBody(p.KwCheck("END IF"))
	case p.IsKeyword("ELSIF"):
		stmt.Alt, err = p.parseIf()
		return stmt, err
	case p.IsKeyword("END IF"):
	default:
		return nil, p.Unexpected("if")
	}
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("END IF") {
		return nil, p.Unexpected("if")
	}
	p.Next()
	return stmt, nil
}

func (p *Parser) parseWhile() (Statement, error) {
	var (
		stmt WhileStatement
		err  error
	)
	p.Next()

	stmt.Cdt, err = p.StartExpression()
	if err = wrapError("while", err); err != nil {
		return nil, err
	}
	if !p.IsKeyword("DO") {
		return nil, p.Unexpected("while")
	}
	p.Next()
	stmt.Body, err = p.ParseBody(p.KwCheck("END WHILE"))
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("END WHILE") {
		return nil, p.Unexpected("while")
	}
	p.Next()
	return stmt, nil
}

func (p *Parser) ParseBody(done func() bool) (Statement, error) {
	var list List
	for !p.Done() && !done() {
		stmt, err := p.ParseStatement()
		if err != nil {
			return nil, err
		}
		if !p.Is(EOL) {
			return nil, p.Unexpected("body")
		}
		p.Next()
		list.Values = append(list.Values, stmt)
	}
	if !done() {
		return nil, p.Unexpected("body")
	}
	return list, nil
}

func (p *Parser) parseReturn() (Statement, error) {
	p.Next()
	var (
		ret Return
		err error
	)
	ret.Statement, err = p.StartExpression()
	return ret, err
}

func (w *Writer) FormatIf(stmt IfStatement) error {
	if err := w.formatIf(stmt, "IF"); err != nil {
		return err
	}
	w.WriteStatement("END IF")
	return nil
}

func (w *Writer) formatIf(stmt IfStatement, kw string) error {
	w.WriteStatement(kw)
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Cdt, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("THEN")
	w.WriteNL()

	w.Enter()
	if err := w.FormatStatement(stmt.Csq); err != nil {
		return err
	}
	w.Leave()

	var err error
	if stmt.Alt != nil {
		if s, ok := stmt.Alt.(IfStatement); ok {
			err = w.formatIf(s, "ELSIF")
		} else {
			w.WriteStatement("ELSE")
			w.WriteNL()
			w.Enter()
			defer w.Leave()
			err = w.FormatStatement(stmt.Alt)
		}
	}
	return err
}

func (w *Writer) FormatWhile(stmt WhileStatement) error {
	w.WriteStatement("WHILE")
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Cdt, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("DO")
	w.WriteNL()
	if err := w.FormatStatement(stmt.Body); err != nil {
		return err
	}
	w.WriteStatement("END WHILE")
	return nil
}

func (w *Writer) FormatSet(stmt SetStatement) error {
	w.WriteStatement("SET")
	w.WriteBlank()
	w.WriteString(stmt.Ident)
	w.WriteBlank()
	w.WriteString("=")
	w.WriteBlank()
	return w.FormatExpr(stmt.Expr, false)
}

func (w *Writer) FormatReturn(stmt Return) error {
	w.WriteStatement("RETURN")
	if stmt.Statement != nil {
		w.WriteBlank()
		if err := w.FormatExpr(stmt.Statement, false); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatDeclare(stmt Declare) error {
	w.WriteStatement("DECLARE")
	w.WriteBlank()
	w.WriteString(stmt.Ident)
	w.WriteBlank()
	if err := w.FormatType(stmt.Type); err != nil {
		return err
	}
	if stmt.Value != nil {
		w.WriteBlank()
		w.WriteKeyword("DEFAULT")
		w.WriteBlank()
		if err := w.FormatExpr(stmt.Value, false); err != nil {
			return err
		}
	}
	return nil
}
