package postgres

import (
	"fmt"
	"io"
	"strings"

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
	local.RegisterParseFunc("SELECT", local.ParseSelect)
	local.RegisterParseFunc("MERGE", local.ParseMerge)
	local.RegisterParseFunc("COPY", local.ParseCopy)
	local.RegisterParseFunc("TRUNCATE", local.ParseTruncate)
	local.RegisterParseFunc("TRUNCATE TABLE", local.ParseTruncate)
	return &local, nil
}

func (p *Parser) ParseMerge() (lang.Statement, error) {
	var (
		stmt MergeStatement
		err  error
	)
	return stmt, err
}

func (p *Parser) ParseCopy() (lang.Statement, error) {
	var (
		stmt CopyStatement
		err  error
	)
	p.Next()
	if !p.Is(lang.Lparen) {
		stmt.Table, err = p.ParseIdent()
		if err != nil {
			return nil, err
		}
	}
	if p.Is(lang.Lparen) {
		p.Next()
		if p.Is(lang.Keyword) {
			stmt.Query, err = p.ParseStatement()
			if err != nil {
				return nil, err
			}
		} else {
			for !p.Done() && !p.Is(lang.Rparen) {
				stmt.Columns = append(stmt.Columns, p.GetCurrLiteral())
				p.Next()
				if err := p.EnsureEnd("copy", lang.Comma, lang.Rparen); err != nil {
					return nil, err
				}
			}
		}
		if !p.Is(lang.Rparen) {
			return nil, p.Unexpected("copy")
		}
		p.Next()
	}
	switch {
	case p.IsKeyword("FROM"):
		stmt.Import = true
		if stmt.Query != nil {
			return nil, fmt.Errorf("query not legal for copy from")
		}
	case p.IsKeyword("TO"):
	default:
		return nil, p.Unexpected("copy")
	}
	p.Next()
	switch {
	case p.IsKeyword("PROGRAM"):
		p.Next()
		stmt.Program = true
		stmt.File = p.GetCurrLiteral()
	case p.IsKeyword("STDIN"):
		stmt.Stdio = true
		stmt.File = "STDIN"
	case p.IsKeyword("STDOUT"):
		stmt.Stdio = true
		stmt.File = "STDOUT"
	case p.Is(lang.Literal):
		stmt.File = p.GetCurrLiteral()
	default:
		return nil, p.Unexpected("copy")
	}
	p.Next()
	if p.IsKeyword("WITH") || p.Is(lang.Lparen) {
		if p.IsKeyword("WITH") {
			p.Next()
			if !p.Is(lang.Lparen) {
				return nil, p.Unexpected("copy")
			}
		}
		p.Next()
		for !p.Done() && !p.Is(lang.Rparen) {
			option := p.GetCurrLiteral()
			p.Next()
			if !p.Is(lang.Literal) && !p.Is(lang.Number) {
				return nil, p.Unexpected("copy")
			}
			switch val := p.GetCurrLiteral(); strings.ToUpper(option) {
			case "FORMAT":
				stmt.Header = val
			case "DELIMITER":
				stmt.Delimiter = val
			case "FREEZE":
				stmt.Freeze = val
			case "NULL":
				stmt.Null = val
			case "HEADER":
				stmt.Header = val
			case "QUOTE":
				stmt.Quote = val
			case "ESCAPE":
				stmt.Escape = val
			case "FORCE_QUOTE":
				stmt.ForceQuote = val
			case "FORCE_NOT_NULL":
				stmt.ForceNotNull = val
			case "FORCE_NULL":
				stmt.ForceNull = val
			case "FORCE_ENCODING":
				stmt.Encoding = val
			default:
				return nil, p.Unexpected("copy")
			}
			p.Next()
			if err := p.EnsureEnd("copy", lang.Comma, lang.Rparen); err != nil {
				return nil, err
			}
		}
		if !p.Is(lang.Rparen) {
			return nil, p.Unexpected("copy")
		}
		p.Next()
	}
	if p.IsKeyword("WHERE") {
		if stmt.Import {
			return nil, fmt.Errorf("where not legal for copy to")
		}
		stmt.Where, err = p.ParseWhere()
		if err != nil {
			return nil, err
		}
	}
	return stmt, nil
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

	for !p.Done() && !p.Is(lang.EOL) && !p.Is(lang.Keyword) {
		ident, err := p.ParseIdent()
		if err != nil {
			return nil, err
		}
		if p.Is(lang.Star) {
			p.Next()
		}
		stmt.Tables = append(stmt.Tables, ident)
		switch {
		case p.Is(lang.Comma):
			p.Next()
			if p.Is(lang.Keyword) || p.Is(lang.EOL) {
				return nil, p.Unexpected("truncate")
			}
		case p.Is(lang.Keyword):
		case p.Is(lang.EOL):
		default:
			return nil, p.Unexpected("truncate")
		}
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

func (p *Parser) ParseSelect() (lang.Statement, error) {
	return p.ParseSelectStatement(p)
}

func (p *Parser) ParseOrderBy() ([]lang.Statement, error) {
	if !p.IsKeyword("ORDER BY") {
		return nil, nil
	}
	p.Next()
	var (
		list []lang.Statement
		err  error
	)
	for !p.Done() && !p.Is(lang.EOL) && !p.Is(lang.Rparen) && !p.Is(lang.Keyword) {
		var stmt lang.Statement
		stmt, err = p.StartExpression()
		if err != nil {
			return nil, err
		}
		order := lang.Order{
			Statement: stmt,
		}
		if p.IsKeyword("ASC") || p.IsKeyword("DESC") {
			order.Orient = p.GetCurrLiteral()
			p.Next()
		} else if p.IsKeyword("USING") {
			p.Next()
			switch {
			case p.Is(lang.Gt):
				order.Orient = ">"
			case p.Is(lang.Ge):
				order.Orient = ">="
			case p.Is(lang.Lt):
				order.Orient = "<"
			case p.Is(lang.Le):
				order.Orient = "<="
			default:
				return nil, fmt.Errorf("invalid operator in using")
			}
			p.Next()
		}
		if p.IsKeyword("NULLS") {
			p.Next()
			if !p.IsKeyword("FIRST") && !p.IsKeyword("LAST") {
				return nil, p.Unexpected("order by")
			}
			order.Nulls = p.GetCurrLiteral()
			p.Next()
		}
		list = append(list, order)
		switch {
		case p.Is(lang.Comma):
			p.Next()
			if p.Is(lang.EOL) || p.Is(lang.Rparen) || p.Is(lang.Keyword) {
				return nil, p.Unexpected("order by")
			}
		case p.Is(lang.Keyword):
		case p.Is(lang.EOL):
		case p.Is(lang.Rparen):
		default:
			return nil, p.Unexpected("order by")
		}
	}
	return list, err
}

func (p *Parser) Unexpected(ctx string) error {
	return p.UnexpectedDialect(ctx, Vendor)
}
