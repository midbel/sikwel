package parser

import (
	"strconv"

	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/token"
)

func (p *Parser) ParseLiteral() (ast.Statement, error) {
	stmt := ast.Value{
		Literal: p.GetCurrLiteral(),
	}
	p.Next()
	return stmt, nil
}

func (p *Parser) ParseConstant() (ast.Statement, error) {
	if !p.Is(token.Keyword) {
		return nil, p.Unexpected("constant")
	}
	switch p.GetCurrLiteral() {
	case "TRUE", "FALSE", "UNKNOWN", "NULL", "DEFAULT":
	default:
		return nil, p.Unexpected("constant")
	}
	return p.ParseLiteral()
}

func (p *Parser) ParseIdentifier() (ast.Statement, error) {
	var name ast.Name
	for p.PeekIs(token.Dot) {
		name.Parts = append(name.Parts, p.GetCurrLiteral())
		p.Next()
		p.Next()
	}
	if !p.Is(token.Ident) && !p.Is(token.Star) {
		return nil, p.Unexpected("identifier")
	}
	name.Parts = append(name.Parts, p.GetCurrLiteral())
	p.Next()
	return name, nil
}

func (p *Parser) ParseIdent() (ast.Statement, error) {
	stmt, err := p.ParseIdentifier()
	if err == nil {
		stmt, err = p.ParseAlias(stmt)
	}
	return stmt, nil
}

func (p *Parser) ParseAlias(stmt ast.Statement) (ast.Statement, error) {
	mandatory := p.IsKeyword("AS")
	if mandatory {
		p.Next()
	}
	switch p.Curr().Type {
	case token.Ident, token.Literal, token.Number:
		stmt = ast.Alias{
			Statement: stmt,
			Alias:     p.GetCurrLiteral(),
		}
		p.Next()
	default:
		if mandatory {
			return nil, p.Unexpected("alias")
		}
	}
	return stmt, nil
}

func (p *Parser) ParseCase() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.Case
		err  error
	)
	if !p.IsKeyword("WHEN") {
		stmt.Cdt, err = p.StartExpression()
		if err = wrapError("case", err); err != nil {
			return nil, err
		}
	}
	for p.IsKeyword("WHEN") {
		var when ast.When
		p.Next()
		when.Cdt, err = p.StartExpression()
		if err = wrapError("when", err); err != nil {
			return nil, err
		}
		if !p.IsKeyword("THEN") {
			return nil, p.Unexpected("case")
		}
		p.Next()
		if p.Is(token.Keyword) {
			when.Body, err = p.ParseStatement()
		} else {
			when.Body, err = p.StartExpression()
		}
		if err = wrapError("then", err); err != nil {
			return nil, err
		}
		stmt.Body = append(stmt.Body, when)
	}
	if p.IsKeyword("ELSE") {
		p.Next()
		stmt.Else, err = p.StartExpression()
		if err = wrapError("else", err); err != nil {
			return nil, err
		}
	}
	if !p.IsKeyword("END") {
		return nil, p.Unexpected("case")
	}
	p.Next()
	return p.ParseAlias(stmt)
}

func (p *Parser) ParseCast() (ast.Statement, error) {
	p.Next()
	if !p.Is(token.Lparen) {
		return nil, p.Unexpected("cast")
	}
	p.Next()
	var (
		cast ast.Cast
		err  error
	)
	cast.Ident, err = p.ParseIdentifier()
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("AS") {
		return nil, p.Unexpected("cast")
	}
	p.Next()
	if cast.Type, err = p.ParseType(); err != nil {
		return nil, err
	}
	if !p.Is(token.Rparen) {
		return nil, p.Unexpected("cast")
	}
	p.Next()
	return cast, nil
}

func (p *Parser) ParseType() (ast.Type, error) {
	var t ast.Type
	if !p.Is(token.Ident) {
		return t, p.Unexpected("type")
	}
	t.Name = p.GetCurrLiteral()
	p.Next()
	if p.Is(token.Lparen) {
		p.Next()
		size, err := strconv.Atoi(p.GetCurrLiteral())
		if err != nil {
			return t, err
		}
		t.Length = size
		p.Next()
		if p.Is(token.Comma) {
			p.Next()
			size, err = strconv.Atoi(p.GetCurrLiteral())
			if err != nil {
				return t, err
			}
			t.Precision = size
			p.Next()
		}
		if !p.Is(token.Rparen) {
			return t, p.Unexpected("type")
		}
		p.Next()
	}
	return t, nil
}

func (p *Parser) ParseRow() (ast.Statement, error) {
	p.Next()
	if !p.Is(token.Lparen) {
		return nil, p.Unexpected("row")
	}
	p.Next()

	p.setDefaultFuncSet()
	defer p.unsetFuncSet()

	var row ast.Row
	for !p.Done() && !p.Is(token.Rparen) {
		expr, err := p.StartExpression()
		if err != nil {
			return nil, err
		}
		row.Values = append(row.Values, expr)
		if err = p.EnsureEnd("row", token.Comma, token.Rparen); err != nil {
			return nil, err
		}
	}
	if !p.Is(token.Rparen) {
		return nil, p.Unexpected("row")
	}
	p.Next()
	return row, nil
}
