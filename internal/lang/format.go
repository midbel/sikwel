package lang

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Writer struct {
	inner       *bufio.Writer
	Compact     bool
	KwUpper     bool
	FnUpper     bool
	KeepComment bool
	Colorize    bool
	Indent      string

	noColor bool
	prefix  int
}

func NewWriter(w io.Writer) *Writer {
	ws := Writer{
		inner:  bufio.NewWriter(w),
		Indent: "  ",
	}
	if w != os.Stdout {
		ws.noColor = true
	}
	return &ws
}

func (w *Writer) SetIndent(indent string) {
	w.Indent = indent
}

func (w *Writer) SetCompact(compact bool) {
	w.Compact = compact
}

func (w *Writer) SetKeepComments(keep bool) {
	w.KeepComment = keep
}

func (w *Writer) SetKeywordUppercase(upper bool) {
	w.KwUpper = upper
}

func (w *Writer) SetFunctionUppercase(upper bool) {
	w.FnUpper = upper
}

func (w *Writer) ColorizeOutput(colorize bool) {
	w.Colorize = colorize
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

func (w *Writer) startStatement(stmt Statement) error {
	defer w.Flush()

	w.Reset()
	com, ok := stmt.(Commented)
	if ok {
		if w.KeepComment {
			for _, s := range com.Before {
				w.WriteComment(s)
			}
		}
		stmt = com.Statement
	}
	err := w.FormatStatement(stmt)
	if err == nil {
		w.WriteEOL()
	}
	return err
}

func (w *Writer) FormatStatement(stmt Statement) error {
	var err error
	switch stmt := stmt.(type) {
	case CreateTableStatement:
		err = w.FormatCreateTable(stmt)
	case CreateProcedureStatement:
		err = w.FormatCreateProcedure(stmt)
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
	case CallStatement:
		err = w.FormatCall(stmt)
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

func (w *Writer) FormatCall(stmt CallStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteString("(")
	for i, a := range stmt.Args {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		if err := w.FormatExpr(a, false); err != nil {
			return err
		}
	}
	w.WriteString(")")
	return nil
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
	w.WriteCall(n.Ident)
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
	w.WriteKeyword("AS")
	w.WriteBlank()
	w.WriteString(alias.Alias)
	return nil
}

func (w *Writer) formatValue(literal string) {
	if literal == "NULL" || literal == "DEFAULT" || literal == "*" {
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

func (w *Writer) WriteComment(str string) {
	if w.Compact {
		return
	}
	w.WritePrefix()
	w.WriteString("--")
	w.WriteBlank()
	w.WriteString(str)
	w.WriteNL()
}

func (w *Writer) WriteEOL() {
	w.WriteString(";")
	w.WriteNL()
}

func (w *Writer) WriteQuoted(str string) {
	if w.withColor() {
		w.WriteString(stringColor)
	}
	w.inner.WriteRune('\'')
	w.WriteString(str)
	w.inner.WriteRune('\'')
	if w.withColor() {
		w.WriteString(resetCode)
	}
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

func (w *Writer) WriteCall(call string) {
	if w.withColor() {
		w.WriteString(callColor)
	}
	w.WriteString(call)
	if w.withColor() {
		w.WriteString(resetCode)
	}
}

func (w *Writer) WriteKeyword(kw string) {
	if !isAlpha(kw) {
		w.WriteString(kw)
		return
	}
	if !w.KwUpper {
		kw = strings.ToLower(kw)
	} else {
		kw = strings.ToUpper(kw)
	}
	if w.withColor() {
		w.WriteString(keywordColor)
	}
	w.WriteString(kw)
	if w.withColor() {
		w.WriteString(resetCode)
	}
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

func (w *Writer) withColor() bool {
	if w.noColor {
		return false
	}
	return w.Colorize
}

func isAlpha(str string) bool {
	other := strings.Map(func(r rune) rune {
		if isLetter(r) || isBlank(r) {
			return r
		}
		return -1
	}, str)
	return other == str
}

const (
	keywordColor = "\033[38;2;173;216;230m"
	numberColor  = "\033[38;2;234;72;72m"
	stringColor  = "\033[38;2;252;245;95m"
	callColor    = "\033[38;2;80;200;120m"
	resetCode    = "\033[0m"
)
