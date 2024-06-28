package format

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/midbel/sweet/internal/lang/ast"
)

func (w *Writer) FormatUnion(stmt ast.UnionStatement) error {
	if err := w.FormatStatement(stmt.Left); err != nil {
		return err
	}
	w.WriteNL()
	w.Enter()
	w.WritePrefix()
	w.WriteKeyword("UNION")
	if stmt.All {
		w.WriteBlank()
		w.WriteKeyword("ALL")
	}
	if stmt.Distinct {
		w.WriteBlank()
		w.WriteKeyword("DISTINCT")
	}
	w.WriteNL()
	w.Leave()
	return w.FormatStatement(stmt.Right)
}

func (w *Writer) FormatExcept(stmt ast.ExceptStatement) error {
	if err := w.FormatStatement(stmt.Left); err != nil {
		return err
	}
	w.WriteNL()
	w.Enter()
	w.WritePrefix()
	w.WriteKeyword("EXCEPT")
	if stmt.All {
		w.WriteBlank()
		w.WriteKeyword("ALL")
	}
	if stmt.Distinct {
		w.WriteBlank()
		w.WriteKeyword("DISTINCT")
	}
	w.WriteNL()
	w.Leave()
	return w.FormatStatement(stmt.Right)
}

func (w *Writer) FormatIntersect(stmt ast.IntersectStatement) error {
	if err := w.FormatStatement(stmt.Left); err != nil {
		return err
	}
	w.WriteNL()
	w.Enter()
	w.WritePrefix()
	w.WriteKeyword("INTERSECT")
	if stmt.All {
		w.WriteBlank()
		w.WriteKeyword("ALL")
	}
	if stmt.Distinct {
		w.WriteBlank()
		w.WriteKeyword("DISTINCT")
	}
	w.WriteNL()
	w.Leave()
	return w.FormatStatement(stmt.Right)
}

