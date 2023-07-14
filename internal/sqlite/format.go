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
		if err = w.format(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatStatement(stmt lang.Statement) error {
	return w.format(stmt)
}

func (w *Writer) format(stmt lang.Statement) error {
	defer w.Flush()

	w.Reset()
	err := w.formatStatement(stmt)
	if err == nil {
		w.WriteString(";")
		w.WriteNL()
	}
	return err
}

func (w *Writer) formatStatement(stmt lang.Statement) error {
	var err error
	switch stmt := stmt.(type) {
	case lang.SelectStatement:
		err = w.FormatSelect(stmt)
	case lang.UnionStatement:
		err = w.FormatUnion(stmt)
	case lang.IntersectStatement:
		err = w.FormatIntersect(stmt)
	case lang.ExceptStatement:
		err = w.FormatExcept(stmt)
	case InsertStatement:
		err = w.FormatInsert(stmt)
	case UpdateStatement:
		err = w.FormatUpdate(stmt)
	case lang.DeleteStatement:
		err = w.FormatDelete(stmt)
	case lang.WithStatement:
		err = w.FormatWith(stmt)
	case lang.CteStatement:
		err = w.FormatCte(stmt)
	default:
		err = fmt.Errorf("unsupported statement type %T", stmt)
	}
	return err
}

func (w *Writer) FormatInsert(stmt InsertStatement) error {
	w.Enter()
	defer w.Leave()

	kw, err := stmt.Keyword()
	if err != nil {
		return err
	}
	w.WriteString(kw)
	w.WriteBlank()
	return w.formatInsertStatement(stmt.Statement)
}

func (w *Writer) formatInsertStatement(stmt lang.Statement) error {
	ins, ok := stmt.(lang.InsertStatement)
	if !ok {
		return fmt.Errorf("insert: unexpected statement type(%T)", stmt)
	}
	if err := w.FormatExpr(ins.Table, false); err != nil {
		return err
	}
	if len(ins.Columns) > 0 {
		w.WriteString("(")
		for i, c := range ins.Columns {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteString(c)
		}
		w.WriteString(")")
	}
	w.WriteBlank()
	if err := w.FormatInsertValues(ins.Values); err != nil {
		return err
	}
	if ins.Upsert != nil {
		w.WriteNL()
		if err := w.FormatUpsert(ins.Upsert); err != nil {
			return err
		}
	}
	if ins.Return != nil {
		w.WriteNL()
		if err := w.FormatReturn(ins.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatUpdate(stmt UpdateStatement) error {
	w.Enter()
	defer w.Leave()

	kw, err := stmt.Keyword()
	if err != nil {
		return err
	}
	w.WritePrefix()
	w.WriteString(kw)
	w.WriteBlank()

	return w.formatUpdateStatement(stmt.Statement)
}

func (w *Writer) formatUpdateStatement(stmt lang.Statement) error {
	up, ok := stmt.(lang.UpdateStatement)
	if !ok {
		return fmt.Errorf("update: unexpected statement type(%T)", stmt)
	}

	switch stmt := up.Table.(type) {
	case lang.Name:
		w.FormatName(stmt)
	case lang.Alias:
		if err := w.FormatAlias(stmt); err != nil {
			return err
		}
	default:
		return w.CanNotUse("update", stmt)
	}

	w.WriteBlank()
	w.WriteString("SET")
	w.WriteNL()

	if len(up.Tables) > 0 {
		w.WriteNL()
		if err := w.FormatFrom(up.Tables); err != nil {
			return err
		}
	}
	if up.Where != nil {
		w.WriteNL()
		if err := w.FormatWhere(up.Where); err != nil {
			return err
		}
	}
	if up.Return != nil {
		w.WriteNL()
		if err := w.FormatReturn(up.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatSelect(stmt lang.SelectStatement) error {
	w.Enter()
	defer w.Leave()

	w.WritePrefix()
	w.WriteString("SELECT")
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
	w.WritePrefix()
	w.WriteString("ORDER BY")
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
		w.WriteString("COLLATE")
		w.WriteBlank()
		w.WriteString(order.Collate)
	}
	if order.Orient != "" {
		w.WriteBlank()
		w.WriteString(order.Orient)
	}
	if order.Nulls != "" {
		w.WriteBlank()
		w.WriteString("NULLS")
		w.WriteBlank()
		w.WriteString(order.Nulls)
	}
	return nil
}

func (w *Writer) canNotUse(ctx string, stmt lang.Statement) error {
	err := w.CanNotUse(ctx, stmt)
	return fmt.Errorf("sqlite: %w", err)
}
