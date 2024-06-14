package format

import (
	"github.com/midbel/sweet/internal/lang/ast"
)

func (w *Writer) FormatIf(stmt ast.If) error {
	if err := w.formatIf(stmt, "IF"); err != nil {
		return err
	}
	w.WriteStatement("END IF")
	return nil
}

func (w *Writer) formatIf(stmt ast.If, kw string) error {
	w.WriteStatement(kw)
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Cdt, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("THEN")
	w.WriteNL()

	if err := w.FormatStatement(stmt.Csq); err != nil {
		return err
	}

	var err error
	if stmt.Alt != nil {
		if s, ok := stmt.Alt.(ast.If); ok {
			err = w.formatIf(s, "ELSIF")
		} else {
			w.WriteStatement("ELSE")
			w.WriteNL()
			err = w.FormatStatement(stmt.Alt)
		}
	}
	return err
}

func (w *Writer) FormatWhile(stmt ast.While) error {
	w.WriteStatement("WHILE")
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Cdt, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("DO")
	w.WriteNL()
	if err := w.FormatStatement(stmt.Body); err != nil {
		return err
	}
	w.WriteStatement("END WHILE")
	return nil
}

func (w *Writer) FormatSet(stmt ast.Set) error {
	w.WriteStatement("SET")
	w.WriteBlank()
	w.WriteString(stmt.Ident)
	w.WriteBlank()
	w.WriteString("=")
	w.WriteBlank()
	return w.FormatExpr(stmt.Expr, false)
}

func (w *Writer) FormatReturn(stmt ast.Return) error {
	w.WriteStatement("RETURN")
	if stmt.Statement != nil {
		w.WriteBlank()
		if err := w.FormatExpr(stmt.Statement, false); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatDeclare(stmt ast.Declare) error {
	w.WriteStatement("DECLARE")
	w.WriteBlank()
	w.WriteString(stmt.Ident)
	w.WriteBlank()
	if err := w.FormatType(stmt.Type); err != nil {
		return err
	}
	if stmt.Value != nil {
		w.WriteBlank()
		w.WriteKeyword("DEFAULT")
		w.WriteBlank()
		if err := w.FormatExpr(stmt.Value, false); err != nil {
			return err
		}
	}
	return nil
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
