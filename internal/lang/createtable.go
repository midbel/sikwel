package lang

type ConstraintParser interface {
	ParsePrimaryKey() (Statement, error)
	ParseForeignKey() (Statement, error)
	ParseNotNull() (Statement, error)
	ParseDefault() (Statement, error)
	ParseCheck() (Statement, error)
	ParseUnique() (Statement, error)
	ParseGeneratedAlways() (Statement, error)
}

type CreateTableParser interface {
	ParseTableName() (Statement, error)
	ParseConstraint() (Statement, error)
	ParseColumnDef(CreateTableParser) (Statement, error)
}

func (p *Parser) ParseCreateTable() (Statement, error) {
	return p.ParseCreateTableStatement(p)
}

func (p *Parser) ParseCreateTableStatement(ctp CreateTableParser) (Statement, error) {
	var (
		stmt CreateTableStatement
		err  error
	)
	if stmt.Name, err = ctp.ParseTableName(); err != nil {
		return nil, err
	}
	if !p.Is(Lparen) {
		return nil, p.Unexpected("create table")
	}
	for !p.Done() && !p.Is(Rparen) {
		if p.IsKeyword("CONSTRAINT") {
			c, err := ctp.ParseConstraint()
			if err != nil {
				return nil, err
			}
			stmt.Constraints = append(stmt.Constraints, c)
			continue
		} else {
			c, err := ctp.ParseColumnDef(ctp)
			if err != nil {
				return nil, err
			}
			stmt.Columns = append(stmt.Columns, c)
		}
		if err = p.EnsureEnd("create table", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("create table")
	}
	p.Next()
	return stmt, err
}

func (p *Parser) ParseTableName() (Statement, error) {
	return p.ParseIdentifier()
}

func (p *Parser) ParseConstraint() (Statement, error) {
	var (
		cst Constraint
		err error
	)
	if p.IsKeyword("CONSTRAINT") {
		p.Next()
		cst.Name = p.GetCurrLiteral()
	}
	p.Next()
	switch {
	case p.IsKeyword("PRIMARY KEY"):
	case p.IsKeyword("FOREIGN KEY"):
	case p.IsKeyword("UNIQUE"):
	case p.IsKeyword("NOT NULL"):
	case p.IsKeyword("CHECK"):
	case p.IsKeyword("DEFAULT"):
	case p.IsKeyword("GENERATED ALWAYS"):
	default:
		return nil, p.Unexpected("constraint")
	}
	return cst, err
}

func (p *Parser) ParseColumnDef(ctp CreateTableParser) (Statement, error) {
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
	if def.Constraint, err = ctp.ParseConstraint(); err != nil {
		return nil, err
	}
	return def, err
}