func (w *Writer) FormatValues(stmt ast.ValuesStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()
	for i := range stmt.List {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		if err := w.FormatExpr(stmt.List[i], false); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatSelect(stmt ast.SelectStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WritePrefix()
	w.WriteKeyword(kw)
	w.WriteNL()
	if err := w.FormatSelectColumns(stmt.Columns); err != nil {
		return err
	}
	w.WriteNL()
	w.WritePrefix()
	if err := w.FormatFrom(stmt.Tables); err != nil {
		return err
	}
	if stmt.Where != nil {
		w.WriteNL()
		w.WritePrefix()
		if err := w.FormatWhere(stmt.Where); err != nil {
			return err
		}
	}
	if len(stmt.Groups) > 0 {
		w.WriteNL()
		w.WritePrefix()
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
	if len(stmt.Windows) > 0 {
		w.WriteNL()
		w.WritePrefix()
		if err := w.FormatWindows(stmt.Windows); err != nil {
			return err
		}
	}
	if len(stmt.Orders) > 0 {
		w.WriteNL()
		w.WritePrefix()
		if err := w.FormatOrderBy(stmt.Orders); err != nil {
			return err
		}
	}
	if stmt.Limit != nil {
		w.WriteNL()
		w.WritePrefix()
		if err := w.FormatLimit(stmt.Limit); err != nil {
			return nil
		}
	}
	return nil
}

func (w *Writer) FormatSelectColumns(columns []ast.Statement) error {
	w.Enter()
	defer w.Leave()
	for i, v := range columns {
		if i > 0 {
			w.WriteString(",")
			w.WriteNL()
		}
		w.WritePrefix()
		if err := w.FormatExpr(v, false); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatWhere(stmt ast.Statement) error {
	if stmt == nil {
		return nil
	}
	w.WriteKeyword("WHERE")
	w.WriteBlank()

	return w.FormatExpr(stmt, true)
}

func (w *Writer) formatFromJoin(join ast.Join) error {
	w.WritePrefix()
	w.WriteKeyword(join.Type)
	w.WriteBlank()

	var err error
	switch s := join.Table.(type) {
	case ast.Name:
		w.FormatName(s)
	case ast.Alias:
		err = w.FormatAlias(s)
	case ast.SelectStatement:
		w.WriteString("(")
		err = w.FormatSelect(s)
		w.WriteString(")")
	default:
		return w.CanNotUse("join", s)
	}
	if err != nil {
		return err
	}
	switch s := join.Where.(type) {
	case ast.Binary:
		w.WriteBlank()
		w.WriteKeyword("ON")
		w.WriteBlank()
		err = w.compact(func() error {
			return w.formatBinary(s, false)
		})
	case ast.List:
		w.WriteBlank()
		w.WriteKeyword("USING")
		w.WriteBlank()
		err = w.formatList(s)
	default:
		return w.CanNotUse("from", s)
	}
	return err
}

func (w *Writer) FormatFrom(list []ast.Statement) error {
	w.WriteKeyword("FROM")
	w.WriteBlank()

	withComma := func(stmt ast.Statement) bool {
		_, ok := stmt.(ast.Join)
		return !ok
	}

	var err error
	for i, s := range list {
		if withComma(s) && i > 0 {
			w.WriteString(",")
		}
		if i > 0 {
			w.WriteNL()
		}
		switch s := s.(type) {
		case ast.Name:
			w.FormatName(s)
		case ast.Alias:
			err = w.FormatAlias(s)
		case ast.Join:
			err = w.formatFromJoin(s)
		case ast.Row:
			err = w.FormatRow(s, true)
		case ast.SelectStatement:
			w.WriteString("(")
			w.WriteNL()
			err = w.FormatStatement(s)
			if err == nil {
				w.WriteString(")")
			}
		default:
			err = w.CanNotUse("from", s)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatGroupBy(groups []ast.Statement) error {
	if len(groups) == 0 {
		return nil
	}
	w.WriteKeyword("GROUP BY")
	w.WriteBlank()
	for i, s := range groups {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		n, ok := s.(ast.Name)
		if !ok {
			return w.CanNotUse("group by", s)
		}
		w.FormatName(n)
	}
	return nil
}

func (w *Writer) FormatWindows(windows []ast.Statement) error {
	w.WriteKeyword("WINDOW")

	if len(windows) > 1 {
		w.WriteNL()
	} else {
		w.WriteBlank()
	}

	for i, c := range windows {
		def, ok := c.(ast.WindowDefinition)
		if !ok {
			return fmt.Errorf("window: unexpected statement type %T", c)
		}
		if i > 0 {
			w.WriteString(",")
			w.WriteNL()
		}
		if err := w.FormatExpr(def.Ident, false); err != nil {
			return err
		}
		w.WriteBlank()
		w.WriteKeyword("AS")
		w.WriteBlank()
		w.WriteString("(")
		win, ok := def.Window.(ast.Window)
		if !ok {
			return fmt.Errorf("window: unexpected statement type %T", def.Window)
		}
		if win.Ident != nil {
			if err := w.FormatExpr(win.Ident, false); err != nil {
				return err
			}
			w.WriteBlank()
		}
		if win.Ident == nil && len(win.Partitions) > 0 {
			w.WriteKeyword("PARTITION BY")
			w.WriteBlank()
			if err := w.formatStmtSlice(win.Partitions); err != nil {
				return err
			}
		}
		if len(win.Orders) > 0 {
			w.WriteBlank()
			w.WriteKeyword("ORDER BY")
			w.WriteBlank()
			for i, s := range win.Orders {
				if i > 0 {
					w.WriteString(",")
					w.WriteBlank()
				}
				order, ok := s.(ast.Order)
				if !ok {
					return w.CanNotUse("order by", s)
				}
				if err := w.formatOrder(order); err != nil {
					return err
				}
			}
		}
		w.WriteString(")")
	}
	return nil
}

func (w *Writer) FormatHaving(having ast.Statement) error {
	if having == nil {
		return nil
	}
	w.WriteKeyword("HAVING")
	w.WriteBlank()
	return w.FormatExpr(having, true)
}

func (w *Writer) FormatOrderBy(orders []ast.Statement) error {
	if len(orders) == 0 {
		return nil
	}
	w.WriteKeyword("ORDER BY")
	w.WriteBlank()
	for i, s := range orders {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		order, ok := s.(ast.Order)
		if !ok {
			return w.CanNotUse("order by", s)
		}
		if err := w.formatOrder(order); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) formatOrder(order ast.Order) error {
	n, ok := order.Statement.(ast.Name)
	if !ok {
		return w.CanNotUse("order by", order.Statement)
	}
	w.FormatName(n)
	switch order.Dir {
	case 0:
	case ast.AscOrder:
		w.WriteBlank()
		w.WriteKeyword("ASC")
	case ast.DescOrder:
		w.WriteBlank()
		w.WriteKeyword("DESC")
	default:
		return fmt.Errorf("invalid order direction")
	}
	if order.Nulls != "" {
		w.WriteBlank()
		w.WriteKeyword("NULLS")
		w.WriteBlank()
		w.WriteKeyword(order.Nulls)
	}
	return nil
}

func (w *Writer) FormatLimit(limit ast.Statement) error {
	if limit == nil {
		return nil
	}
	lim, ok := limit.(ast.Limit)
	if !ok {
		return w.FormatOffset(limit)
	}
	w.WriteKeyword("LIMIT")
	w.WriteBlank()
	w.WriteString(strconv.Itoa(lim.Count))
	if lim.Offset > 0 {
		w.WriteBlank()
		w.WriteKeyword("OFFSET")
		w.WriteBlank()
		w.WriteString(strconv.Itoa(lim.Offset))
	}
	return nil
}

func (w *Writer) FormatOffset(limit ast.Statement) error {
	lim, ok := limit.(ast.Offset)
	if !ok {
		return w.CanNotUse("fetch", limit)
	}
	if lim.Offset > 0 {
		w.WriteKeyword("OFFSET")
		w.WriteBlank()
		w.WriteString(strconv.Itoa(lim.Offset))
		w.WriteBlank()
		w.WriteKeyword("ROWS")
		w.WriteBlank()
	}
	w.WriteKeyword("FETCH")
	w.WriteBlank()
	if lim.Next {
		w.WriteKeyword("NEXT")
	} else {
		w.WriteKeyword("FIRST")
	}
	w.WriteBlank()
	w.WriteString(strconv.Itoa(lim.Count))
	w.WriteBlank()
	w.WriteKeyword("ROWS ONLY")
	return nil
}

func (w *Writer) FormatWith(stmt ast.WithStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	if stmt.Recursive {
		w.WriteBlank()
		w.WriteString("RECURSIVE")
	}
	w.WriteNL()

	for i, q := range stmt.Queries {
		if i > 0 {
			w.WriteString(",")
			w.WriteNL()
		}
		if err := w.FormatStatement(q); err != nil {
			return err
		}
	}
	w.WriteNL()
	return w.FormatStatement(stmt.Statement)
}

func (w *Writer) FormatCte(stmt ast.CteStatement) error {
	ident := stmt.Ident
	if w.Upperize.Identifier() {
		ident = strings.ToUpper(ident)
	}
	w.WriteString(ident)
	if len(stmt.Columns) > 0 {
		w.WriteString("(")
		for i, s := range stmt.Columns {
			if i > 0 {
				w.WriteString(",")
				if !w.Compact {
					w.WriteBlank()
				}
			}
			if w.Upperize.Identifier() {
				s = strings.ToUpper(s)
			}
			if w.UseQuote {
				s = w.Quote(s)
			}
			w.WriteString(s)
		}
		w.WriteString(")")
	}
	w.WriteBlank()
	w.WriteKeyword("AS")
	w.WriteBlank()
	w.WriteString("(")
	w.WriteNL()

	w.Enter()
	if err := w.FormatStatement(stmt.Statement); err != nil {
		return err
	}
	w.Leave()
	w.WriteNL()
	w.WriteString(")")
	return nil
}
