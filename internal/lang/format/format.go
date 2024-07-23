package format

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/midbel/sweet/internal/lang"
	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/lang/parser"
)

type ansiFormatter struct{}

func (_ ansiFormatter) Quote(str string) string {
	return fmt.Sprintf("\"%s\"", str)
}

func GetFormatter() lang.Formatter {
	return ansiFormatter{}
}

type Writer struct {
	inner *bufio.Writer

	Compact       bool
	UseQuote      bool
	UseAs         bool
	UseIndent     int
	UseSpace      bool
	UseColor      bool
	UseCrlf       bool
	UseKeepSpace  bool
	PrependComma  bool
	KeepComment   bool
	ForceOptional bool
	// Compact       CompactMode
	Upperize UpperMode
	Rules    RewriteRule

	noColor   bool
	currDepth int

	lang.Formatter
}

func NewWriter(w io.Writer) *Writer {
	ws := Writer{
		inner:        bufio.NewWriter(w),
		UseIndent:    4,
		UseSpace:     true,
		UseKeepSpace: true,
		Formatter:    ansiFormatter{},
		Upperize:     UpperNone,
		Rules:        0,
	}
	if w != os.Stdout {
		ws.noColor = true
	}
	return &ws
}

func Compact(w io.Writer) *Writer {
	ws := NewWriter(w)
	ws.Compact = true
	return ws
}

func (w *Writer) configure(ps lang.Parser) {
	p, ok := ps.(*parser.Parser)
	if !ok {
		return
	}
	w.Compact = p.GetDefaultBool("compact", w.Compact)
	w.UseIndent = int(p.GetDefaultInt("indent", int64(w.UseIndent)))
	w.UseSpace = p.GetDefaultBool("space", w.UseSpace)
	w.UseKeepSpace = p.GetDefaultBool("keepspace", w.UseKeepSpace)
	w.UseAs = p.GetDefaultBool("as", w.UseAs)
	w.UseQuote = p.GetDefaultBool("quote", w.UseQuote)
	w.UseCrlf = p.GetDefaultBool("crlf", w.UseCrlf)
	w.KeepComment = p.GetDefaultBool("comment", w.KeepComment)
	for _, r := range p.GetStrings("rewrite") {
		switch r {
		case "all":
			w.Rules |= RewriteAll
		case "use-std-op":
			w.Rules |= RewriteStdOp
		case "use-std-expr":
			w.Rules |= RewriteStdExpr
		case "missing-cte-alias":
			w.Rules |= RewriteMissCteAlias
		case "missing-view-alias":
			w.Rules |= RewriteMissViewAlias
		case "subquery-as-cte":
			w.Rules |= RewriteWithCte
		case "cte-as-subquery":
			w.Rules |= RewriteWithSubqueries
		case "join-as-subquery":
			w.Rules |= RewriteJoinSubquery
		case "join-without-literal":
			w.Rules |= RewriteJoinPredicate
		default:
		}
	}
	for _, r := range p.GetStrings("upperize") {
		switch r {
		case "all":
			w.Upperize |= UpperId | UpperKw | UpperFn | UpperType
		case "keyword", "kw":
			w.Upperize |= UpperKw
		case "function", "fn":
			w.Upperize |= UpperFn
		case "identifier", "ident", "id":
			w.Upperize |= UpperId
		case "type":
			w.Upperize |= UpperType
		case "none":
			w.Upperize = UpperNone
		default:
		}
	}
}

