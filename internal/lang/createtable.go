package lang

// type ConstraintParser interface {
// 	ParsePrimaryKey() (Statement, error)
// 	ParseForeignKey() (Statement, error)
// 	ParseNotNull() (Statement, error)
// 	ParseDefault() (Statement, error)
// 	ParseCheck() (Statement, error)
// 	ParseUnique() (Statement, error)
// 	ParseGeneratedAlways() (Statement, error)
// }

// type CreateTableParser interface {
// 	ParseTableName() (Statement, error)
// 	ParseConstraint(ConstraintParser) (Statement, error)
// 	ParseColumnDef(CreateTableParser) (Statement, error)
// }

func (p *Parser) ParseCreateTable() (Statement, error) {
	return p.ParseCreateTableStatement()
}

func (p *Parser) ParseCreateTableStatement() (Statement, error) {
	p.Next()
	var (
		stmt CreateTableStatement
		err  error
	)
	if stmt.Name, err = p.ParseTableName(); err != nil {
		return nil, err
	}
	if err := p.Expect("create table", Lparen); err != nil {
		return nil, err
	}
	for !p.Done() && !p.Is(Rparen) && !p.Is(Keyword) {
		def, err := p.ParseColumnDef()
		if err != nil {
			return nil, err
		}
		stmt.Columns = append(stmt.Columns, def)
		if err = p.EnsureEnd("create table", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	for !p.Done() && !p.Is(Rparen) {
		cst, err := p.ParseConstraint(false)
		if err != nil {
			return nil, err
		}
		stmt.Constraints = append(stmt.Constraints, cst)
		if err = p.EnsureEnd("create table", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	return stmt, p.Expect("create table", Rparen)
}

func (p *Parser) ParseTableName() (Statement, error) {
	return p.ParseIdentifier()
}

func (p *Parser) ParseColumnDef() (Statement, error) {
	var (
		def ColumnDef
		err error
	)
	def.Name = p.GetCurrLiteral()
	p.Next()
	if def.Type, err = p.ParseType(); err != nil {
		return nil, err
	}
	if p.Is(Comma) {
		return def, nil
	}
	for !p.Done() && !p.Is(Comma) && !p.Is(Rparen) {
		cst, err := p.ParseConstraint(true)
		if err != nil {
			return nil, err
		}
		def.Constraints = append(def.Constraints, cst)
	}
	return def, err
}

func (p *Parser) ParseConstraint(column bool) (Statement, error) {
	var (
		cst Constraint
		err error
	)
	if p.IsKeyword("CONSTRAINT") {
		p.Next()
		cst.Name = p.GetCurrLiteral()
		p.Next()
	}
	switch {
	case p.IsKeyword("PRIMARY KEY"):
		cst.Statement, err = p.ParsePrimaryKeyConstraint(column)
	case p.IsKeyword("FOREIGN KEY") || p.IsKeyword("REFERENCES"):
		cst.Statement, err = p.ParseForeignKeyConstraint(column)
	case p.IsKeyword("UNIQUE"):
		cst.Statement, err = p.ParseUniqueConstraint(column)
	case p.IsKeyword("NOT"):
		if !column {
			return nil, p.Unexpected("constraint")
		}
		cst.Statement, err = p.ParseNotNullConstraint()
	case p.IsKeyword("CHECK"):
		cst.Statement, err = p.ParseCheckConstraint()
	case p.IsKeyword("DEFAULT"):
		if !column {
			return nil, p.Unexpected("constraint")
		}
		cst.Statement, err = p.ParseDefaultConstraint()
	case p.IsKeyword("GENERATED ALWAYS"):
		cst.Statement, err = p.ParseGeneratedAlwaysConstraint()
	default:
		return nil, p.Unexpected("constraint")
	}
	return cst, err
}

func (p *Parser) ParsePrimaryKeyConstraint(short bool) (Statement, error) {
	p.Next()
	var cst PrimaryKeyConstraint
	if short {
		return cst, nil
	}
	if err := p.Expect("primary key", Lparen); err != nil {
		return nil, err
	}
	for !p.Done() && !p.Is(Rparen) {
		if !p.Is(Ident) {
			return nil, p.Unexpected("primary key")
		}
		cst.Columns = append(cst.Columns, p.GetCurrLiteral())
		p.Next()
		if err := p.EnsureEnd("primary key", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	return cst, p.Expect("primary key", Rparen)
}

func (p *Parser) ParseForeignKeyConstraint(short bool) (Statement, error) {
	var cst ForeignKeyConstraint
	if p.IsKeyword("FOREIGN KEY") {
		p.Next()
		if err := p.Expect("foreign key", Lparen); err != nil {
			return nil, err
		}
		for !p.Done() && !p.Is(Rparen) {
			if !p.Is(Ident) {
				return nil, p.Unexpected("foreign key")
			}
			cst.Locals = append(cst.Locals, p.GetCurrLiteral())
			p.Next()
			if err := p.EnsureEnd("foreign key", Comma, Rparen); err != nil {
				return nil, err
			}
		}
		if err := p.Expect("foreign key", Rparen); err != nil {
			return nil, err
		}
	}
	if !p.IsKeyword("REFERENCES") {
		return nil, p.Unexpected("foreign key")
	}
	p.Next()
	if !p.Is(Ident) {
		return nil, p.Unexpected("foreign key")
	}
	cst.Table = p.GetCurrLiteral()
	p.Next()
	if err := p.Expect("foreign key", Lparen); err != nil {
		return nil, err
	}
	for !p.Done() && !p.Is(Rparen) {
		if !p.Is(Ident) {
			return nil, p.Unexpected("foreign key")
		}
		cst.Remotes = append(cst.Remotes, p.GetCurrLiteral())
		p.Next()
		if err := p.EnsureEnd("foreign key", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	return cst, p.Expect("foreign key", Rparen)
}

func (p *Parser) ParseUniqueConstraint(short bool) (Statement, error) {
	p.Next()
	var cst UniqueConstraint
	if short {
		return cst, nil
	}
	if err := p.Expect("unique", Lparen); err != nil {
		return nil, err
	}
	for !p.Done() && !p.Is(Rparen) {
		if !p.Is(Ident) {
			return nil, p.Unexpected("unique")
		}
		cst.Columns = append(cst.Columns, p.GetCurrLiteral())
		p.Next()
		if err := p.EnsureEnd("unique", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	return cst, p.Expect("unique", Rparen)
}

func (p *Parser) ParseNotNullConstraint() (Statement, error) {
	p.Next()
	var cst NotNullConstraint
	if !p.IsKeyword("NULL") {
		return nil, p.Unexpected("not null")
	}
	p.Next()
	return cst, nil
}

func (p *Parser) ParseCheckConstraint() (Statement, error) {
	p.Next()
	var (
		cst CheckConstraint
		err error
	)
	cst.Expr, err = p.StartExpression()
	return cst, err
}

func (p *Parser) ParseDefaultConstraint() (Statement, error) {
	p.Next()
	var (
		cst DefaultConstraint
		err error
	)
	cst.Expr, err = p.StartExpression()
	return cst, err
}

func (p *Parser) ParseGeneratedAlwaysConstraint() (Statement, error) {
	var (
		cst GeneratedConstraint
		err error
	)
	cst.Expr, err = p.StartExpression()
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("STORED") {
		return nil, p.Unexpected("generated always")
	}
	p.Next()
	return cst, nil
}
