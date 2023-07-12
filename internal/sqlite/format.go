package sqlite

import (
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

func (w *Writer) FormatOrderBy(orders []lang.Statement) error {
	if len(orders) == 0 {
		return nil
	}
	w.WriteNL()
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
