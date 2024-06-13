package lang

import (
	"github.com/midbel/sweet/internal/lang/ast"
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
	if !p.Is(Lparen) {
		return nil, p.Unexpected("call")
	}
	p.Next()
	for !p.Done() && !p.Is(Rparen) {
		if p.peekIs(Arrow) && p.Is(Ident) {
			stmt.Names = append(stmt.Names, p.GetCurrLiteral())
			p.Next()
			p.Next()
		}
		arg, err := p.StartExpression()
		if err = wrapError("call", err); err != nil {
			return nil, err
		}
		if err := p.EnsureEnd("call", Comma, Rparen); err != nil {
			return nil, err
		}
		stmt.Args = append(stmt.Args, arg)
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("call")
	}
	p.Next()
	return stmt, err
}

func (w *Writer) FormatCall(stmt ast.CallStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteString("(")
	defer w.WriteString(")")

	w.Enter()
	defer w.Leave()

	w.WriteNL()
	for i, a := range stmt.Args {
		if i > 0 {
			w.WriteString(",")
			w.WriteNL()
		}
		w.WritePrefix()
		if err := w.FormatExpr(a, false); err != nil {
			return err
		}
	}
	w.WriteNL()
	return nil
}
