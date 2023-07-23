package lang

func (p *Parser) parseSet() (Statement, error) {
	p.Next()
	var (
		stmt Assignment
		err  error
	)
	if stmt.Field, err = p.ParseIdent(); err != nil {
		return nil, wrapError("set", err)
	}
	if !p.Is(Eq) {
		return nil, p.Unexpected("set")
	}
	p.Next()

	stmt.Value, err = p.StartExpression()
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
	stmt.Ident = p.curr.Literal
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
		return p.parseIf()
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
