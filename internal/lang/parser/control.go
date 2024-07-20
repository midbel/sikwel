package parser

import (
	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/token"
)

func (p *Parser) parseSet() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.Set
		err  error
	)
	stmt.Ident = p.GetCurrLiteral()
	p.Next()
	if !p.Is(token.Eq) {
		return nil, p.Unexpected("set")
	}
	p.Next()

	stmt.Expr, err = p.StartExpression()
	return stmt, err
}

func (p *Parser) ParseDeclare() (ast.Statement, error) {
	p.Next()

	var (
		stmt ast.Declare
		err  error
	)
	if !p.Is(token.Ident) {
		return nil, p.Unexpected("declare")
	}
	stmt.Ident = p.GetCurrLiteral()
	p.Next()

	stmt.Type, err = p.ParseType()
	if err != nil {
		return nil, err
	}

	if p.IsKeyword("DEFAULT") {
		p.Next()
		stmt.Value, err = p.StartExpression()
		if err != nil {
			return nil, err
		}
	}
	return stmt, nil
}

func (p *Parser) parseIf() (ast.Statement, error) {
	p.Next()

	var (
		stmt ast.If
		err  error
	)
	if stmt.Cdt, err = p.StartExpression(); err != nil {
		return nil, err
	}
	if !p.IsKeyword("THEN") {
		return nil, p.Unexpected("if")
	}
	p.Next()
	stmt.Csq, err = p.ParseBody(p.KwCheck("ELSE", "ELSIF", "END IF"))
	if err != nil {
		return nil, err
	}
	switch {
	case p.IsKeyword("ELSE"):
		p.Next()
		stmt.Alt, err = p.ParseBody(p.KwCheck("END IF"))
	case p.IsKeyword("ELSIF"):
		stmt.Alt, err = p.parseIf()
		return stmt, err
	case p.IsKeyword("END IF"):
	default:
		return nil, p.Unexpected("if")
	}
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("END IF") {
		return nil, p.Unexpected("if")
	}
	p.Next()
	return stmt, nil
}

func (p *Parser) parseWhile() (ast.Statement, error) {
	var (
		stmt ast.While
		err  error
	)
	p.Next()

	stmt.Cdt, err = p.StartExpression()
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("DO") {
		return nil, p.Unexpected("while")
	}
	p.Next()
	stmt.Body, err = p.ParseBody(p.KwCheck("END WHILE"))
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("END WHILE") {
		return nil, p.Unexpected("while")
	}
	p.Next()
	return stmt, nil
}

func (p *Parser) ParseBody(done func() bool) (ast.Statement, error) {
	var list ast.List
	for !p.Done() && !done() {
		stmt, err := p.ParseStatement()
		if err != nil {
			return nil, err
		}
		if !p.Is(token.EOL) {
			return nil, p.Unexpected("body")
		}
		p.Next()
		list.Values = append(list.Values, stmt)
	}
	if !done() {
		return nil, p.Unexpected("body")
	}
	return list, nil
}

func (p *Parser) parseReturn() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.Return
		err  error
	)
	stmt.Statement, err = p.StartExpression()
	return stmt, err
}
