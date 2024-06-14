package parser

import (
	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/token"
)

func (p *Parser) ParseCall() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.CallStatement
		err  error
	)
	stmt.Ident, err = p.ParseIdentifier()
	if err != nil {
		return nil, err
	}
	if !p.Is(token.Lparen) {
		return nil, p.Unexpected("call")
	}
	p.Next()
	for !p.Done() && !p.Is(token.Rparen) {
		if p.peekIs(token.Arrow) && p.Is(token.Ident) {
			stmt.Names = append(stmt.Names, p.GetCurrLiteral())
			p.Next()
			p.Next()
		}
		arg, err := p.StartExpression()
		if err = wrapError("call", err); err != nil {
			return nil, err
		}
		if err := p.EnsureEnd("call", token.Comma, token.Rparen); err != nil {
			return nil, err
		}
		stmt.Args = append(stmt.Args, arg)
	}
	if !p.Is(token.Rparen) {
		return nil, p.Unexpected("call")
	}
	p.Next()
	return stmt, err
}
