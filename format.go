package sweet

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
	Indent  string
	prefix  int
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		inner:  bufio.NewWriter(w),
		Indent: "  ",
	}
}

func WriteAnsi(r io.Reader, w io.Writer) error {
	ws := NewWriter(w)
	return ws.Format(r, AnsiKeywords())
}

func (w *Writer) Format(r io.Reader, keywords KeywordSet) error {
	p, err := NewParser(r, keywords)
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

func (w *Writer) FormatStatement(stmt Statement) error {
	return w.format(stmt)
}

func (w *Writer) format(stmt Statement) error {
	defer w.inner.Flush()

	w.prefix = -1
	err := w.formatStatement(stmt)
	if err == nil {
		w.writeString(";")
		w.writeNL()
	}
	return err
}

func (w *Writer) formatStatement(stmt Statement) error {
	var err error
	switch stmt := stmt.(type) {
	case SelectStatement:
		err = w.formatSelect(stmt)
	case UnionStatement:
		err = w.formatUnion(stmt)
	case IntersectStatement:
		err = w.formatIntersect(stmt)
	case ExceptStatement:
		err = w.formatExcept(stmt)
	case InsertStatement:
		err = w.formatInsert(stmt)
	case UpdateStatement:
		err = w.formatUpdate(stmt)
	case DeleteStatement:
		err = w.formatDelete(stmt)
	case WithStatement:
	case CteStatement:
	default:
		err = fmt.Errorf("unsupported statement type %T", stmt)
	}
	return err
}

func (w *Writer) formatUnion(stmt UnionStatement) error {
	if err := w.formatStatement(stmt.Left); err != nil {
		return err
	}
	w.writeNL()
	w.writeString("UNION")
	if stmt.All {
		w.writeBlank()
		w.writeString("ALL")
	}
	if stmt.Distinct {
		w.writeBlank()
		w.writeString("DISTINCT")
	}
	w.writeNL()
	return w.formatStatement(stmt.Right)
}

func (w *Writer) formatExcept(stmt ExceptStatement) error {
	if err := w.formatStatement(stmt.Left); err != nil {
		return err
	}
	w.writeNL()
	w.writeString("EXCEPT")
	if stmt.All {
		w.writeBlank()
		w.writeString("ALL")
	}
	if stmt.Distinct {
		w.writeBlank()
		w.writeString("DISTINCT")
	}
	w.writeNL()
	return w.formatStatement(stmt.Right)
}

func (w *Writer) formatIntersect(stmt IntersectStatement) error {
	if err := w.formatStatement(stmt.Left); err != nil {
		return err
	}
	w.writeNL()
	w.writeString("INTERSECT")
	if stmt.All {
		w.writeBlank()
		w.writeString("ALL")
	}
	if stmt.Distinct {
		w.writeBlank()
		w.writeString("DISTINCT")
	}
	w.writeNL()
	return w.formatStatement(stmt.Right)
}

func (w *Writer) formatDelete(stmt DeleteStatement) error {
	w.enter()
	defer w.leave()

	w.writeString(strings.Repeat(w.Indent, w.prefix))
	w.writeString("DELETE FROM")
	w.writeBlank()	
	w.writeString(stmt.Table)
	w.writeBlank()
	if err := w.formatWhere(stmt.Where); err != nil {
		return err
	}
	return nil
}

func (w *Writer) formatUpdate(stmt UpdateStatement) error {
	w.enter()
	defer w.leave()

	w.writeString(strings.Repeat(w.Indent, w.prefix))
	w.writeString("UPDATE")
	w.writeBlank()
	switch stmt := stmt.Table.(type) {
	case Name:
		w.formatName(stmt)
	case Alias:
		if err := w.formatAlias(stmt); err != nil {
			return err
		}
	default:
		return fmt.Errorf("update: unexpected expression type (%T)", stmt)
	}
	w.writeBlank()
	w.writeString("SET")
	w.writeNL()
	if err := w.formatUpdateList(stmt); err != nil {
		return err
	}

	if err := w.formatUpdateFrom(stmt); err != nil {
		return err
	}
	if err := w.formatUpdateWhere(stmt); err != nil {
		return err
	}
	return nil
}

func (w *Writer) formatUpdateList(stmt UpdateStatement) error {
	w.enter()
	defer w.leave()

	var err error
	for i, s := range stmt.List {
		if i > 0 {
			w.writeString(",")
			w.writeBlank()
		}
		ass, ok := s.(Assignment)
		if !ok {
			return fmt.Errorf("update: unexpected expression type (%T)", s)
		}
		w.writeString(strings.Repeat(w.Indent, w.prefix))
		switch field := ass.Field.(type) {
		case Name:
			w.formatName(field)
		case List:
			err = w.formatList(field)
		default:
			err = fmt.Errorf("update: unexpected expression type (%T)", s)
		}
		if err != nil {
			return err
		}
		w.writeString("=")
		switch value := ass.Value.(type) {
		case List:
			err = w.formatList(value)
		default:
			err = w.formatExpr(value, false)
		}
		if err != nil {
			return err
		}
	}
	return err
}

func (w *Writer) formatUpdateFrom(stmt UpdateStatement) error {
	if len(stmt.Tables) == 0 {
		return nil
	}
	return w.formatFrom(stmt.Tables)
}

func (w *Writer) formatUpdateWhere(stmt UpdateStatement) error {
	return w.formatWhere(stmt.Where)
}

func (w *Writer) formatInsert(stmt InsertStatement) error {
	w.enter()
	defer w.leave()

	w.writeString(strings.Repeat(w.Indent, w.prefix))
	w.writeString("INSERT INTO")
	w.writeBlank()
	w.writeString(stmt.Table)
	w.writeBlank()
	if len(stmt.Columns) > 0 {
		w.writeString("(")
		for i, c := range stmt.Columns {
			if i > 0 {
				w.writeString(",")
				w.writeBlank()
			}
			w.writeString(c)
		}
		w.writeString(")")
	}
	w.writeBlank()
	w.writeString("VALUES")

	w.enter()
	defer w.leave()

	var err error
	switch stmt := stmt.Values.(type) {
	case List:
		w.writeBlank()
		w.writeNL()
		for i, v := range stmt.Values {
			if i > 0 {
				w.writeString(",")
				w.writeNL()
			}
			w.writeString(strings.Repeat(w.Indent, w.prefix))
			if err := w.formatExpr(v, false); err != nil {
				return err
			}
		}
	case SelectStatement:
		w.writeNL()
		err = w.formatSelect(stmt)
	}
	return err
}

func (w *Writer) formatSelect(stmt SelectStatement) error {
	w.enter()
	defer w.leave()

	w.writeString(strings.Repeat(w.Indent, w.prefix))
	w.writeString("SELECT")
	if err := w.formatSelectColumns(stmt); err != nil {
		return err
	}
	if err := w.formatSelectFrom(stmt); err != nil {
		return err
	}
	if err := w.formatSelectWhere(stmt); err != nil {
		return err
	}
	if err := w.formatSelectGroupBy(stmt); err != nil {
		return err
	}
	if err := w.formatSelectHaving(stmt); err != nil {
		return err
	}
	if err := w.formatSelectOrderBy(stmt); err != nil {
		return err
	}
	if err := w.formatSelectLimit(stmt); err != nil {
		return nil
	}
	return nil
}

func (w *Writer) formatSelectFrom(stmt SelectStatement) error {
	return w.formatFrom(stmt.Tables)
}

func (w *Writer) formatSelectWhere(stmt SelectStatement) error {
	return w.formatWhere(stmt.Where)
}

func (w *Writer) formatSelectColumns(stmt SelectStatement) error {
	w.enter()
	defer w.leave()

	var (
		err    error
		prefix = strings.Repeat(w.Indent, w.prefix)
	)
	w.writeNL()
	defer w.writeNL()
	for i, s := range stmt.Columns {
		if i > 0 {
			w.writeString(",")
			w.writeNL()
		}
		w.writeString(prefix)
		switch s := s.(type) {
		case Value:
			w.writeString(s.Literal)
		case Name:
			w.formatName(s)
		case Alias:
			err = w.formatAlias(s)
		case Call, Binary, Unary:
			err = w.formatExpr(s, false)
		case CaseStatement:
			err = w.formatCase(s)
		default:
			err = fmt.Errorf("select: unsupported expression type in columns (%T)", s)
		}
		if err != nil {
			break
		}
	}
	return err
}

func (w *Writer) formatFrom(list []Statement) error {
	w.writeString(strings.Repeat(w.Indent, w.prefix))
	w.writeString("FROM")
	w.writeBlank()

	w.enter()
	defer w.leave()

	var (
		err    error
		prefix = strings.Repeat(w.Indent, w.prefix)
	)
	for i, s := range list {
		if i > 0 {
			w.writeNL()
			w.writeString(prefix)
		}
		switch s := s.(type) {
		case Name:
			w.formatName(s)
		case Alias:
			err = w.formatAlias(s)
		case Join:
			err = w.formatFromJoin(s)
		case SelectStatement:
		default:
			err = fmt.Errorf("from: unsupported statement (%T)", s)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) formatSelectGroupBy(stmt SelectStatement) error {
	if len(stmt.Groups) == 0 {
		return nil
	}
	w.writeNL()
	w.writeString("GROUP BY")
	w.writeBlank()
	for i, s := range stmt.Groups {
		if i > 0 {
			w.writeString(",")
			w.writeBlank()
		}
		n, ok := s.(Name)
		if !ok {
			return fmt.Errorf("group by: unexpected expression type (%T)", s)
		}
		w.formatName(n)
	}
	return nil
}

func (w *Writer) formatSelectHaving(stmt SelectStatement) error {
	w.enter()
	defer w.leave()

	if stmt.Having == nil {
		return nil
	}
	w.writeNL()
	w.writeString("HAVING")
	w.writeBlank()
	return w.formatExpr(stmt.Having, true)
}

func (w *Writer) formatSelectOrderBy(stmt SelectStatement) error {
	if len(stmt.Orders) == 0 {
		return nil
	}
	w.writeNL()
	w.writeString("ORDER BY")
	w.writeBlank()
	for i, s := range stmt.Orders {
		if i > 0 {
			w.writeString(",")
			w.writeBlank()
		}
		order, ok := s.(Order)
		if !ok {
			return fmt.Errorf("order by: unexpected statement type (%T)", s)
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
		return fmt.Errorf("order by: unexpected statement type (%T)", order.Statement)
	}
	w.formatName(n)
	if order.Orient != "" {
		w.writeBlank()
		w.writeString(order.Orient)
	}
	if order.Nulls != "" {
		w.writeBlank()
		w.writeString("NULLS")
		w.writeBlank()
		w.writeString(order.Nulls)
	}
	return nil
}

func (w *Writer) formatSelectLimit(stmt SelectStatement) error {
	if stmt.Limit == nil {
		return nil
	}
	lim, ok := stmt.Limit.(Limit)
	if !ok {
		return fmt.Errorf("limit: unexpected statement type (%T)", stmt.Limit)
	}
	w.writeNL()
	w.writeString("LIMIT")
	w.writeBlank()
	w.writeString(strconv.Itoa(lim.Count))
	if lim.Offset > 0 {
		w.writeBlank()
		w.writeString("OFFSET")
		w.writeBlank()
		w.writeString(strconv.Itoa(lim.Offset))
	}
	return nil
}

func (w *Writer) formatWhere(stmt Statement) error {
	if stmt == nil {
		return nil
	}

	w.writeNL()
	w.writeString(strings.Repeat(w.Indent, w.prefix))
	w.writeString("WHERE")
	w.writeBlank()

	w.enter()
	defer w.leave()

	return w.formatExpr(stmt, true)
}

func (w *Writer) formatFromJoin(join Join) error {
	w.writeString(join.Type)
	w.writeBlank()

	var err error
	switch s := join.Table.(type) {
	case Name:
		w.formatName(s)
	case Alias:
		err = w.formatAlias(s)
	case SelectStatement:
		w.writeString("(")
		err = w.formatSelect(s)
		w.writeString(")")
	default:
		err = fmt.Errorf("from: unsupported statement (%T)", s)
	}
	if err != nil {
		return err
	}
	switch s := join.Where.(type) {
	case Binary:
		w.writeBlank()
		w.writeString("ON")
		w.writeBlank()
		err = w.formatBinary(s, false)
	case List:
		w.writeBlank()
		w.writeString("USING")
		w.writeBlank()
		err = w.formatList(s)
	default:
		err = fmt.Errorf("from: unsupported statement in on/using statement (%T)", s)
	}
	return err
}

func (w *Writer) formatCase(stmt CaseStatement) error {
	// w.writeString(strings.Repeat(w.Indent, w.prefix))
	w.writeString("CASE")
	if stmt.Cdt != nil {
		w.writeBlank()
		w.formatExpr(stmt.Cdt, false)
	}
	w.writeBlank()
	w.enter()
	for _, s := range stmt.Body {
		w.writeNL()
		if err := w.formatExpr(s, false); err != nil {
			return err
		}
	}
	if stmt.Else != nil {
		w.writeNL()
		w.writeString(strings.Repeat(w.Indent, w.prefix))
		w.writeString("ELSE")
		w.writeBlank()
		if err := w.formatExpr(stmt.Else, false); err != nil {
			return err
		}
	}
	w.leave()
	w.writeNL()
	w.writeString(strings.Repeat(w.Indent, w.prefix))
	w.writeString("END")
	return nil
}

func (w *Writer) formatWhen(stmt WhenStatement) error {
	w.writeString(strings.Repeat(w.Indent, w.prefix))
	w.writeString("WHEN")
	w.writeBlank()
	if err := w.formatExpr(stmt.Cdt, false); err != nil {
		return err
	}
	w.writeBlank()
	w.writeString("THEN")
	w.writeBlank()
	return w.formatExpr(stmt.Body, false)
}

func (w *Writer) formatExpr(stmt Statement, nl bool) error {
	var err error
	switch stmt := stmt.(type) {
	case Name:
		w.formatName(stmt)
	case Value:
		w.writeQuoted(stmt.Literal)
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
	case CaseStatement:
		err = w.formatCase(stmt)
	case WhenStatement:
		err = w.formatWhen(stmt)
	default:
		err = fmt.Errorf("unexpected expression type (%T)", stmt)
	}
	return err
}

func (w *Writer) formatList(stmt List) error {
	w.writeString("(")
	for i, v := range stmt.Values {
		if i > 0 {
			w.writeString(",")
			w.writeBlank()
		}
		if err := w.formatExpr(v, false); err != nil {
			return err
		}
	}
	w.writeString(")")
	return nil
}

func (w *Writer) formatCall(call Call) error {
	n, ok := call.Ident.(Name)
	if !ok {
		return fmt.Errorf("call: unexpected expression type (%T)", call.Ident)
	}
	w.writeString(n.Ident)
	w.writeString("(")
	for i, s := range call.Args {
		if i > 0 {
			w.writeString(",")
			w.writeBlank()
		}
		if err := w.formatExpr(s, false); err != nil {
			return err
		}
	}
	w.writeString(")")
	return nil
}

func (w *Writer) formatBetween(stmt Between, nl bool) error {
	if err := w.formatExpr(stmt.Ident, nl); err != nil {
		return err
	}
	w.writeBlank()
	w.writeString("BETWEEN")
	w.writeBlank()
	if err := w.formatExpr(stmt.Lower, false); err != nil {
		return err
	}
	w.writeBlank()
	w.writeString("AND")
	w.writeBlank()
	return w.formatExpr(stmt.Upper, false)
}

func (w *Writer) formatUnary(stmt Unary, nl bool) error {
	w.writeString(stmt.Op)
	return w.formatExpr(stmt.Right, nl)
}

func (w *Writer) formatBinary(stmt Binary, nl bool) error {
	if err := w.formatExpr(stmt.Left, nl); err != nil {
		return err
	}
	if nl && (stmt.Op == "AND" || stmt.Op == "OR") {
		w.writeNL()
		w.writeString(strings.Repeat(w.Indent, w.prefix))
	} else {
		w.writeBlank()
	}
	w.writeString(stmt.Op)
	w.writeBlank()
	if err := w.formatExpr(stmt.Right, nl); err != nil {
		return err
	}
	return nil
}

func (w *Writer) formatName(name Name) {
	if name.Prefix != "" {
		w.writeString(name.Prefix)
		w.writeString(".")
	}
	w.writeString(name.Ident)
}

func (w *Writer) formatAlias(alias Alias) error {
	var err error
	switch s := alias.Statement.(type) {
	case Name:
		w.formatName(s)
	case Call:
		err = w.formatCall(s)
	case CaseStatement:
		err = w.formatCase(s)
	case SelectStatement:
		w.writeString("(")
		w.writeNL()
		err = w.formatSelect(s)
		if err != nil {
			break
		}
		w.writeNL()
		w.writeString(strings.Repeat(w.Indent, w.prefix))
		w.writeString(")")
	default:
		return fmt.Errorf("alias: unsupported expression type used with alias (%T)", s)
	}
	if err != nil {
		return err
	}
	w.writeBlank()
	w.writeString("AS")
	w.writeBlank()
	w.writeString(alias.Alias)
	return nil
}

func (w *Writer) enter() {
	if w.Compact {
		return
	}
	w.prefix++
}

func (w *Writer) leave() {
	if w.Compact {
		return
	}
	w.prefix--
}

func (w *Writer) writeString(str string) {
	if w.Compact && str == "\n" {
		str = " "
	}
	w.inner.WriteString(str)
}

func (w *Writer) writeQuoted(str string) {
	w.inner.WriteRune('\'')
	w.writeString(str)
	w.inner.WriteRune('\'')
}

func (w *Writer) writeNL() {
	if w.Compact {
		return
	}
	w.inner.WriteRune('\n')
}

func (w *Writer) writeBlank() {
	w.inner.WriteRune(' ')
}
