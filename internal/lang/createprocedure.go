package lang

// import (
// 	"fmt"
// )

func (p *Parser) ParseCreateProcedure() (Statement, error) {
	var (
		stmt CreateProcedureStatement
		err  error
	)
	if p.IsKeyword("CREATE OR REPLACE PROCEDURE") {
		stmt.Replace = true
	}
	p.Next()
	stmt.Name = p.GetCurrLiteral()
	p.Next()
	if stmt.Parameters, err = p.ParseProcedureParameters(); err != nil {
		return nil, err
	}
	if p.IsKeyword("LANGUAGE") {
		p.Next()
		stmt.Language = p.GetCurrLiteral()
		p.Next()
	}
	if !p.IsKeyword("BEGIN") {
		return nil, p.Unexpected("procedure")
	}
	p.Next()
	stmt.Body, err = p.ParseBody(func() bool {
		return p.IsKeyword("END")
	})
	if err == nil {
		p.Next()
	}
	return stmt, err
}

func (p *Parser) ParseProcedureParameters() ([]Statement, error) {
	if err := p.Expect("procedure", Lparen); err != nil {
		return nil, err
	}
	p.Next()
	var list []Statement
	for !p.Done() && !p.Is(Rparen) {
		stmt, err := p.ParseProcedureParameter()
		if err != nil {
			return nil, err
		}
		list = append(list, stmt)
		if err := p.EnsureEnd("procedure", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	return list, p.Expect("procedure", Rparen)
}

func (p *Parser) ParseProcedureParameter() (Statement, error) {
	var (
		param ProcedureParameter
		err   error
	)
	switch {
	case p.IsKeyword("IN"):
		param.Mode = ModeIn
	case p.IsKeyword("OUT"):
		param.Mode = ModeOut
	case p.IsKeyword("INOUT"):
		param.Mode = ModeInOut
	default:
	}
	if param.Mode != 0 {
		p.Next()
	}
	if !p.Is(Ident) {
		return nil, p.Unexpected("procedure")
	}
	param.Name = p.GetCurrLiteral()
	p.Next()
	if param.Type, err = p.ParseType(); err != nil {
		return nil, err
	}
	if p.IsKeyword("DEFAULT") || p.Is(Eq) {
		p.Next()
		param.Default, err = p.StartExpression()
		if err != nil {
			return nil, err
		}
	}
	return param, nil
}
