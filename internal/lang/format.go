package lang

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Writer struct {
	inner   *bufio.Writer
	Compact bool
	KwUpper bool
	FnUpper bool
	Indent  string

	prefix int
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		inner:  bufio.NewWriter(w),
		Indent: "  ",
	}
}

func (w *Writer) SetIndent(indent string) {
	w.Indent = indent
}

func (w *Writer) SetCompact(compact bool) {
	w.Compact = compact
}

func (w *Writer) SetKeywordUppercase(upper bool) {
	w.KwUpper = upper
}

func (w *Writer) SetFunctionUppercase(upper bool) {
	w.FnUpper = upper
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

// func (w *Writer) FormatStatement(stmt Statement) error {
// 	return w.format(stmt)
// }

func (w *Writer) startStatement(stmt Statement) error {
	defer w.Flush()

	w.Reset()
	err := w.FormatStatement(stmt)
	if err == nil {
		w.WriteEOL()
	}
	return err
}

func (w *Writer) FormatStatement(stmt Statement) error {
	var err error
	switch stmt := stmt.(type) {
	case SelectStatement:
		err = w.FormatSelect(stmt)
	case ValuesStatement:
		err = w.FormatValues(stmt)
	case UnionStatement:
		err = w.FormatUnion(stmt)
	case IntersectStatement:
		err = w.FormatIntersect(stmt)
	case ExceptStatement:
		err = w.FormatExcept(stmt)
	case InsertStatement:
		err = w.FormatInsert(stmt)
	case UpdateStatement:
		err = w.FormatUpdate(stmt)
	case DeleteStatement:
		err = w.FormatDelete(stmt)
	case WithStatement:
		err = w.FormatWith(stmt)
	case CteStatement:
		err = w.FormatCte(stmt)
	case Commit:
		err = w.FormatCommit(stmt)
	case Rollback:
		err = w.FormatRollback(stmt)
	case StartTransaction:
		err = w.FormatStartTransaction(stmt)
	case SetTransaction:
		err = w.FormatSetTransaction(stmt)
	case Savepoint:
		err = w.FormatSavepoint(stmt)
	case ReleaseSavepoint:
		err = w.FormatReleaseSavepoint(stmt)
	case RollbackSavepoint:
		err = w.FormatRollbackSavepoint(stmt)
	case List:
		err = w.FormatBody(stmt)
	default:
		err = fmt.Errorf("unsupported statement type %T", stmt)
	}
	return err
}

func (w *Writer) FormatStartTransaction(stmt StartTransaction) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Mode > 0 {
		w.WriteBlank()
		switch stmt.Mode {
		case ModeReadWrite:
			w.WriteKeyword("READ WRITE")
		case ModeReadOnly:
			w.WriteKeyword("READ ONLY")
		default:
			return fmt.Errorf("unknown transaction mode")
		}
	}
	if stmt.Body != nil {
		w.WriteNL()
		if err := w.FormatStatement(stmt.Body); err != nil {
			return err
		}
	}
	if stmt.End == nil {
		return nil
	}
	return w.FormatStatement(stmt.End)
}

func (w *Writer) FormatBody(list List) error {
	w.Enter()
	defer w.Leave()
	for _, v := range list.Values {
		if err := w.FormatStatement(v); err != nil {
			return err
		}
		w.WriteEOL()
	}
	return nil
}

func (w *Writer) FormatSetTransaction(stmt SetTransaction) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Level > 0 {
		w.WriteBlank()
		w.WriteKeyword("ISOLATION LEVEL")
		w.WriteBlank()
	}
	if stmt.Mode > 0 {
		w.WriteBlank()
		switch stmt.Mode {
		case ModeReadWrite:
			w.WriteKeyword("READ WRITE")
		case ModeReadOnly:
			w.WriteKeyword("READ ONLY")
		default:
			return fmt.Errorf("unknown transaction mode")
		}
	}
	w.WriteBlank()
	return nil
}

func (w *Writer) FormatSavepoint(stmt Savepoint) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Name != "" {
		w.WriteBlank()
		w.WriteString(stmt.Name)
	}
	return nil
}

