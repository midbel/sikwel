package sweet

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

type Writer struct {
	inner   *bufio.Writer
	compact bool
	prefix  int
	indent  string
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		inner:  bufio.NewWriter(w),
		indent: "  ",
		prefix: -1,
	}
}

func WriteAnsi(r io.Reader, w io.Writer) error {
	ws := NewWriter(w)
	return ws.Format(r, AnsiKeywords())
}

func (w *Writer) Format(r io.Reader, keywords KeywordSet) error {
	defer w.inner.Flush()

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

func (w *Writer) format(stmt Statement) error {
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
	case DeleteStatement:
	case UpdateStatement:
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

func (w *Writer) formatSelect(stmt SelectStatement) error {
	w.enter()
	defer w.leave()

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

func (w *Writer) formatSelectColumns(stmt SelectStatement) error {
	w.enter()
	defer w.leave()

	var (
		err    error
		prefix = strings.Repeat(w.indent, w.prefix)
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
		default:
			err = fmt.Errorf("select: unsupported expression type in columns (%T)", s)
		}
		if err != nil {
			break
		}
	}
	return err
}

func (w *Writer) formatSelectFrom(stmt SelectStatement) error {
	w.enter()
	defer w.leave()

	w.writeString("FROM")
	w.writeBlank()

	var (
		err    error
		prefix = strings.Repeat(w.indent, w.prefix)
	)
	for i, s := range stmt.Tables {
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
			err = fmt.Errorf("select: unsupported statement (%T)", s)
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
	if stmt.Limit == "" {
		return nil
	}
	w.writeNL()
	w.writeString("LIMIT")
	w.writeBlank()
	w.writeString(stmt.Limit)
	if stmt.Offset != "" {
		w.writeBlank()
		w.writeString("OFFSET")
		w.writeBlank()
		w.writeString(stmt.Offset)
	}
	return nil
}

func (w *Writer) formatSelectWhere(stmt SelectStatement) error {
	w.enter()
	defer w.leave()

	if stmt.Where == nil {
		return nil
	}
	w.writeNL()
	w.writeString("WHERE")
	w.writeBlank()
	return w.formatExpr(stmt.Where, true)
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

func (w *Writer) formatExpr(stmt Statement, nl bool) error {
	var err error
	switch stmt := stmt.(type) {
	case Name:
		w.formatName(stmt)
	case Value:
		w.writeQuoted(stmt.Literal)
	case Call:
		err = w.formatCall(stmt)
	case Binary:
		err = w.formatBinary(stmt, nl)
	case Unary:
		err = w.formatUnary(stmt, nl)
	default:
		err = fmt.Errorf("unexpected expression type (%T)", stmt)
	}
	return err
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
		w.writeString(strings.Repeat(w.indent, w.prefix))
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

func (w *Writer) formatList(list List) error {
	w.writeString("(")

	var err error
	for j, v := range list.Values {
		if j > 0 {
			w.writeString(",")
			w.writeBlank()
		}
		switch v := v.(type) {
		case Name:
			w.formatName(v)
		case Alias:
			err = w.formatAlias(v)
		default:
			err = fmt.Errorf("list: unsupported expression type (%T)", v)
		}
		if err != nil {
			return err
		}
	}
	w.writeString(")")
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
	switch s := alias.Statement.(type) {
	case Name:
		w.formatName(s)
	case Call:
		w.formatCall(s)
	default:
		return fmt.Errorf("alias: unsupported expression type used with alias (%T)", s)
	}
	w.writeBlank()
	w.writeString("AS")
	w.writeBlank()
	w.writeString(alias.Alias)
	return nil
}

func (w *Writer) enter() {
	if w.compact {
		return
	}
	w.prefix++
}

func (w *Writer) leave() {
	if w.compact {
		return
	}
	w.prefix--
}

func (w *Writer) writeString(str string) {
	if w.compact && str == "\n" {
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
	if w.compact {
		return
	}
	w.inner.WriteRune('\n')
}

func (w *Writer) writeBlank() {
	w.inner.WriteRune(' ')
}
