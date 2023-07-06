package sqlite

import (
	"io"

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
	if local.Parser, err = lang.NewParser(r); err != nil {
		return nil, err
	}
	local.RegisterParseFunc("ORDER BY", local.parseOrder)
	return &local, nil
}

func (p *Parser) parseOrder() (lang.Statement, error) {
	return nil, nil
}