func (w *Writer) FormatReleaseSavepoint(stmt ReleaseSavepoint) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Name != "" {
		w.WriteBlank()
		w.WriteString(stmt.Name)
	}
	return nil
}

func (w *Writer) FormatRollbackSavepoint(stmt RollbackSavepoint) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Name != "" {
		w.WriteBlank()
		w.WriteString(stmt.Name)
	}
	return nil
}

func (w *Writer) FormatCommit(stmt Commit) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	return nil
}

func (w *Writer) FormatRollback(stmt Rollback) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	return nil
}

func (w *Writer) FormatWith(stmt WithStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
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

func (w *Writer) FormatCte(stmt CteStatement) error {
	w.Enter()
	defer w.Leave()

	w.WritePrefix()
	w.WriteString(stmt.Ident)
	if len(stmt.Columns) > 0 {
		w.WriteString("(")
		for i, s := range stmt.Columns {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
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
	if err := w.FormatStatement(stmt.Statement); err != nil {
		return err
	}
	w.WriteNL()
	w.WriteString(")")
	return nil
}

func (w *Writer) FormatUnion(stmt UnionStatement) error {
	if err := w.FormatStatement(stmt.Left); err != nil {
		return err
	}
	w.WriteNL()
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
	return w.FormatStatement(stmt.Right)
}

func (w *Writer) FormatExcept(stmt ExceptStatement) error {
	if err := w.FormatStatement(stmt.Left); err != nil {
		return err
	}
	w.WriteNL()
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
	return w.FormatStatement(stmt.Right)
}

func (w *Writer) FormatIntersect(stmt IntersectStatement) error {
	if err := w.FormatStatement(stmt.Left); err != nil {
		return err
	}
	w.WriteNL()
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
	return w.FormatStatement(stmt.Right)
}

func (w *Writer) FormatDelete(stmt DeleteStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	w.WriteString(stmt.Table)
	if stmt.Where != nil {
		w.WriteNL()
		if err := w.FormatWhere(stmt.Where); err != nil {
			return err
		}
	}
	if stmt.Return != nil {
		w.WriteNL()
		if err := w.FormatReturn(stmt.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatUpdate(stmt UpdateStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	switch stmt := stmt.Table.(type) {
	case Name:
		w.FormatName(stmt)
	case Alias:
		if err := w.FormatAlias(stmt); err != nil {
			return err
		}
	default:
		return w.CanNotUse("update", stmt)
	}
	w.WriteBlank()
	w.WriteKeyword("SET")
	w.WriteNL()

	if err := w.FormatAssignment(stmt.List); err != nil {
		return err
	}

	if len(stmt.Tables) > 0 {
		w.WriteNL()
		if err := w.FormatFrom(stmt.Tables); err != nil {
			return err
		}
	}
	if stmt.Where != nil {
		w.WriteNL()
		if err := w.FormatWhere(stmt.Where); err != nil {
			return err
		}
	}
	if stmt.Return != nil {
		w.WriteNL()
		if err := w.FormatReturn(stmt.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatInsert(stmt InsertStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Table, false); err != nil {
		return err
	}
	if len(stmt.Columns) > 0 {
		w.WriteBlank()
		w.WriteString("(")
		for i, c := range stmt.Columns {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteString(c)
		}
		w.WriteString(")")
	}
	w.WriteBlank()
	if err := w.FormatInsertValues(stmt.Values); err != nil {
		return err
	}
	if stmt.Upsert != nil {
		w.WriteNL()
		if err := w.FormatUpsert(stmt.Upsert); err != nil {
			return err
		}
	}
	if stmt.Return != nil {
		w.WriteNL()
		if err := w.FormatReturn(stmt.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatInsertValues(values Statement) error {
	if values == nil {
		return nil
	}
	var err error
	switch stmt := values.(type) {
	case List:
		w.WriteKeyword("VALUES")
		w.WriteNL()

		w.Enter()
		defer w.Leave()
		for i, v := range stmt.Values {
			if i > 0 {
				w.WriteString(",")
				w.WriteNL()
			}
			w.WritePrefix()
			if err = w.FormatExpr(v, false); err != nil {
				break
			}
		}
	case SelectStatement:
		w.WriteNL()
		err = w.FormatSelect(stmt)
	default:
		err = fmt.Errorf("values: unexpected statement type(%T)", values)
	}
	return err
}

func (w *Writer) FormatUpsert(stmt Statement) error {
	if stmt == nil {
		return nil
	}
	upsert, ok := stmt.(UpsertStatement)
	if !ok {
		return w.CanNotUse("insert(upsert)", stmt)
	}
	w.WriteStatement("ON CONFLICT")
	w.WriteBlank()

	if len(upsert.Columns) > 0 {
		w.WriteString("(")
		for i, s := range upsert.Columns {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteString(s)
		}
		w.WriteString(")")
	}
	w.WriteBlank()
	if len(upsert.List) == 0 {
		w.WriteKeyword("DO NOTHING")
		return nil
	}
	w.WriteKeyword("UPDATE SET")
	w.WriteNL()
	if err := w.FormatAssignment(upsert.List); err != nil {
		return err
	}
	return w.FormatWhere(upsert.Where)
}

func (w *Writer) FormatValues(stmt ValuesStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	return w.formatStmtSlice(stmt.List)
}

func (w *Writer) FormatSelect(stmt SelectStatement) error {
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
	if len(stmt.Windows) > 0 {
		w.WriteNL()
		if err := w.FormatWindows(stmt.Windows); err != nil {
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

func (w *Writer) FormatSelectColumns(columns []Statement) error {
	w.Enter()
	defer w.Leave()
	for i, s := range columns {
		if i > 0 {
			w.WriteString(",")
			w.WriteNL()
		}
		w.WritePrefix()
		if err := w.FormatExpr(s, false); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatFrom(list []Statement) error {
	w.WriteStatement("FROM")
	w.WriteBlank()

	w.Enter()
	defer w.Leave()

	var err error
	for i, s := range list {
		if i > 0 {
			w.WriteNL()
			w.WritePrefix()
		}
		switch s := s.(type) {
		case Name:
			w.FormatName(s)
		case Alias:
			err = w.FormatAlias(s)
		case Join:
			err = w.formatFromJoin(s)
		case SelectStatement:
			w.WriteString("(")
			err = w.FormatStatement(s)
			if err == nil {
				w.WriteNL()
				w.WriteString(")")
				w.WriteNL()
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

func (w *Writer) FormatGroupBy(groups []Statement) error {
	if len(groups) == 0 {
		return nil
	}
	w.WriteStatement("GROUP BY")
	w.WriteBlank()
	for i, s := range groups {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		n, ok := s.(Name)
		if !ok {
			return w.CanNotUse("group by", s)
		}
		w.FormatName(n)
	}
	return nil
}

func (w *Writer) FormatWindows(windows []Statement) error {
	w.WriteStatement("WINDOW")

	w.Enter()
	defer w.Leave()

	if len(windows) > 1 {
		w.WriteNL()
		w.WritePrefix()
	} else {
		w.WriteBlank()
	}

	for i, c := range windows {
		def, ok := c.(WindowDefinition)
		if !ok {
			return fmt.Errorf("window: unexpected statement type %T", c)
		}
		if i > 0 {
			w.WriteString(",")
			w.WriteNL()
			w.WritePrefix()
		}
		if err := w.FormatExpr(def.Ident, false); err != nil {
			return err
		}
		w.WriteBlank()
		w.WriteKeyword("AS")
		w.WriteBlank()
		w.WriteString("(")
		win, ok := def.Window.(Window)
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
				order, ok := s.(Order)
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

func (w *Writer) FormatHaving(having Statement) error {
	w.Enter()
	defer w.Leave()

	if having == nil {
		return nil
	}
	w.WriteStatement("HAVING")
	w.WriteBlank()
	return w.FormatExpr(having, true)
}

func (w *Writer) FormatOrderBy(orders []Statement) error {
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
			return w.CanNotUse("order by", s)
		}
		if err := w.formatOrder(order); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) formatOrder(order Order) error {
	n, ok := order.Statement.(Name)
	if !ok {
		return w.CanNotUse("order by", order.Statement)
	}
	w.FormatName(n)
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

func (w *Writer) FormatLimit(limit Statement) error {
	if limit == nil {
		return nil
	}
	lim, ok := limit.(Limit)
	if !ok {
		return w.FormatOffset(limit)
	}
	w.WriteStatement("LIMIT")
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

func (w *Writer) FormatOffset(limit Statement) error {
	lim, ok := limit.(Offset)
	if !ok {
		return w.CanNotUse("fetch", limit)
	}
	w.WritePrefix()
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

func (w *Writer) FormatAssignment(list []Statement) error {
	w.Enter()
	defer w.Leave()

	var err error
	for i, s := range list {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		ass, ok := s.(Assignment)
		if !ok {
			return w.CanNotUse("assignment", s)
		}
		w.WritePrefix()
		switch field := ass.Field.(type) {
		case Name:
			w.FormatName(field)
		case List:
			err = w.formatList(field)
		default:
			return w.CanNotUse("assignment", s)
		}
		if err != nil {
			return err
		}
		w.WriteString("=")
		switch value := ass.Value.(type) {
		case List:
			err = w.formatList(value)
		default:
			err = w.FormatExpr(value, false)
		}
		if err != nil {
			return err
		}
	}
	return err
}

func (w *Writer) FormatReturn(stmt Statement) error {
	if stmt == nil {
		return nil
	}
	w.WriteStatement("RETURNING")
	w.WriteBlank()

	list, ok := stmt.(List)
	if !ok {
		return w.FormatExpr(stmt, false)
	}
	return w.formatStmtSlice(list.Values)
}

func (w *Writer) FormatWhere(stmt Statement) error {
	if stmt == nil {
		return nil
	}
	w.WriteStatement("WHERE")
	w.WriteBlank()

	w.Enter()
	defer w.Leave()

	return w.FormatExpr(stmt, true)
}

func (w *Writer) formatFromJoin(join Join) error {
	w.WriteString(join.Type)
	w.WriteBlank()

	var err error
	switch s := join.Table.(type) {
	case Name:
		w.FormatName(s)
	case Alias:
		err = w.FormatAlias(s)
	case SelectStatement:
		w.WriteString("(")
		err = w.FormatSelect(s)
		w.WriteString(")")
	default:
		return w.CanNotUse("from", s)
		err = fmt.Errorf("from: unsupported statement (%T)", s)
	}
	if err != nil {
		return err
	}
	switch s := join.Where.(type) {
	case Binary:
		w.WriteBlank()
		w.WriteKeyword("ON")
		w.WriteBlank()
		err = w.formatBinary(s, false)
	case List:
		w.WriteBlank()
		w.WriteKeyword("USING")
		w.WriteBlank()
		err = w.formatList(s)
	default:
		return w.CanNotUse("from", s)
	}
	return err
}

func (w *Writer) formatCase(stmt CaseStatement) error {
	w.WriteKeyword("CASE")
	if stmt.Cdt != nil {
		w.WriteBlank()
		w.FormatExpr(stmt.Cdt, false)
	}
	w.WriteBlank()
	w.Enter()
	for _, s := range stmt.Body {
		w.WriteNL()
		if err := w.FormatExpr(s, false); err != nil {
			return err
		}
	}
	if stmt.Else != nil {
		w.WriteNL()
		w.WriteStatement("ELSE")
		w.WriteBlank()
		if err := w.FormatExpr(stmt.Else, false); err != nil {
			return err
		}
	}
	w.Leave()
	w.WriteNL()
	w.WriteStatement("END")
	return nil
}

func (w *Writer) formatWhen(stmt WhenStatement) error {
	w.WriteStatement("WHEN")
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Cdt, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("THEN")
	w.WriteBlank()
	return w.FormatExpr(stmt.Body, false)
}

func (w *Writer) FormatExpr(stmt Statement, nl bool) error {
	var err error
	switch stmt := stmt.(type) {
	case Name:
		w.FormatName(stmt)
	case Value:
		w.formatValue(stmt.Literal)
	case Row:
		err = w.formatRow(stmt, nl)
	case Alias:
		err = w.FormatAlias(stmt)
	case Call:
		err = w.formatCall(stmt)
	case List:
		err = w.formatList(stmt)
	case Binary:
		err = w.formatBinary(stmt, nl)
	case Unary:
		err = w.formatUnary(stmt, nl)
	case Between:
		err = w.formatBetween(stmt, nl)
	case Collate:
		err = w.formatCollate(stmt, nl)
	case Cast:
		err = w.formatCast(stmt, nl)
	case Exists:
		err = w.formatExists(stmt, nl)
	case Not:
		err = w.formatNot(stmt, nl)
	case CaseStatement:
		err = w.formatCase(stmt)
	case WhenStatement:
		err = w.formatWhen(stmt)
	default:
		err = w.FormatStatement(stmt)
	}
	return err
}

func (w *Writer) formatRow(stmt Row, nl bool) error {
	w.WriteKeyword("ROW")
	w.WriteString("(")
	for i, v := range stmt.Values {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		if nl {
			w.WriteNL()
			w.WritePrefix()
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

func (w *Writer) formatNot(stmt Not, _ bool) error {
	return nil
}

func (w *Writer) formatExists(stmt Exists, _ bool) error {
	w.WriteKeyword("EXISTS")
	w.WriteString("(")
	if err := w.FormatExpr(stmt.Statement, false); err != nil {
		return err
	}
	w.WriteString(")")
	return nil
}

func (w *Writer) formatCast(stmt Cast, _ bool) error {
	w.WriteKeyword("CAST")
	w.WriteString("(")
	if err := w.FormatExpr(stmt.Ident, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("AS")
	w.WriteBlank()
	if err := w.formatType(stmt.Type); err != nil {
		return err
	}
	w.WriteString(")")
	return nil
}

func (w *Writer) formatType(dt Type) error {
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

func (w *Writer) formatCollate(stmt Collate, _ bool) error {
	if err := w.FormatExpr(stmt.Statement, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("COLLATE")
	w.WriteBlank()
	w.WriteString(stmt.Collation)
	return nil
}

func (w *Writer) formatStmtSlice(values []Statement) error {
	for i, v := range values {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		if err := w.FormatExpr(v, false); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) formatList(stmt List) error {
	w.WriteString("(")
	defer w.WriteString(")")
	return w.formatStmtSlice(stmt.Values)
}

func (w *Writer) formatCall(call Call) error {
	n, ok := call.Ident.(Name)
	if !ok {
		return w.CanNotUse("call", call.Ident)
	}
	w.WriteKeyword(n.Ident)
	w.WriteString("(")
	if call.Distinct {
		w.WriteKeyword("DISTINCT")
		w.WriteBlank()
	}
	if err := w.formatStmtSlice(call.Args); err != nil {
		return err
	}
	w.WriteString(")")
	if call.Filter != nil {
		w.WriteBlank()
		w.WriteKeyword("FILTER")
		w.WriteString("(")
		w.WriteKeyword("WHERE")
		w.WriteBlank()
		if err := w.FormatExpr(call.Filter, false); err != nil {
			return err
		}
		w.WriteString(")")
	}
	if call.Over != nil {
		w.WriteBlank()
		w.WriteKeyword("OVER")
		w.WriteBlank()
		switch over := call.Over.(type) {
		case Name:
			w.WriteBlank()
			return w.FormatExpr(over, false)
		case Window:
			w.WriteString("(")
			if over.Ident != nil {
				if err := w.FormatExpr(over.Ident, false); err != nil {
					return err
				}
			}
			if over.Ident == nil && len(over.Partitions) > 0 {
				w.WriteKeyword("PARTITION BY")
				w.WriteBlank()
				if err := w.formatStmtSlice(over.Partitions); err != nil {
					return err
				}
			}
			if len(over.Orders) > 0 {
				w.WriteBlank()
				w.WriteKeyword("ORDER BY")
				w.WriteBlank()
				for i, s := range over.Orders {
					if i > 0 {
						w.WriteString(",")
						w.WriteBlank()
					}
					o, ok := s.(Order)
					if !ok {
						return w.CanNotUse("over", s)
					}
					if err := w.formatOrder(o); err != nil {
						return err
					}
				}
			}
			w.WriteString(")")
		default:
			return fmt.Errorf("window: unsupported statement type %T", over)
		}
	}
	return nil
}

func (w *Writer) formatBetween(stmt Between, nl bool) error {
	if err := w.FormatExpr(stmt.Ident, nl); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("BETWEEN")
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Lower, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("AND")
	w.WriteBlank()
	return w.FormatExpr(stmt.Upper, false)
}

func (w *Writer) formatUnary(stmt Unary, nl bool) error {
	w.WriteString(stmt.Op)
	w.WriteBlank()
	return w.FormatExpr(stmt.Right, nl)
}

func (w *Writer) formatBinary(stmt Binary, nl bool) error {
	if err := w.FormatExpr(stmt.Left, nl); err != nil {
		return err
	}
	if nl && (stmt.Op == "AND" || stmt.Op == "OR") {
		w.WriteNL()
		w.WritePrefix()
	} else {
		w.WriteBlank()
	}
	w.WriteKeyword(stmt.Op)
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Right, nl); err != nil {
		return err
	}
	return nil
}

func (w *Writer) FormatName(name Name) {
	if name.Prefix != "" {
		w.WriteString(name.Prefix)
		w.WriteString(".")
	}
	w.WriteString(name.Ident)
}

func (w *Writer) FormatAlias(alias Alias) error {
	var err error
	switch s := alias.Statement.(type) {
	case Name:
		w.FormatName(s)
	case Call:
		err = w.formatCall(s)
	case CaseStatement:
		err = w.formatCase(s)
	case SelectStatement:
		w.WriteString("(")
		w.WriteNL()
		err = w.FormatSelect(s)
		if err != nil {
			break
		}
		w.WriteNL()
		w.WritePrefix()
		w.WriteString(")")
	default:
		return fmt.Errorf("alias: unsupported expression type used with alias (%T)", s)
	}
	if err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteString("AS")
	w.WriteBlank()
	w.WriteString(alias.Alias)
	return nil
}

func (w *Writer) formatValue(literal string) {
	if literal == "NULL" || literal == "DEFAULT" || literal == "*" {
		w.WriteKeyword(literal)
		return
	}
	if _, err := strconv.Atoi(literal); err == nil {
		w.WriteString(literal)
		return
	}
	w.WriteQuoted(literal)
}

func (w *Writer) Enter() {
	if w.Compact {
		return
	}
	w.prefix++
}

func (w *Writer) Leave() {
	if w.Compact {
		return
	}
	w.prefix--
}

func (w *Writer) WriteString(str string) {
	if w.Compact && str == "\n" {
		str = " "
	}
	w.inner.WriteString(str)
}

func (w *Writer) WriteEOL() {
	w.WriteString(";")
	w.WriteNL()
}

func (w *Writer) WriteQuoted(str string) {
	w.inner.WriteRune('\'')
	w.WriteString(str)
	w.inner.WriteRune('\'')
}

func (w *Writer) WriteNL() {
	if w.Compact {
		w.WriteBlank()
		return
	}
	w.inner.WriteRune('\n')
}

func (w *Writer) WriteBlank() {
	w.inner.WriteRune(' ')
}

func (w *Writer) WriteStatement(kw string) {
	w.WritePrefix()
	w.WriteKeyword(kw)
}

func (w *Writer) WriteKeyword(kw string) {
	if !w.KwUpper {
		kw = strings.ToLower(kw)
	} else {
		kw = strings.ToUpper(kw)
	}
	w.WriteString(kw)
}

func (w *Writer) WritePrefix() {
	if w.prefix <= 0 {
		return
	}
	w.WriteString(strings.Repeat(w.Indent, w.prefix))
}

func (w *Writer) Flush() {
	w.inner.Flush()
}

func (w *Writer) Reset() {
	w.prefix = -1
}

func (w *Writer) CanNotUse(ctx string, stmt Statement) error {
	return fmt.Errorf("%T can not be used as statement in %s", stmt, ctx)
}
