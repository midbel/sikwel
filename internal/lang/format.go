package lang

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type SqlFormatter interface {
	Quote(string) string
}

func GetDialectFormatter(name string) (SqlFormatter, error) {
	switch name {
	case "my", "mysql":
		return mysqlFormatter{}, nil
	case "mssql":
		return tsqlFormatter{}, nil
	case "ansi", "pg", "postgres", "sqlite", "lite":
		return ansiFormatter{}, nil
	default:
		return nil, fmt.Errorf("%s unsupported dialect", name)
	}
}

type mysqlFormatter struct{}

func (_ mysqlFormatter) Quote(str string) string {
	return fmt.Sprintf("`%s`", str)
}

type ansiFormatter struct{}

func (_ ansiFormatter) Quote(str string) string {
	return fmt.Sprintf("\"%s\"", str)
}

type tsqlFormatter struct{}

func (_ tsqlFormatter) Quote(str string) string {
	return fmt.Sprintf("[%s]", str)
}

type Writer struct {
	inner *bufio.Writer

	Compact      bool
	UseQuote     bool
	UseAs        bool
	UseIndent    int
	UseSpace     bool
	UseColor     bool
	UseSubQuery  bool
	UseCte       bool
	UseCrlf      bool
	PrependComma bool
	KeepComment  bool
	Upperize     bool
	UpperizeK    bool
	UpperizeF    bool

	UseNames bool

	noColor       bool
	currExprDepth int
	currDepth     int

	SqlFormatter
}

