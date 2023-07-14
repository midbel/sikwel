package postgres

import (
	"io"

	"github.com/midbel/sweet/internal/lang"
)

const Vendor = "postgres"

type Parser struct {
	*lang.Parser
}

func NewParser(r io.Reader) (*Parser, error) {
	var (
		local Parser
		err   error
	)
	base := lang.GetKeywords()
	if local.Parser, err = lang.NewParserWithKeywords(r, base.Merge(keywords)); err != nil {
		return nil, err
	}
	local.RegisterParseFunc("TRUNCATE", local.ParseTruncate)
	local.RegisterParseFunc("TRUNCATE TABLE", local.ParseTruncate)
	return &local, nil
}

func (p *Parser) ParseTruncate() (lang.Statement, error) {
	var (
		stmt TruncateStatement
		err  error
	)
	p.Next()
	if p.IsKeyword("ONLY") {
		p.Next()
		stmt.Only = true
	}

	stmt.Tables, err = p.ParseStatementList("truncate", p.ParseAlias)
	if err != nil {
		return nil, err
	}
	switch {
	case p.IsKeyword("RESTART IDENTITY"):
		stmt.Identity = "restart"
		p.Next()
	case p.IsKeyword("CONTINUE IDENTITY"):
		stmt.Identity = "continue"
		p.Next()
	default:
	}
	switch {
	case p.IsKeyword("CASCADE"):
		stmt.Cascade = true
		p.Next()
	case p.IsKeyword("RESTRICT"):
		stmt.Restrict = true
		p.Next()
	default:
	}
	return stmt, err
}