func (w *Writer) Format(r io.Reader) error {
	p, err := parser.NewParser(r)
	if err != nil {
		return err
	}
	w.configure(p)
	for {
		stmt, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if stmt, err = w.Rewrite(stmt); err != nil {
			return err
		}
		if err = w.startStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) startStatement(stmt ast.Statement) error {
	defer w.Flush()

	w.Reset()
	w.writeCommentBefore(stmt)
	err := w.FormatStatement(stmt)
	if err == nil {
		w.WriteNL()
		w.WriteEOL()
		w.writeCommentAfter(stmt)
		w.WriteNL()
	}
	return err
}

func (w *Writer) FormatStatement(stmt ast.Statement) error {
	var err error
	switch stmt := stmt.(type) {
	case ast.Node:
		err = w.FormatStatement(stmt.Statement)
	case ast.GrantStatement:
		err = w.FormatGrant(stmt)
	case ast.RevokeStatement:
		err = w.FormatRevoke(stmt)
	case ast.CreateTableStatement:
		err = w.FormatCreateTable(stmt)
	case ast.AlterTableStatement:
		err = w.FormatAlterTable(stmt)
	case ast.CreateViewStatement:
		err = w.FormatCreateView(stmt)
	case ast.DropTableStatement:
		err = w.FormatDropTable(stmt)
	case ast.DropViewStatement:
		err = w.FormatDropView(stmt)
	case ast.CreateProcedureStatement:
		err = w.FormatCreateProcedure(stmt)
	case ast.SelectStatement:
		err = w.FormatSelect(stmt)
	case ast.ValuesStatement:
		err = w.FormatValues(stmt)
	case ast.UnionStatement:
		err = w.FormatUnion(stmt)
	case ast.IntersectStatement:
		err = w.FormatIntersect(stmt)
	case ast.ExceptStatement:
		err = w.FormatExcept(stmt)
	case ast.InsertStatement:
		err = w.FormatInsert(stmt)
	case ast.UpdateStatement:
		err = w.FormatUpdate(stmt)
	case ast.DeleteStatement:
		err = w.FormatDelete(stmt)
	case ast.TruncateStatement:
		err = w.FormatTruncate(stmt)
	case ast.MergeStatement:
		err = w.FormatMerge(stmt)
	case ast.WithStatement:
		err = w.FormatWith(stmt)
	case ast.CteStatement:
		err = w.FormatCte(stmt)
	case ast.CallStatement:
		err = w.FormatCall(stmt)
	case ast.Commit:
		err = w.FormatCommit(stmt)
	case ast.Rollback:
		err = w.FormatRollback(stmt)
	case ast.StartTransaction:
		err = w.FormatStartTransaction(stmt)
	case ast.SetTransaction:
		err = w.FormatSetTransaction(stmt)
	case ast.Savepoint:
		err = w.FormatSavepoint(stmt)
	case ast.ReleaseSavepoint:
		err = w.FormatReleaseSavepoint(stmt)
	case ast.RollbackSavepoint:
		err = w.FormatRollbackSavepoint(stmt)
	case ast.List:
		err = w.FormatBody(stmt)
	case ast.Declare:
		err = w.FormatDeclare(stmt)
	case ast.Return:
		err = w.FormatReturn(stmt)
	case ast.Set:
		err = w.FormatSet(stmt)
	case ast.If:
		err = w.FormatIf(stmt)
	case ast.While:
		err = w.FormatWhile(stmt)
	case ast.Case:
		err = w.FormatCase(stmt)
	case ast.Join:
		err = w.formatJoin(stmt)
	default:
		err = w.FormatExpr(stmt, false)
	}
	return err
}

func (w *Writer) writeCommentAfter(stmt ast.Statement) bool {
	if !w.KeepComment {
		return false
	}
	n, ok := stmt.(ast.Node)
	if !ok {
		return false
	}
	if n.After == "" {
		return false
	}
	w.WriteBlank()
	w.WriteString("--")
	w.WriteBlank()
	w.WriteString(n.After)
	return true
}

func (w *Writer) writeCommentBefore(stmt ast.Statement) {
	if !w.KeepComment {
		return
	}
	n, ok := stmt.(ast.Node)
	if !ok {
		return
	}
	for i := range n.Before {
		w.WritePrefix()
		w.WriteString("--")
		w.WriteBlank()
		w.WriteString(n.Before[i])
		w.WriteNL()
	}
}

func (w *Writer) FormatBody(list ast.List) error {
	doFmt := func(stmt ast.Statement) error {
		return w.FormatStatement(stmt)
	}
	for _, v := range list.Values {
		if err := doFmt(v); err != nil {
			return err
		}
		w.WriteEOL()
	}
	return nil
}

func (w *Writer) FormatExpr(stmt ast.Statement, nl bool) error {
	var err error
	switch stmt := stmt.(type) {
	case ast.Node:
		return w.FormatExpr(stmt.Statement, nl)
	case ast.Placeholder:
		w.FormatPlaceholder(stmt)
	case ast.Name:
		w.FormatName(stmt)
	case ast.Value:
		w.FormatLiteral(stmt.Literal)
	case ast.Group:
		err = w.formatGroup(stmt)
	case ast.Row:
		err = w.FormatRow(stmt, nl)
	case ast.Alias:
		err = w.FormatAlias(stmt)
	case ast.Call:
		err = w.formatCall(stmt)
	case ast.List:
		err = w.formatList(stmt)
	case ast.Binary:
		err = w.formatBinary(stmt, nl)
	case ast.All:
		err = w.formatAll(stmt, nl)
	case ast.Any:
		err = w.formatAny(stmt, nl)
	case ast.Unary:
		err = w.formatUnary(stmt, nl)
	case ast.Between:
		err = w.formatBetween(stmt, false, nl)
	case ast.Is:
		err = w.formatIs(stmt, false, nl)
	case ast.In:
		err = w.formatIn(stmt, false, nl)
	case ast.Collate:
		err = w.formatCollate(stmt, nl)
	case ast.Order:
		err = w.formatOrder(stmt)
	case ast.Cast:
		err = w.FormatCast(stmt, nl)
	case ast.Exists:
		err = w.formatExists(stmt, nl)
	case ast.Not:
		err = w.formatNot(stmt, nl)
	case ast.Case:
		err = w.FormatCase(stmt)
	case ast.When:
		err = w.FormatWhen(stmt)
	default:
		// err = w.FormatStatement(stmt)
		return fmt.Errorf("%T unsupported expression type", stmt)
	}
	return err
}

func (w *Writer) formatNot(stmt ast.Not, _ bool) error {
	switch stmt := stmt.Statement.(type) {
	case ast.Between:
		return w.formatBetween(stmt, true, false)
	case ast.Is:
		return w.formatIs(stmt, true, false)
	case ast.In:
		return w.formatIn(stmt, true, false)
	default:
		w.WriteKeyword("NOT")
		w.WriteBlank()
		return w.FormatExpr(stmt, false)
	}
}

func (w *Writer) formatExists(stmt ast.Exists, _ bool) error {
	w.WriteKeyword("EXISTS")
	w.WriteString("(")
	w.WriteNL()
	if err := w.FormatStatement(stmt.Statement); err != nil {
		return err
	}
	w.WriteNL()
	w.WriteString(")")
	return nil
}

func (w *Writer) formatCollate(stmt ast.Collate, _ bool) error {
	if err := w.FormatExpr(stmt.Statement, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("COLLATE")
	w.WriteBlank()
	w.WriteString("\"")
	w.WriteString(stmt.Collation)
	w.WriteString("\"")
	return nil
}

func (w *Writer) formatStmtSlice(values []ast.Statement) error {
	for i, v := range values {
		w.WriteComma(i)
		if err := w.FormatExpr(v, false); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) formatList(stmt ast.List) error {
	w.WriteString("(")
	defer w.WriteString(")")
	for i, v := range stmt.Values {
		if i > 0 {
			w.WriteString(",")
			if w.UseKeepSpace {
				w.WriteBlank()
			}
		}
		if err := w.FormatExpr(v, false); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) formatCall(call ast.Call) error {
	n, ok := call.Ident.(ast.Name)
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
		case ast.Name:
			w.WriteBlank()
			return w.FormatExpr(over, false)
		case ast.Window:
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
					o, ok := s.(ast.Order)
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

func (w *Writer) formatIs(stmt ast.Is, not, nl bool) error {
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

func (w *Writer) formatIn(stmt ast.In, not, nl bool) error {
	if err := w.FormatExpr(stmt.Ident, nl); err != nil {
		return err
	}
	w.WriteBlank()
	if not {
		w.WriteKeyword("NOT")
		w.WriteBlank()
	}
	w.WriteKeyword("IN")
	if !w.Compact {
		w.WriteBlank()
	}
	if stmt, ok := stmt.Value.(ast.Group); ok {
		return w.compact(func() error {
			return w.formatGroup(stmt)
		})
	}
	return w.FormatExpr(stmt.Value, false)
}

func (w *Writer) formatBetween(stmt ast.Between, not, nl bool) error {
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

func (w *Writer) formatUnary(stmt ast.Unary, nl bool) error {
	w.WriteString(stmt.Op)
	w.WriteBlank()
	return w.FormatExpr(stmt.Right, nl)
}

func (w *Writer) formatGroup(stmt ast.Group) error {
	if _, ok := stmt.Statement.(ast.SelectStatement); ok {
		w.WriteString("(")
		if !w.Compact {
			w.WriteNL()
		}
		if err := w.FormatStatement(stmt.Statement); err != nil {
			return err
		}
		if !w.Compact {
			w.WriteNL()
			w.WritePrefix()
		}
		w.WriteString(")")
		return nil
	}
	w.WriteString("(")
	defer w.WriteString(")")
	return w.FormatExpr(stmt.Statement, false)
}

func (w *Writer) formatRelation(stmt ast.Binary, nl bool) error {
	if err := w.FormatExpr(stmt.Left, false); err != nil {
		return err
	}
	w.WriteNL()
	w.Enter()
	w.WritePrefix()
	w.WriteKeyword(stmt.Op)
	w.WriteBlank()
	w.Leave()
	return w.FormatExpr(stmt.Right, false)
}

func (w *Writer) formatAll(stmt ast.All, _ bool) error {
	w.WriteKeyword("ALL")
	w.WriteString("(")
	defer w.WriteString(")")
	return w.compact(func() error {
		return w.FormatExpr(stmt.Statement, false)
	})
}

func (w *Writer) formatAny(stmt ast.Any, _ bool) error {
	w.WriteKeyword("ANY")
	w.WriteString("(")
	defer w.WriteString(")")
	return w.compact(func() error {
		return w.FormatExpr(stmt.Statement, false)
	})
}

func (w *Writer) formatBinary(stmt ast.Binary, nl bool) error {
	if stmt.IsRelation() {
		return w.formatRelation(stmt, nl)
	}
	if err := w.FormatExpr(stmt.Left, nl); err != nil {
		return err
	}
	if w.UseKeepSpace {
		w.WriteBlank()
	}
	w.WriteKeyword(stmt.Op)
	if w.UseKeepSpace {
		w.WriteBlank()
	}
	if err := w.FormatExpr(stmt.Right, nl); err != nil {
		return err
	}
	return nil
}

func (w *Writer) WriteCall(call string) {
	if w.withColor() {
		w.WriteString(callColor)
	}
	if w.Upperize.Function() || w.Upperize.All() {
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
	w.WriteString("--")
	w.WriteBlank()
	w.WriteString(str)
	w.WriteNL()
}

func (w *Writer) WriteEOL() {
	w.WriteString(";")
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

func (w *Writer) WriteKeyword(kw string) {
	if w.Upperize.Keyword() || w.Upperize.All() {
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

func (w *Writer) compact(fn func() error) error {
	c := w.Compact
	defer func() {
		w.Compact = c
	}()
	w.Compact = true
	return fn()
}

func (w *Writer) CanNotUse(ctx string, stmt ast.Statement) error {
	return fmt.Errorf("%T can not be used as statement in %s", stmt, ctx)
}

func (w *Writer) withColor() bool {
	if w.noColor {
		return false
	}
	return w.UseColor
}

const (
	keywordColor = "\033[38;2;173;216;230m"
	numberColor  = "\033[38;2;234;72;72m"
	stringColor  = "\033[38;2;252;245;95m"
	callColor    = "\033[38;2;80;200;120m"
	resetCode    = "\033[0m"
)
