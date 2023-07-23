package lang

type CreateTableParser interface {
	ParseTable() (Statement, error)
	ParseColumnDef() (Statement, error)
}

func (p *Parser) ParseCreateTable() (Statement, error) {
	return nil, nil
}