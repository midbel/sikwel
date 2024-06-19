package format

import (
	"strings"

	"github.com/midbel/sweet/internal/lang/ast"
)

func (w *Writer) FormatCreateProcedure(stmt ast.CreateProcedureStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Name, false); err != nil {
		return err
	}
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
