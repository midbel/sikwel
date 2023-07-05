package sqlite

import (
	"github.com/midbel/sweet/internal/lang"
)

type Parser struct {
	*lang.Parser
}

func NewParser(r io.Reader) (*Parser, error) {
	var (
		local Parser
		err   error
	)
	if local.Parser, err = lang.NewParser(r, lang.GetKeywords()); err != nil {
		return nil, err
	}
	local.RegisterParseFunc("ORDER BY", local.parseOrder)
	return &local, nil
}

func (p *Parser) parseOrder() (ast.Statement, error) {
	return nil, nil
}
