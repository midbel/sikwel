package lang

func (p *Parser) ParseCall() (Statement, error) {
	p.Next()
	var (
		stmt CallStatement
		err  error
	)
	stmt.Ident, err = p.ParseIdentifier()
	if err != nil {
		return nil, err
	}
	if !p.Is(Lparen) {
		return nil, p.Unexpected("call")
	}
	p.Next()
	for !p.Done() && !p.Is(Rparen) {
		if p.peekIs(Arrow) && p.Is(Ident) {
			stmt.Names = append(stmt.Names, p.GetCurrLiteral())
			p.Next()
			p.Next()
		}
		arg, err := p.StartExpression()
		if err = wrapError("call", err); err != nil {
			return nil, err
		}
		if err := p.EnsureEnd("call", Comma, Rparen); err != nil {
			return nil, err
		}
		stmt.Args = append(stmt.Args, arg)
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("call")
	}
	p.Next()
	return stmt, err
}