func NewWriter(w io.Writer) *Writer {
	ws := Writer{
		inner:        bufio.NewWriter(w),
		UseIndent:    4,
		UseSpace:     true,
		SqlFormatter: ansiFormatter{},
	}
	if w != os.Stdout {
		ws.noColor = true
	}
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
	if w.UseSubQuery {
		with, ok := stmt.(WithStatement)
		if ok {
			stmt := substituteQueries(with.Queries, with.Statement)
			return w.FormatStatement(stmt)
		}
	}
	var err error
	switch stmt := stmt.(type) {
	case GrantStatement:
		err = w.FormatGrant(stmt)
	case RevokeStatement:
		err = w.FormatRevoke(stmt)
	case CreateTableStatement:
		err = w.FormatCreateTable(stmt)
	case AlterTableStatement:
		err = w.FormatAlterTable(stmt)
	case CreateViewStatement:
		err = w.FormatCreateView(stmt)
	case DropTableStatement:
		err = w.FormatDropTable(stmt)
	case DropViewStatement:
		err = w.FormatDropView(stmt)
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
	case TruncateStatement:
		err = w.FormatTruncate(stmt)
	case MergeStatement:
		err = w.FormatMerge(stmt)
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
	case Declare:
		err = w.FormatDeclare(stmt)
	case Return:
		err = w.FormatReturn(stmt)
	case SetStatement:
		err = w.FormatSet(stmt)
	case IfStatement:
		err = w.FormatIf(stmt)
	case WhileStatement:
		err = w.FormatWhile(stmt)
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

func (w *Writer) FormatExpr(stmt Statement, nl bool) error {
	w.enterExpr()
	defer w.leaveExpr()
	var err error
	switch stmt := stmt.(type) {
	case Name:
		w.FormatName(stmt)
	case Value:
		w.FormatLiteral(stmt.Literal)
	case Group:
		err = w.formatGroup(stmt)
	case Row:
		err = w.FormatRow(stmt, nl)
	case Alias:
		err = w.FormatAlias(stmt)
	case Call:
		err = w.formatCall(stmt)
	case List:
		err = w.formatList(stmt)
	case Binary:
		err = w.formatBinary(stmt, nl)
	case All:
		err = w.formatAll(stmt, nl)
	case Any:
		err = w.formatAny(stmt, nl)
	case Unary:
		err = w.formatUnary(stmt, nl)
	case Between:
		err = w.formatBetween(stmt, false, nl)
	case Is:
		err = w.formatIs(stmt, false, nl)
	case In:
		err = w.formatIn(stmt, false, nl)
	case Collate:
		err = w.formatCollate(stmt, nl)
	case Cast:
		err = w.FormatCast(stmt, nl)
	case Exists:
		err = w.formatExists(stmt, nl)
	case Not:
		err = w.formatNot(stmt, nl)
	case CaseStatement:
		err = w.FormatCase(stmt)
	case WhenStatement:
		err = w.FormatWhen(stmt)
	default:
		err = w.FormatStatement(stmt)
	}
	return err
}

func (w *Writer) formatNot(stmt Not, _ bool) error {
	switch stmt := stmt.Statement.(type) {
	case Between:
		return w.formatBetween(stmt, true, false)
	case Is:
		return w.formatIs(stmt, true, false)
	case In:
		return w.formatIn(stmt, true, false)
	default:
		w.WriteKeyword("NOT")
		w.WriteBlank()
		return w.FormatStatement(stmt)
	}
}

func (w *Writer) formatExists(stmt Exists, _ bool) error {
	compact := w.Compact
	defer func() {
		w.Compact = compact
	}()
	w.Compact = true
	w.WriteKeyword("EXISTS")
	w.WriteString("(")
	if err := w.FormatExpr(stmt.Statement, false); err != nil {
		return err
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
		w.WriteComma(i)
		if err := w.FormatExpr(v, false); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) formatList(stmt List) error {
	w.WriteString("(")
	defer w.WriteString(")")
	for i, v := range stmt.Values {
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

func (w *Writer) formatCall(call Call) error {
	n, ok := call.Ident.(Name)
	if !ok {
		return w.CanNotUse("call", call.Ident)
	}
	w.WriteCall(n.Ident())
	w.WriteString("(")
	if call.Distinct {
		w.WriteKeyword("DISTINCT")
		w.WriteBlank()
	}
	for i := range call.Args {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		if err := w.FormatExpr(call.Args[i], false); err != nil {
			return err
		}
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

func (w *Writer) formatIs(stmt Is, not, nl bool) error {
	if err := w.FormatExpr(stmt.Ident, nl); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("IS")
	w.WriteBlank()
	if not {
		w.WriteKeyword("NOT")
		w.WriteBlank()
	}
	return w.FormatExpr(stmt.Value, false)
}

func (w *Writer) formatIn(stmt In, not, nl bool) error {
	if err := w.FormatExpr(stmt.Ident, nl); err != nil {
		return err
	}
	w.WriteBlank()
	if not {
		w.WriteKeyword("NOT")
		w.WriteBlank()
	}
	w.WriteKeyword("IN")
	w.WriteBlank()

	if stmt, ok := stmt.Value.(SelectStatement); ok {
		w.WriteString("(")
		err := w.FormatSelect(stmt)
		w.WriteString(")")
		return err
	}
	return w.FormatExpr(stmt.Value, false)
}

func (w *Writer) formatBetween(stmt Between, not, nl bool) error {
	if err := w.FormatExpr(stmt.Ident, nl); err != nil {
		return err
	}
	w.WriteBlank()
	if not {
		w.WriteKeyword("NOT")
		w.WriteBlank()
	}
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

func (w *Writer) formatGroup(stmt Group) error {
	w.WriteString("(")
	defer w.WriteString(")")

	w.Enter()
	defer w.Leave()
	return w.FormatExpr(stmt.Statement, false)
}

func (w *Writer) formatRelation(stmt Binary, nl bool) error {
	if err := w.FormatExpr(stmt.Left, false); err != nil {
		return err
	}
	w.WriteNL()
	w.WritePrefix()
	w.WriteKeyword(stmt.Op)
	w.WriteBlank()
	return w.FormatExpr(stmt.Right, false)
}

func (w *Writer) formatAll(stmt All, _ bool) error {
	w.WriteKeyword("ALL")
	w.WriteBlank()
	return w.FormatExpr(stmt.Statement, false)
}

func (w *Writer) formatAny(stmt Any, _ bool) error {
	w.WriteKeyword("ANY")
	w.WriteBlank()
	return w.FormatExpr(stmt.Statement, false)
}

func (w *Writer) formatBinary(stmt Binary, nl bool) error {
	if stmt.IsRelation() {
		return w.formatRelation(stmt, nl)
	}
	if err := w.FormatExpr(stmt.Left, nl); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword(stmt.Op)
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Right, nl); err != nil {
		return err
	}
	return nil
}

func (w *Writer) WriteCall(call string) {
	if w.withColor() {
		w.WriteString(callColor)
	}
	if w.UpperizeF || w.Upperize {
		call = strings.ToUpper(call)
	}
	w.WriteString(call)
	if w.withColor() {
		w.WriteString(resetCode)
	}
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

func (w *Writer) WriteComma(i int) {
	if (!w.PrependComma || w.Compact) && i > 0 {
		w.WriteString(",")
	}
	if i > 0 {
		w.WriteNL()
	}
	w.WritePrefix()
	if w.PrependComma && !w.Compact {
		if i == 0 {
			w.WriteBlank()
		} else {
			w.WriteString(",")
		}
	}
}

func (w *Writer) WriteNL() {
	if w.Compact {
		w.WriteBlank()
		return
	}
	if w.UseCrlf {
		w.inner.WriteRune('\r')
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
	if !isAlpha(kw) {
		w.WriteString(kw)
		return
	}
	if w.UpperizeK || w.Upperize {
		kw = strings.ToUpper(kw)
	} else {
		kw = strings.ToLower(kw)
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
	if w.Compact {
		return
	}
	if !w.UseSpace {
		w.inner.WriteRune('\t')
	}
	if w.UseIndent <= 0 {
		return
	}

	w.WriteString(strings.Repeat(" ", w.UseIndent*w.getCurrDepth()))
}

func (w *Writer) Flush() {
	w.inner.Flush()
}

func (w *Writer) Reset() {
	w.currDepth = -1
}

func (w *Writer) Enter() {
	if w.Compact {
		return
	}
	w.currDepth++
}

func (w *Writer) Leave() {
	if w.Compact || w.currDepth < 0 {
		return
	}
	w.currDepth--
}

func (w *Writer) getCurrDepth() int {
	if w.currDepth < 0 {
		return 0
	}
	return w.currDepth
}

func (w *Writer) enterExpr() {
	w.currExprDepth++
}

func (w *Writer) leaveExpr() {
	w.currExprDepth--
}

func (w *Writer) CanNotUse(ctx string, stmt Statement) error {
	return fmt.Errorf("%T can not be used as statement in %s", stmt, ctx)
}

func (w *Writer) withColor() bool {
	if w.noColor {
		return false
	}
	return w.UseColor
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
