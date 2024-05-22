package lang

func (p *Parser) ParseGrant() (Statement, error) {
	p.Next()
	var (
		stmt GrantStatement
		err  error
	)
	if stmt.Privileges, err = p.parsePrivileges(); err != nil {
		return nil, err
	}
	if !p.IsKeyword("ON") {
		return nil, p.Unexpected("grant")
	}
	p.Next()
	if !p.Is(Ident) {
		return nil, p.Unexpected("grant")
	}
	stmt.Object = p.GetCurrLiteral()
	p.Next()
	if !p.IsKeyword("TO") {
		return nil, p.Unexpected("grant")
	}
	p.Next()
	if stmt.Users, err = p.parseGranted(); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *Parser) ParseRevoke() (Statement, error) {
	p.Next()
	var (
		stmt RevokeStatement
		err  error
	)
	if stmt.Privileges, err = p.parsePrivileges(); err != nil {
		return nil, err
	}
	if !p.IsKeyword("ON") {
		return nil, p.Unexpected("revoke")
	}
	p.Next()
	if !p.Is(Ident) {
		return nil, p.Unexpected("revoke")
	}
	stmt.Object = p.GetCurrLiteral()
	p.Next()
	if !p.IsKeyword("FROM") {
		return nil, p.Unexpected("revoke")
	}
	p.Next()
	if stmt.Users, err = p.parseGranted(); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *Parser) parseGranted() ([]string, error) {
	var list []string
	for !p.QueryEnds() && !p.Done() {
		if !p.Is(Ident) {
			return nil, p.Unexpected("role")
		}
		list = append(list, p.GetCurrLiteral())
		p.Next()
		switch {
		case p.Is(Comma):
			p.Next()
			if p.QueryEnds() {
				return nil, p.Unexpected("role")
			}
		case p.QueryEnds():
		default:
			return nil, p.Unexpected("role")
		}
	}
	return list, nil
}

func (p *Parser) parsePrivileges() ([]string, error) {
	if p.IsKeyword("ALL") || p.IsKeyword("ALL PRIVILEGES") {
		p.Next()
		return nil, nil
	}
	var list []string
	for !p.QueryEnds() && !p.Done() && !p.IsKeyword("ON") {
		if !p.Is(Keyword) {
			return nil, p.Unexpected("privileges")
		}
		list = append(list, p.GetCurrLiteral())
		p.Next()
		switch {
		case p.Is(Comma):
			p.Next()
			if p.IsKeyword("ON") {
				return nil, p.Unexpected("privileges")
			}
		case p.IsKeyword("ON"):
		default:
			return nil, p.Unexpected("privileges")
		}
	}
	return list, nil
}
