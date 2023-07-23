package lang

func (p *Parser) parseSetTransaction() (Statement, error) {
	p.Next()
	var (
		stmt SetTransaction
		err  error
	)
	if p.IsKeyword("ISOLATION LEVEL") {
		p.Next()
		switch {
		case p.IsKeyword("REPEATABLE READ"):
			stmt.Level = LevelReadRepeat
		case p.IsKeyword("READ COMMITTED"):
			stmt.Level = LevelReadCommit
		case p.IsKeyword("READ UNCOMMITTED"):
			stmt.Level = LevelReadUncommit
		case p.IsKeyword("SERIALIZABLE"):
			stmt.Level = LevelSerializable
		default:
			return nil, p.Unexpected("transaction")
		}
		p.Next()
	}
	switch {
	case p.IsKeyword("READ ONLY"):
		stmt.Mode = ModeReadOnly
		p.Next()
	case p.IsKeyword("READ WRITE"):
		stmt.Mode = ModeReadWrite
		p.Next()
	case p.Is(EOL):
	default:
		return nil, p.Unexpected("transaction")
	}
	return stmt, err
}

func (p *Parser) parseStartTransaction() (Statement, error) {
	defer func() {
		p.UnregisterParseFunc("SAVEPOINT")
		p.UnregisterParseFunc("COMMIT")
		p.UnregisterParseFunc("ROLLBACK")
		p.UnregisterParseFunc("RELEASE")
		p.UnregisterParseFunc("RELEASE SAVEPOINT")
		p.UnregisterParseFunc("ROLLBACK TO SAVEPOINT")
		p.UnregisterParseFunc("SET TRANSACTION")
	}()
	p.Next()

	var (
		stmt StartTransaction
		err  error
	)
	switch {
	case p.IsKeyword("READ ONLY"):
		stmt.Mode = ModeReadOnly
		p.Next()
	case p.IsKeyword("READ WRITE"):
		stmt.Mode = ModeReadWrite
		p.Next()
	case p.Is(EOL):
	default:
		return nil, p.Unexpected("transaction")
	}
	if !p.Is(EOL) {
		return nil, p.Unexpected("transaction")
	}
	p.Next()

	p.RegisterParseFunc("SAVEPOINT", p.parseSavepoint)
	p.RegisterParseFunc("RELEASE", p.parseReleaseSavepoint)
	p.RegisterParseFunc("RELEASE SAVEPOINT", p.parseReleaseSavepoint)
	p.RegisterParseFunc("ROLLBACK TO SAVEPOINT", p.parseRollbackSavepoint)
	p.RegisterParseFunc("COMMIT", p.parseCommit)
	p.RegisterParseFunc("ROLLBACK", p.parseRollback)
	p.RegisterParseFunc("SET TRANSACTION", p.parseSetTransaction)

	stmt.Body, err = p.ParseBody(p.KwCheck("END", "COMMIT", "ROLLBACK"))
	if err != nil {
		return nil, err
	}
	switch {
	case p.IsKeyword("END") || p.IsKeyword("COMMIT"):
		stmt.End = Commit{}
	case p.IsKeyword("ROLLBACK"):
		stmt.End = Rollback{}
	default:
		return nil, p.Unexpected("transaction")
	}
	p.Next()
	return stmt, err
}

func (p *Parser) parseSavepoint() (Statement, error) {
	p.Next()
	var (
		stmt Savepoint
		err  error
	)
	if p.Is(Ident) {
		stmt.Name = p.GetCurrLiteral()
		p.Next()
	}
	return stmt, err
}

func (p *Parser) parseReleaseSavepoint() (Statement, error) {
	p.Next()
	var (
		stmt ReleaseSavepoint
		err  error
	)
	if !p.Is(Ident) {
		return nil, p.Unexpected("release savepoint")
	}
	stmt.Name = p.GetCurrLiteral()
	p.Next()
	return stmt, err
}

func (p *Parser) parseRollbackSavepoint() (Statement, error) {
	p.Next()
	var (
		stmt RollbackSavepoint
		err  error
	)
	if !p.Is(Ident) {
		return nil, p.Unexpected("rollback savepoint")
	}
	stmt.Name = p.GetCurrLiteral()
	p.Next()
	return stmt, err
}

func (p *Parser) parseCommit() (Statement, error) {
	p.Next()
	return Commit{}, nil
}

func (p *Parser) parseRollback() (Statement, error) {
	p.Next()
	return Rollback{}, nil
}