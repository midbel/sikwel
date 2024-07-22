package parser

import (
	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/token"
)

func (p *Parser) ParseGrant() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.GrantStatement
		err  error
	)
	if stmt.Privileges, err = p.parsePrivileges(); err != nil {
		return nil, err
	}
	if !p.IsKeyword("ON") {
		return nil, p.Unexpected("grant", keywordExpected("ON"))
	}
	p.Next()
	if !p.Is(token.Ident) {
		return nil, p.Unexpected("grant", identExpected)
	}
	stmt.Object = p.GetCurrLiteral()
	p.Next()
	if !p.IsKeyword("TO") {
		return nil, p.Unexpected("grant", keywordExpected("TO"))
	}
	p.Next()
	if stmt.Users, err = p.parseGranted(); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *Parser) ParseRevoke() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.RevokeStatement
		err  error
	)
	if stmt.Privileges, err = p.parsePrivileges(); err != nil {
		return nil, err
	}
	if !p.IsKeyword("ON") {
		return nil, p.Unexpected("revoke", keywordExpected("ON"))
	}
	p.Next()
	if !p.Is(token.Ident) {
		return nil, p.Unexpected("revoke", identExpected)
	}
	stmt.Object = p.GetCurrLiteral()
	p.Next()
	if !p.IsKeyword("FROM") {
		return nil, p.Unexpected("revoke", keywordExpected("FROM"))
	}
	p.Next()
	if stmt.Users, err = p.parseGranted(); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *Parser) parseGranted() ([]string, error) {
	var list []string
	for !p.QueryEnds() && !p.Done() {
		if !p.Is(token.Ident) {
			return nil, p.Unexpected("role", identExpected)
		}
		list = append(list, p.GetCurrLiteral())
		p.Next()
		switch {
		case p.Is(token.Comma):
			p.Next()
			if p.QueryEnds() {
				return nil, p.Unexpected("role", "unexpected comma before end of statement")
			}
		case p.QueryEnds():
		default:
			return nil, p.Unexpected("role", defaultReason)
		}
	}
	return list, nil
}

func (p *Parser) parsePrivileges() ([]string, error) {
	if p.IsKeyword("ALL") || p.IsKeyword("ALL PRIVILEGES") {
		p.Next()
		return nil, nil
	}
	var list []string
	for !p.QueryEnds() && !p.Done() && !p.IsKeyword("ON") {
		if !p.Is(token.Keyword) {
			return nil, p.Unexpected("privileges", "keyword expected")
		}
		list = append(list, p.GetCurrLiteral())
		p.Next()
		switch {
		case p.Is(token.Comma):
			p.Next()
			if p.IsKeyword("ON") {
				return nil, p.Unexpected("privileges", keywordExpected("ON"))
			}
		case p.IsKeyword("ON"):
		default:
			return nil, p.Unexpected("privileges", defaultReason)
		}
	}
	return list, nil
}
