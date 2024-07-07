package format

import (
	"strconv"
	"strings"

	"github.com/midbel/sweet/internal/lang/ast"
)

func (w *Writer) FormatPlaceholder(name ast.Placeholder) error {
	if name.Statement == nil {
		w.WriteString("?")
		return nil
	}
	switch stmt := name.Statement.(type) {
	case ast.Value:
		w.WriteString("$")
		w.WriteString(stmt.Literal)
	case ast.Name:
		w.WriteString(":")
		w.FormatName(stmt)
	default:
		return w.CanNotUse("placeholder", name.Statement)
	}
	return nil
}

func (w *Writer) FormatName(name ast.Name) error {
	for i := range name.Parts {
		if i > 0 {
			w.WriteString(".")
		}
		str := name.Parts[i]
		if str == "" && i == len(name.Parts)-1 {
			str = "*"
		}
		if w.Upperize.Identifier() || w.Upperize.All() {
			str = strings.ToUpper(str)
		}
		if w.UseQuote && str != "*" {
			str = w.Quote(str)
		}
		w.WriteString(str)
	}
	return nil
}

func (w *Writer) FormatAlias(alias ast.Alias) error {
	err := w.FormatExpr(alias.Statement, false)
	if err != nil {
		return err
	}
	w.WriteBlank()
	if w.UseAs {
		w.WriteKeyword("AS")
		w.WriteBlank()
	}
	str := alias.Alias
	if w.Upperize.Identifier() || w.Upperize.All() {
		str = strings.ToUpper(str)
	}
	if w.UseQuote {
		str = w.Quote(str)
	}
	w.WriteString(str)
	return nil
}

func (w *Writer) FormatLiteral(literal string) {
	if literal == "NULL" || literal == "DEFAULT" || literal == "TRUE" || literal == "FALSE" || literal == "*" {
		if w.withColor() {
			w.WriteString(keywordColor)
		}
		w.WriteKeyword(literal)
		if w.withColor() {
			w.WriteString(resetCode)
		}
		return
	}
	if _, err := strconv.Atoi(literal); err == nil {
		if w.withColor() {
			w.WriteString(numberColor)
		}
		w.WriteString(literal)
		if w.withColor() {
			w.WriteString(resetCode)
		}
		return
	}
	if _, err := strconv.ParseFloat(literal, 64); err == nil {
		if w.withColor() {
			w.WriteString(numberColor)
		}
		w.WriteString(literal)
		if w.withColor() {
			w.WriteString(resetCode)
		}
		return
	}
	w.WriteQuoted(literal)
}

func (w *Writer) FormatRow(stmt ast.Row, nl bool) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	w.WriteString("(")
	for i, v := range stmt.Values {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		if nl {
			w.WriteNL()
		}
		if err := w.FormatExpr(v, false); err != nil {
			return err
		}
	}
	if nl {
		w.WriteNL()
	}
	w.WriteString(")")
	return nil
}

func (w *Writer) FormatCase(stmt ast.Case) error {
	w.WriteKeyword("CASE")
	if stmt.Cdt != nil {
		w.WriteBlank()
		w.FormatExpr(stmt.Cdt, false)
	}
	for _, s := range stmt.Body {
		w.WriteNL()
		if err := w.FormatExpr(s, false); err != nil {
			return err
		}
	}
	if stmt.Else != nil {
		w.WriteNL()
		w.Enter()
		w.WritePrefix()
		w.WriteKeyword("ELSE")
		w.WriteBlank()

		if err := w.FormatExpr(stmt.Else, false); err != nil {
			return err
		}
		w.Leave()
	}
	w.WriteNL()
	w.WritePrefix()
	w.WriteKeyword("END")
	return nil
}

func (w *Writer) FormatWhen(stmt ast.When) error {
	w.Enter()
	defer w.Leave()
	w.WritePrefix()
	w.WriteKeyword("WHEN")
	w.WriteBlank()

	err := w.compact(func() error {
		return w.FormatExpr(stmt.Cdt, false)
	})
	if err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("THEN")
	w.WriteBlank()

	return w.FormatExpr(stmt.Body, false)
}

func (w *Writer) FormatCast(stmt ast.Cast, _ bool) error {
	w.WriteKeyword("CAST")
	w.WriteString("(")
	if err := w.FormatExpr(stmt.Ident, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("AS")
	w.WriteBlank()
	if err := w.FormatType(stmt.Type); err != nil {
		return err
	}
	w.WriteString(")")
	return nil
}

func (w *Writer) FormatType(dt ast.Type) error {
	if w.Upperize.Type() || w.Upperize.All() {
		dt.Name = strings.ToUpper(dt.Name)
	}
	w.WriteString(dt.Name)
	if dt.Length <= 0 {
		return nil
	}
	w.WriteString("(")
	w.WriteString(strconv.Itoa(dt.Length))
	if dt.Precision > 0 {
		w.WriteString(",")
		w.WriteBlank()
		w.WriteString(strconv.Itoa(dt.Precision))
	}
	w.WriteString(")")
	return nil
}
