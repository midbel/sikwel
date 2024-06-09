package lang

func (p *Parser) ParseCreateProcedure() (Statement, error) {
	var (
		stmt CreateProcedureStatement
		err  error
	)
	if p.IsKeyword("CREATE OR REPLACE PROCEDURE") {
		stmt.Replace = true
	}
	p.Next()
	stmt.Name = p.GetCurrLiteral()
	p.Next()
	if stmt.Parameters, err = p.ParseProcedureParameters(); err != nil {
		return nil, err
	}
	if p.IsKeyword("LANGUAGE") {
		p.Next()
		stmt.Language = p.GetCurrLiteral()
		p.Next()
	}
	if !p.IsKeyword("BEGIN") {
		return nil, p.Unexpected("procedure")
	}
	p.Next()

	// p.RegisterParseFunc("CASE", p.parseCase)
	// p.RegisterParseFunc("IF", p.parseIf)
	// p.RegisterParseFunc("WHILE", p.parseWhile)
	// p.RegisterParseFunc("DECLARE", p.parseDeclare)
	// p.RegisterParseFunc("SET", p.parseSet)
	// p.RegisterParseFunc("RETURN", p.parseReturn)

	// defer func() {
	// 	p.UnregisterParseFunc("CASE")
	// 	p.UnregisterParseFunc("IF")
	// 	p.UnregisterParseFunc("WHILE")
	// 	p.UnregisterParseFunc("DECLARE")
	// 	p.UnregisterParseFunc("RETURN")
	// }()

	stmt.Body, err = p.ParseBody(func() bool {
		return p.IsKeyword("END")
	})
	if err == nil {
		p.Next()
	}
	return stmt, err
}

func (p *Parser) ParseProcedureParameters() ([]Statement, error) {
	if err := p.Expect("procedure", Lparen); err != nil {
		return nil, err
	}
	p.Next()
	var list []Statement
	for !p.Done() && !p.Is(Rparen) {
		stmt, err := p.ParseProcedureParameter()
		if err != nil {
			return nil, err
		}
		list = append(list, stmt)
		if err := p.EnsureEnd("procedure", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	return list, p.Expect("procedure", Rparen)
}

func (p *Parser) ParseProcedureParameter() (Statement, error) {
	var (
		param ProcedureParameter
		err   error
	)
	switch {
	case p.IsKeyword("IN"):
		param.Mode = ModeIn
	case p.IsKeyword("OUT"):
		param.Mode = ModeOut
	case p.IsKeyword("INOUT"):
		param.Mode = ModeInOut
	default:
	}
	if param.Mode != 0 {
		p.Next()
	}
	if !p.Is(Ident) {
		return nil, p.Unexpected("procedure")
	}
	param.Name = p.GetCurrLiteral()
	p.Next()
	if param.Type, err = p.ParseType(); err != nil {
		return nil, err
	}
	if p.IsKeyword("DEFAULT") || p.Is(Eq) {
		p.Next()
		param.Default, err = p.StartExpression()
		if err != nil {
			return nil, err
		}
	}
	return param, nil
}

func (w *Writer) FormatCreateProcedure(stmt CreateProcedureStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	w.WriteString(stmt.Name)
	w.WriteString("(")

	w.Enter()
	for i, s := range stmt.Parameters {
		if i > 0 {
			w.WriteString(",")
		}
		p, ok := s.(ProcedureParameter)
		if !ok {
			return w.CanNotUse(s)
		}
		w.WriteNL()
		w.WritePrefix()
		switch p.Mode {
		case ModeIn:
			w.WriteKeyword("IN")
		case ModeOut:
			w.WriteKeyword("OUT")
		case ModeInOut:
			w.WriteKeyword("INOUT")
		}
		if p.Mode != 0 {
			w.WriteBlank()
		}
		w.WriteString(p.Name)
		w.WriteBlank()
		if err := w.FormatType(p.Type); err != nil {
			return err
		}
		if p.Default != nil {
			w.WriteBlank()
			w.WriteKeyword("DEFAULT")
			w.WriteBlank()
			if err := w.FormatExpr(p.Default, false); err != nil {
				return err
			}
		}
	}
	w.Leave()
	w.WriteNL()
	w.WriteString(")")
	w.WriteNL()
	if stmt.Language != "" {
		w.WriteKeyword("LANGUAGE")
		w.WriteBlank()
		w.WriteString(stmt.Language)
		w.WriteNL()
	}
	w.WriteKeyword("BEGIN")
	w.WriteNL()
	w.Enter()
	if err := w.FormatStatement(stmt.Body); err != nil {
		return err
	}
	w.Leave()
	w.WriteKeyword("END")
	return nil
}
