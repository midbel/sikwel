package sqlite

import (
	"io"
	"strings"

	"github.com/midbel/sweet/internal/lang"
)

const (
	CollateBinary = "BINARY"
	CollateNocase = "NOCASE"
	CollateTrim   = "RTRIM"
)

var keywords = lang.KeywordSet{
	{"collate"},
}

const Vendor = "sqlite"

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
	local.RegisterParseFunc("SELECT", local.ParseSelect)
	return &local, nil
}

func (p *Parser) ParseSelect() (lang.Statement, error) {
	return p.ParseSelectStatement(p)
}

func (p *Parser) ParseOrderBy() ([]lang.Statement, error) {
	if !p.IsKeyword("ORDER BY") {
		return nil, nil
	}
	p.Next()
	do := func(stmt lang.Statement) (lang.Statement, error) {
		base := lang.Order{
			Statement: stmt,
		}
		order := Order{
			Order: base,
		}
		if p.IsKeyword("COLLATE") {
			p.Next()
			order.Collate = p.GetCurrLiteral()
			if !isValidCollate(order.Collate) {
				return nil, p.UnexpectedDialect("order by", Vendor)
			}
			p.Next()
		}
		if p.IsKeyword("ASC") || p.IsKeyword("DESC") {
			order.Orient = p.GetCurrLiteral()
			p.Next()
		}
		if p.IsKeyword("NULLS") {
			p.Next()
			if !p.IsKeyword("FIRST") && !p.IsKeyword("LAST") {
				return nil, p.UnexpectedDialect("order by", Vendor)
			}
			order.Nulls = p.GetCurrLiteral()
			p.Next()
		}
		return order, nil
	}
	return p.ParseStatementList("order by", do)
}

func isValidCollate(str string) bool {
	switch strings.ToUpper(str) {
	case CollateBinary, CollateNocase, CollateTrim:
		return true
	default:
		return false
	}
}
