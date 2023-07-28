package sqlite

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
	case InsertStatement:
		err = w.FormatInsert(stmt)
	case UpdateStatement:
		err = w.FormatUpdate(stmt)
	case VacuumStatement:
		err = w.FormatVacuum(stmt)
	case BeginStatement:
		err = w.FormatBegin(stmt)
	case lang.SelectStatement:
		err = w.FormatSelect(stmt)
	case lang.CreateTableStatement:
		err = w.FormatCreateTableWithFormatter(w, stmt)
	default:
		err = w.Writer.FormatStatement(stmt)
	}
	return err
}

func (w *Writer) FormatConstraint(cst lang.Statement) error {
	conflict, ok := cst.(ConflictConstraint)
	if ok {
		cst = conflict.Constraint
	}
	if err := w.Writer.FormatConstraint(cst); err != nil {
		return err
	}
	if ok && conflict.Conflict != "" {
		w.WriteBlank()
		w.WriteKeyword("ON CONFLICT")
		w.WriteBlank()
		w.WriteKeyword(conflict.Conflict)
	}
	return nil
}

func (w *Writer) FormatBegin(stmt BeginStatement) error {
	kw, err := stmt.Keyword()
	if err != nil {
		return err
	}
	w.WriteStatement(kw)
	w.WriteNL()
	if err := w.FormatStatement(stmt.Body); err != nil {
		return err
	}
	return w.FormatStatement(stmt.End)
}

func (w *Writer) FormatVacuum(stmt VacuumStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	if stmt.Schema != "" {
		w.WriteString(stmt.Schema)
		w.WriteBlank()
	}
	if stmt.File != "" {
		w.WriteKeyword("INTO")
		w.WriteBlank()
		w.WriteString(stmt.File)
	}
	return nil
}

func (w *Writer) FormatInsert(stmt InsertStatement) error {
	kw, err := stmt.Keyword()
	if err != nil {
		return err
	}
	insert, ok := stmt.Statement.(lang.InsertStatement)
	if !ok {
		return fmt.Errorf("insert: unexpected statement type(%T)", stmt)
	}
	return w.Writer.FormatInsertWithKeyword(kw, insert)
}

func (w *Writer) FormatUpdate(stmt UpdateStatement) error {
	kw, err := stmt.Keyword()
	if err != nil {
		return err
	}
	update, ok := stmt.Statement.(lang.UpdateStatement)
	if !ok {
		return fmt.Errorf("update: unexpected statement type(%T)", stmt)
	}
	return w.Writer.FormatUpdateWithKeyword(kw, update)
}

func (w *Writer) FormatSelect(stmt lang.SelectStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteNL()
	if err := w.FormatSelectColumns(stmt.Columns); err != nil {
		return err
	}
	w.WriteNL()
	if err := w.FormatFrom(stmt.Tables); err != nil {
		return err
	}
	if stmt.Where != nil {
		w.WriteNL()
		if err := w.FormatWhere(stmt.Where); err != nil {
			return err
		}
	}
	if len(stmt.Groups) > 0 {
		w.WriteNL()
		if err := w.FormatGroupBy(stmt.Groups); err != nil {
			return err
		}
	}
	if stmt.Having != nil {
		w.WriteNL()
		if err := w.FormatHaving(stmt.Having); err != nil {
			return err
		}
	}
	if len(stmt.Orders) > 0 {
		w.WriteNL()
		if err := w.FormatOrderBy(stmt.Orders); err != nil {
			return err
		}
	}
	if stmt.Limit != nil {
		w.WriteNL()
		if err := w.FormatLimit(stmt.Limit); err != nil {
			return nil
		}
	}
	return nil
}

func (w *Writer) FormatOrderBy(orders []lang.Statement) error {
	if len(orders) == 0 {
		return nil
	}
	w.WriteStatement("ORDER BY")
	w.WriteBlank()
	for i, s := range orders {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		order, ok := s.(Order)
		if !ok {
			return w.canNotUse("order by", s)
		}
		if err := w.formatOrder(order); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) formatOrder(order Order) error {
	n, ok := order.Order.Statement.(lang.Name)
	if !ok {
		return w.CanNotUse("order by", order.Statement)
	}
	w.FormatName(n)
	if order.Collate != "" {
		w.WriteBlank()
		w.WriteKeyword("COLLATE")
		w.WriteBlank()
		w.WriteString(order.Collate)
	}
	if order.Orient != "" {
		w.WriteBlank()
		w.WriteString(order.Orient)
	}
	if order.Nulls != "" {
		w.WriteBlank()
		w.WriteKeyword("NULLS")
		w.WriteBlank()
		w.WriteString(order.Nulls)
	}
	return nil
}

func (w *Writer) canNotUse(ctx string, stmt lang.Statement) error {
	err := w.CanNotUse(ctx, stmt)
	return fmt.Errorf("sqlite: %w", err)
}
