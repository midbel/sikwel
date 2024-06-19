package db2

import (
	"io"

	"github.com/midbel/sweet/internal/lang"
	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/lang/parser"
	"github.com/midbel/sweet/internal/token"
)

type Parser struct {
	*parser.Parser
}

func Parse(r io.Reader) (lang.Parser, error) {
	scan, err := Scan(r)
	if err != nil {
		return nil, err
	}
	var ps Parser
	ps.Parser, err = parser.ParseWithScanner(scan)
	if err != nil {
		return nil, err
	}

	ps.RegisterParseFunc("CREATE PROCEDURE", ps.ParseCreateProcedure)
	ps.RegisterParseFunc("CREATE OR REPLACE PROCEDURE", ps.ParseCreateProcedure)

	return &ps, err
}

func (p *Parser) ParseCreateProcedure() (ast.Statement, error) {
	var (
		stmt CreateProcedureStatement
		err  error
	)
	if p.IsKeyword("CREATE OR REPLACE PROCEDURE") {
		stmt.Replace = true
	}
	p.Next()
	stmt.Name, err = p.ParseProcedureName()
	if err != nil {
		return nil, err
	}
	if stmt.Parameters, err = p.ParseProcedureParameters(); err != nil {
		return nil, err
	}
	if stmt.Language, err = p.ParseProcedureLanguage(); err != nil {
		return nil, err
	}
	if p.IsKeyword("DETERMINISTIC") || p.IsKeyword("NOT DETERMINISTIC") {
		stmt.Deterministic = p.IsKeyword("DETERMINISTIC")
		p.Next()
	}
	if p.IsKeyword("MODIFIES SQL DATA") {
		stmt.StmtSpec = ModifiesSql
		p.Next()
	} else if p.IsKeyword("READ SQL DATA") {
		stmt.StmtSpec = ReadsSql
		p.Next()
	} else if p.IsKeyword("CONTAINS SQL") {
		stmt.StmtSpec = ContainsSql
		p.Next()
	}
	if p.IsKeyword("CALLED ON NULL INPUT") {
		stmt.NullInput = true
		p.Next()
	}
	if stmt.Options, err = p.ParseProcedureOptions(); err != nil {
		return nil, err
	}
	stmt.Body, err = p.ParseProcedureBody()
	return stmt, err
}

func (p *Parser) ParseProcedureOptions() (ast.Statement, error) {
	if !p.IsKeyword("SET OPTION") {
		return nil, nil
	}
	p.Next()
	var list ast.List
	for !p.Done() && p.PeekIs(token.Eq) {
		if !p.Is(token.Ident) && !p.Is(token.Keyword) {
			return nil, p.Unexpected("set option")
		}
		key := ast.Name{
			Parts: []string{p.GetCurrLiteral()},
		}
		p.Next()
		if !p.Is(token.Eq) {
			return nil, p.Unexpected("set option")
		}
		p.Next()
		val := ast.Name{
			Parts: []string{p.GetCurrLiteral()},
		}
		p.Next()

		ass := ast.Assignment{
			Field: key,
			Value: val,
		}
		list.Values = append(list.Values, ass)
	}
	return list, nil
}
