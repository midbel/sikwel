package lang

import (
	"strings"

	"github.com/midbel/sweet/internal/lang/ast"
)

func (p *Parser) ParseCreateProcedure() (ast.Statement, error) {
	var (
		stmt ast.CreateProcedureStatement
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

func (p *Parser) ParseProcedureParameters() ([]ast.Statement, error) {
	if err := p.Expect("procedure", Lparen); err != nil {
		return nil, err
	}
	var list []ast.Statement
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

func (p *Parser) ParseProcedureParameter() (ast.Statement, error) {
	var (
		param ast.ProcedureParameter
		err   error
	)
	switch {
	case p.IsKeyword("IN"):
		param.Mode = ast.ModeIn
	case p.IsKeyword("OUT"):
		param.Mode = ast.ModeOut
	case p.IsKeyword("INOUT"):
		param.Mode = ast.ModeInOut
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

func (w *Writer) FormatCreateProcedure(stmt ast.CreateProcedureStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	w.WriteString(stmt.Name)
	w.WriteString("(")
	w.WriteNL()

	for i, s := range stmt.Parameters {
		if i > 0 {
			w.WriteString(",")
			w.WriteNL()
		}
		p, ok := s.(ast.ProcedureParameter)
		if !ok {
			return w.CanNotUse("create procedure", s)
		}
		if err := w.formatParamter(p); err != nil {
			return err
		}
	}
	w.WriteNL()
	w.WriteString(")")
	w.WriteNL()
	if stmt.Language != "" {
		w.WriteKeyword("LANGUAGE")
		w.WriteBlank()
		w.WriteString(stmt.Language)
		w.WriteNL()
	}
	w.WriteKeyword("BEGIN")
	w.WriteNL()
	if err := w.FormatStatement(stmt.Body); err != nil {
		return err
	}
	w.WriteKeyword("END")
	return nil
}

func (w *Writer) formatParamter(param ast.ProcedureParameter) error {
	w.Enter()
	defer w.Leave()

	w.WritePrefix()
	switch param.Mode {
	case ast.ModeIn:
		w.WriteKeyword("IN")
	case ast.ModeOut:
		w.WriteKeyword("OUT")
	case ast.ModeInOut:
		w.WriteKeyword("INOUT")
	}
	if param.Mode != 0 {
		w.WriteBlank()
	}
	if w.Upperize.Identifier() || w.Upperize.All() {
		param.Name = strings.ToUpper(param.Name)
	}
	w.WriteString(param.Name)
	w.WriteBlank()
	if err := w.FormatType(param.Type); err != nil {
		return err
	}
	if param.Default != nil {
		w.WriteBlank()
		w.WriteKeyword("DEFAULT")
		w.WriteBlank()
		if err := w.FormatExpr(param.Default, false); err != nil {
			return err
		}
	}
	return nil
}
