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
	{"replace", "into"},
	{"insert", "or", "abort", "into"},
	{"insert", "or", "fail", "into"},
	{"insert", "or", "ignore", "into"},
	{"insert", "or", "replace", "into"},
	{"insert", "or", "rollback", "into"},
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
	local.RegisterParseFunc("REPLACE INTO", local.ParseInsert)
	local.RegisterParseFunc("INSERT OR ABORT INTO", local.ParseInsert)
	local.RegisterParseFunc("INSERT OR FAIL INTO", local.ParseInsert)
	local.RegisterParseFunc("INSERT OR IGNORE INTO", local.ParseInsert)
	local.RegisterParseFunc("INSERT OR REPLACE INTO", local.ParseInsert)
	local.RegisterParseFunc("INSERT OR ROLLBACK INTO", local.ParseInsert)
	return &local, nil
}

func (p *Parser) ParseSelect() (lang.Statement, error) {
	return p.ParseSelectStatement(p)
}

func (p *Parser) ParseInsert() (lang.Statement, error) {
	var (
		stmt InsertStatement
		err  error
	)
	switch {
	case p.IsKeyword("INSERT INTO") || p.IsKeyword("REPLACE INTO"):
	case p.IsKeyword("INSERT OR ABORT INTO"):
		stmt.Action = "ABORT"
	case p.IsKeyword("INSERT OR FAIL INTO"):
		stmt.Action = "FAIL"
	case p.IsKeyword("INSERT OR IGNORE INTO"):
		stmt.Action = "IGNORE"
	case p.IsKeyword("INSERT OR REPLACE INTO"):
		stmt.Action = "REPLACE"
	case p.IsKeyword("INSERT OR ROLLBACK INTO"):
		stmt.Action = "REPLACE"
	default:
		return nil, p.UnexpectedDialect("insert", Vendor)
	}
	stmt.Statement, err = p.Parser.ParseInsert()
	return stmt, err
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
