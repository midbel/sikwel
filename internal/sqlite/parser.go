package sqlite

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/midbel/sweet/internal/lang"
)

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
	local.RegisterParseFunc("UPDATE OR ABORT", local.ParseUpdate)
	local.RegisterParseFunc("UPDATE OR FAIL", local.ParseUpdate)
	local.RegisterParseFunc("UPDATE OR IGNORE", local.ParseUpdate)
	local.RegisterParseFunc("UPDATE OR REPLACE", local.ParseUpdate)
	local.RegisterParseFunc("UPDATE OR ROLLBACK", local.ParseUpdate)
	local.RegisterParseFunc("VACUUM", local.ParseVacuum)
	return &local, nil
}

func (p *Parser) ParseVacuum() (lang.Statement, error) {
	var (
		stmt VacuumStatement
		err  error
	)
	p.Next()
	if p.Is(lang.Ident) {
		stmt.Schema = p.GetCurrLiteral()
		p.Next()
	}
	if p.IsKeyword("INTO") {
		p.Next()
		stmt.File = p.GetCurrLiteral()
		p.Next()
	}
	return stmt, err
}

func (p *Parser) ParseType() (lang.Type, error) {
	t, err := p.Parser.ParseType()
	if err == nil {
		switch t.Name {
		case TypeInteger:
		case TypeReal:
		case TypeText:
		case TypeBlob:
		default:
			return t, fmt.Errorf("%s not a sqlite type", t.Name)
		}
	}
	return t, err
}

func (p *Parser) ParseUpdate() (lang.Statement, error) {
	var (
		stmt UpdateStatement
		err  error
	)
	switch {
	case p.IsKeyword("UPDATE"):
	case p.IsKeyword("UPDATE OR ABORT"):
		stmt.Action = "ABORT"
	case p.IsKeyword("UPDATE OR FAIL"):
		stmt.Action = "FAIL"
	case p.IsKeyword("UPDATE OR IGNORE"):
		stmt.Action = "IGNORE"
	case p.IsKeyword("UPDATE OR REPLACE"):
		stmt.Action = "REPLACE"
	case p.IsKeyword("UPDATE OR ROLLBACK"):
		stmt.Action = "ROLLBACK"
	default:
		return nil, p.UnexpectedDialect("update", Vendor)
	}
	stmt.Statement, err = p.Parser.ParseUpdate()
	return stmt, err
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
		stmt.Action = "ROLLBACK"
	default:
		return nil, p.UnexpectedDialect("insert", Vendor)
	}
	stmt.Statement, err = p.Parser.ParseInsert()
	return stmt, err
}

func (p *Parser) ParseSelect() (lang.Statement, error) {
	return p.ParseSelectStatement(p)
}

func (p *Parser) ParseLimit() (lang.Statement, error) {
	if !p.IsKeyword("LIMIT") {
		return nil, nil
	}
	var (
		lim lang.Limit
		err error
	)
	p.Next()
	lim.Count, err = strconv.Atoi(p.GetCurrLiteral())
	if err != nil {
		return nil, p.Unexpected("limit")
	}
	p.Next()
	if !p.Is(lang.Comma) && !p.IsKeyword("OFFSET") {
		return lim, nil
	}
	p.Next()
	lim.Offset, err = strconv.Atoi(p.GetCurrLiteral())
	if err != nil {
		return nil, p.Unexpected("offset")
	}
	p.Next()
	return lim, nil
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
