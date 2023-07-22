package postgres

import (
	"errors"
	"fmt"
	"io"

	"github.com/midbel/sweet/internal/lang"
)

type Writer struct {
	*lang.Writer
}

func NewWriter(w io.Writer) *Writer {
	var ws Writer
	ws.Writer = lang.NewWriter(w)
	return &ws
}

func (w *Writer) Format(r io.Reader) error {
	p, err := NewParser(r)
	if err != nil {
		return err
	}
	for {
		stmt, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if err = w.startStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) startStatement(stmt lang.Statement) error {
	defer w.Flush()

	w.Reset()
	if com, ok := stmt.(lang.Commented); ok {
		stmt = com.Statement
	}
	err := w.FormatStatement(stmt)
	if err == nil {
		w.WriteString(";")
		w.WriteNL()
	}
	return err
}

func (w *Writer) FormatStatement(stmt lang.Statement) error {
	var err error
	switch stmt := stmt.(type) {
	case TruncateStatement:
		err = w.FormatTruncate(stmt)
	case CopyStatement:
		err = w.FormatCopy(stmt)
	default:
		err = w.Writer.FormatStatement(stmt)
	}
	return err
}

func (w *Writer) FormatCopy(stmt CopyStatement) error {
	w.Enter()
	defer w.Leave()
	
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	
	return nil
}

func (w *Writer) FormatTruncate(stmt TruncateStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Only {
		w.WriteBlank()
		w.WriteKeyword("ONLY")
	}
	w.WriteBlank()
	for i, t := range stmt.Tables {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		if err := w.FormatExpr(t, false); err != nil {
			return err
		}
	}
	if stmt.Identity != "" {
		w.WriteBlank()
		w.WriteString(stmt.Identity)
	}
	if stmt.Cascade || stmt.Restrict {
		w.WriteBlank()
	}
	switch {
	case stmt.Cascade:
		w.WriteKeyword("CASCADE")
	case stmt.Restrict:
		w.WriteKeyword("RESTRICT")
	default:
	}
	return nil
}

func (w *Writer) canNotUse(ctx string, stmt lang.Statement) error {
	err := w.CanNotUse(ctx, stmt)
	return fmt.Errorf("%s: %w", Vendor, err)
}
