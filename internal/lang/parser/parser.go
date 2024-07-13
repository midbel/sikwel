package parser

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/midbel/sweet/internal/config"
	"github.com/midbel/sweet/internal/lang"
	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/scanner"
	"github.com/midbel/sweet/internal/token"
)

var errDone = errors.New("done")

type ItemFunc func() (ast.Statement, error)

type Parser struct {
	*frame
	*config.Config
	stack []*frame

	level int

	keywords map[string]func() (ast.Statement, error)
	infix    *stack[infixFunc]
	prefix   *stack[prefixFunc]

	withAlias bool

	queries map[string]ast.Statement
	values  map[string]ast.Statement
}

func NewParser(r io.Reader) (lang.Parser, error) {
	scan, err := scanner.Scan(r, lang.GetKeywords())
	if err != nil {
		return nil, err
	}
	return ParseWithScanner(scan)

}

func ParseWithScanner(scan *scanner.Scanner) (*Parser, error) {
	f, err := createFrameFromScanner(scan)
	if err != nil {
		return nil, err
	}
	var p Parser
	p.Config = config.Make()
	p.frame = f
	p.queries = make(map[string]ast.Statement)
	p.values = make(map[string]ast.Statement)
	p.infix = emptyStack[infixFunc]()
	p.prefix = emptyStack[prefixFunc]()

	p.setParseFunc()
	p.setDefaultFuncSet()
	p.toggleAlias()

	return &p, p.start()
}

func (p *Parser) DefineVars(file string) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	return nil
}

func (p *Parser) start() error {
	for p.Is(token.Macro) {
		var err error
		switch p.GetCurrLiteral() {
		case "FORMAT":
			err = p.ParseFormatMacro()
		case "LINT":
			err = p.ParseLintMacro()
		default:
			err = p.ParseMacro()
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) Parse() (ast.Statement, error) {
	if p.Done() {
		return nil, io.EOF
	}
	p.reset()
	stmt, err := p.parse()
	if err != nil {
		p.restore()
	}
	return stmt, err
}

func (p *Parser) ParseStatement() (ast.Statement, error) {
	p.Enter()
	defer p.Leave()

	p.setDefaultFuncSet()
	defer func() {
		p.prefix.Pop()
		p.infix.Pop()
	}()

	if p.Done() {
		return nil, io.EOF
	}
	if !p.Is(token.Keyword) {
		return nil, p.wantError("statement", "keyword")
	}
	fn, ok := p.keywords[p.GetCurrLiteral()]
	if !ok {
		return nil, p.Unexpected("statement")
	}
	return fn()
}

func (p *Parser) Level() int {
	return p.level
}

func (p *Parser) Enter() {
	p.level++
}

func (p *Parser) Leave() {
	p.level--
}

func (p *Parser) Nested() bool {
	return p.level > 1
}

func (p *Parser) reset() {
	p.level = 0
}

func (p *Parser) QueryEnds() bool {
	if p.Nested() {
		return p.Is(token.Rparen)
	}
	return p.Is(token.EOL) || p.Done()
}

func (p *Parser) Done() bool {
	if p.frame.Done() {
		if n := len(p.stack); n > 0 {
			p.frame = p.stack[n-1]
			p.stack = p.stack[:n-1]
		}
	}
	return p.frame.Done()
}

func (p *Parser) Expect(ctx string, r rune) error {
	if !p.Is(r) {
		return p.Unexpected(ctx)
	}
	p.Next()
	return nil
}

func (p *Parser) restore() {
	defer p.Next()
	for !p.Done() && !p.Is(token.EOL) {
		p.Next()
	}
}

func (p *Parser) parse() (ast.Statement, error) {
	if p.Is(token.Macro) {
		if err := p.ParseMacro(); err != nil {
			return nil, err
		}
		return p.Parse()
	}
	return p.parseItem(func() (ast.Statement, error) {
		stmt, err := p.ParseStatement()
		if err != nil {
			return nil, err
		}
		if !p.Is(token.EOL) {
			return nil, p.Unexpected("statement")
		}
		p.Next()
		return stmt, nil
	})
}

func (p *Parser) parseItem(parse ItemFunc) (ast.Statement, error) {
	var node ast.Node
	for p.Is(token.Comment) {
		node.Before = append(node.Before, p.GetCurrLiteral())
		p.Next()
	}
	var (
		pos = p.curr.Position
		err error
	)
	if node.Statement, err = parse(); err != nil && !errors.Is(err, errDone) {
		return nil, err
	}
	if p.Is(token.Comment) && pos.Column < p.curr.Column {
		node.After = p.GetCurrLiteral()
		p.Next()
	}
	return node.Get(), err
}

func (p *Parser) RegisterParseFunc(kw string, fn func() (ast.Statement, error)) {
	kw = strings.ToUpper(kw)
	p.keywords[kw] = fn
}

func (p *Parser) UnregisterParseFunc(kw string) {
	kw = strings.ToUpper(kw)
	delete(p.keywords, kw)
}

func (p *Parser) UnregisterAllParseFunc() {
	p.keywords = make(map[string]func() (ast.Statement, error))
}

func (p *Parser) RegisterPrefix(literal string, kind rune, fn prefixFunc) {
	p.prefix.Register(literal, kind, fn)
}

func (p *Parser) UnregisterPrefix(literal string, kind rune) {
	p.prefix.Unregister(literal, kind)
}

func (p *Parser) RegisterInfix(literal string, kind rune, fn infixFunc) {
	p.infix.Register(literal, kind, fn)
}

func (p *Parser) UnregisterInfix(literal string, kind rune) {
	p.infix.Unregister(literal, kind)
}

func (p *Parser) parseColumnsList() ([]string, error) {
	if !p.Is(token.Lparen) {
		return nil, nil
	}
	p.Next()

	var (
		list []string
		err  error
	)

	for !p.Done() && !p.Is(token.Rparen) {
		if !p.Curr().IsValue() {
			return nil, p.Unexpected("columns")
		}
		list = append(list, p.GetCurrLiteral())
		p.Next()
		if err := p.EnsureEnd("columns", token.Comma, token.Rparen); err != nil {
			return nil, err
		}
	}
	if !p.Is(token.Rparen) {
		return nil, p.Unexpected("columns")
	}
	p.Next()

	return list, err
}

func (p *Parser) IsKeyword(kw string) bool {
	return p.Curr().Type == token.Keyword && p.GetCurrLiteral() == kw
}

func (p *Parser) wantError(ctx, str string) error {
	curr := p.Curr()
	return fmt.Errorf("%s: expected %q at %d:%d! got %s", ctx, str, curr.Line, curr.Column, curr.Literal)
}

func (p *Parser) Unexpected(ctx string) error {
	return p.UnexpectedDialect(ctx, "lang")
}

func (p *Parser) UnexpectedDialect(ctx, dialect string) error {
	return wrapErrorWithDialect(dialect, ctx, unexpected(p.Curr()))
}

func (p *Parser) EnsureEnd(ctx string, sep, end rune) error {
	switch {
	case p.Is(sep):
		p.Next()
		if p.Is(end) {
			return p.Unexpected(ctx)
		}
	case p.Is(end):
	default:
		return p.Unexpected(ctx)
	}
	return nil
}

func (p *Parser) tokCheck(kind ...rune) func() bool {
	sort.Slice(kind, func(i, j int) bool {
		return kind[i] < kind[j]
	})
	return func() bool {
		i := sort.Search(len(kind), func(i int) bool {
			return p.Is(kind[i])
		})
		return i < len(kind) && kind[i] == p.Curr().Type
	}
}

func (p *Parser) KwCheck(str ...string) func() bool {
	sort.Strings(str)
	return func() bool {
		if !p.Is(token.Keyword) {
			return false
		}
		if len(str) == 1 {
			return str[0] == p.GetCurrLiteral()
		}
		i := sort.SearchStrings(str, p.GetCurrLiteral())
		return i < len(str) && str[i] == p.GetCurrLiteral()
	}
}

func (p *Parser) setParseFunc() {
	p.keywords = make(map[string]func() (ast.Statement, error))
	p.RegisterParseFunc("SELECT", p.ParseSelect)
	p.RegisterParseFunc("VALUES", p.ParseValues)
	p.RegisterParseFunc("DELETE FROM", p.ParseDelete)
	p.RegisterParseFunc("TRUNCATE", p.ParseTruncate)
	p.RegisterParseFunc("TRUNCATE TABLE", p.ParseTruncate)
	p.RegisterParseFunc("UPDATE", p.ParseUpdate)
	p.RegisterParseFunc("MERGE", p.ParseMerge)
	p.RegisterParseFunc("MERGE INTO", p.ParseMerge)
	p.RegisterParseFunc("INSERT INTO", p.ParseInsert)
	p.RegisterParseFunc("WITH", p.parseWith)
	p.RegisterParseFunc("CASE", p.ParseCase)
	p.RegisterParseFunc("CALL", p.ParseCall)
	p.RegisterParseFunc("IF", p.parseIf)
	p.RegisterParseFunc("WHILE", p.parseWhile)
	p.RegisterParseFunc("DECLARE", p.ParseDeclare)
	p.RegisterParseFunc("SET", p.parseSet)
	p.RegisterParseFunc("RETURN", p.parseReturn)
	p.RegisterParseFunc("BEGIN", p.ParseBegin)
	p.RegisterParseFunc("START TRANSACTION", p.parseStartTransaction)
	p.RegisterParseFunc("CREATE VIEW", p.ParseCreateView)
	p.RegisterParseFunc("CREATE TEMP VIEW", p.ParseCreateView)
	p.RegisterParseFunc("CREATE TEMPORARY VIEW", p.ParseCreateView)
	p.RegisterParseFunc("CREATE TABLE", p.ParseCreateTable)
	p.RegisterParseFunc("CREATE TEMP TABLE", p.ParseCreateTable)
	p.RegisterParseFunc("CREATE TEMPORARY TABLE", p.ParseCreateTable)
	p.RegisterParseFunc("CREATE PROCEDURE", p.ParseCreateProcedure)
	p.RegisterParseFunc("CREATE OR REPLACE PROCEDURE", p.ParseCreateProcedure)
	p.RegisterParseFunc("ALTER TABLE", p.ParseAlterTable)
	p.RegisterParseFunc("DROP TABLE", p.ParseDropTable)
	p.RegisterParseFunc("DROP VIEW", p.ParseDropView)
	p.RegisterParseFunc("GRANT", p.ParseGrant)
	p.RegisterParseFunc("REVOKE", p.ParseRevoke)
}

func (p *Parser) setFuncSetForTable() {
	prefix := newFuncSet[prefixFunc]()
	prefix.Register("", token.Ident, p.ParseIdent)
	prefix.Register("", token.Lparen, p.parseGroupExpr)
	prefix.Register("ROW", token.Keyword, p.ParseRow)

	p.prefix.Push(prefix)

	infix := newFuncSet[infixFunc]()
	p.infix.Push(infix)
}

func (p *Parser) setDefaultFuncSet() {
	infix := newFuncSet[infixFunc]()
	infix.Register("", token.Plus, p.parseInfixExpr)
	infix.Register("", token.Minus, p.parseInfixExpr)
	infix.Register("", token.Slash, p.parseInfixExpr)
	infix.Register("", token.Star, p.parseInfixExpr)
	infix.Register("", token.Concat, p.parseInfixExpr)
	infix.Register("", token.Eq, p.parseInfixExpr)
	infix.Register("", token.Ne, p.parseInfixExpr)
	infix.Register("", token.Lt, p.parseInfixExpr)
	infix.Register("", token.Le, p.parseInfixExpr)
	infix.Register("", token.Gt, p.parseInfixExpr)
	infix.Register("", token.Ge, p.parseInfixExpr)
	infix.Register("", token.Lparen, p.parseCallExpr)
	infix.Register("AND", token.Keyword, p.parseKeywordExpr)
	infix.Register("OR", token.Keyword, p.parseKeywordExpr)
	infix.Register("NOT", token.Keyword, p.parseKeywordExpr)
	infix.Register("LIKE", token.Keyword, p.parseKeywordExpr)
	infix.Register("SIMILAR", token.Keyword, p.parseKeywordExpr)
	infix.Register("ILIKE", token.Keyword, p.parseKeywordExpr)
	infix.Register("BETWEEN", token.Keyword, p.parseKeywordExpr)
	infix.Register("COLLATE", token.Keyword, p.parseCollateExpr)
	infix.Register("IN", token.Keyword, p.parseKeywordExpr)
	infix.Register("IS", token.Keyword, p.parseKeywordExpr)
	infix.Register("ISNULL", token.Keyword, p.parseKeywordExpr)
	infix.Register("NOTNULL", token.Keyword, p.parseKeywordExpr)
	infix.Register("ALL", token.Keyword, p.parseKeywordExpr)

	p.infix.Push(infix)

	prefix := newFuncSet[prefixFunc]()
	prefix.Register("", token.Ident, p.ParseIdentifier)
	prefix.Register("", token.Star, p.ParseIdentifier)
	prefix.Register("", token.Literal, p.ParseLiteral)
	prefix.Register("", token.Number, p.ParseLiteral)
	prefix.Register("", token.Lparen, p.parseGroupExpr)
	prefix.Register("", token.Minus, p.parseUnary)
	prefix.Register("", token.Keyword, p.parseUnary)
	prefix.Register("", token.Placeholder, p.ParsePlaceholder)
	prefix.Register("", token.NamedHolder, p.ParsePlaceholder)
	prefix.Register("", token.PositionHolder, p.ParsePlaceholder)
	prefix.Register("NOT", token.Keyword, p.parseUnary)
	prefix.Register("NULL", token.Keyword, p.ParseConstant)
	prefix.Register("DEFAULT", token.Keyword, p.ParseConstant)
	prefix.Register("TRUE", token.Keyword, p.ParseConstant)
	prefix.Register("FALSE", token.Keyword, p.ParseConstant)
	prefix.Register("CASE", token.Keyword, p.ParseCase)
	prefix.Register("SELECT", token.Keyword, p.ParseStatement)
	prefix.Register("CAST", token.Keyword, p.ParseCast)
	prefix.Register("ROW", token.Keyword, p.ParseRow)
	prefix.Register("EXISTS", token.Keyword, p.parseExists)

	p.prefix.Push(prefix)
}

func (p *Parser) toggleAlias() {
	p.withAlias = !p.withAlias
}

func (p *Parser) unsetFuncSet() {
	p.infix.Pop()
	p.prefix.Pop()
}

type frame struct {
	scan *scanner.Scanner

	file string
	curr token.Token
	peek token.Token
}

func createFrameFromScanner(scan *scanner.Scanner) (*frame, error) {
	f := &frame{
		scan: scan,
	}
	f.Next()
	f.Next()
	return f, nil
}

func createFrame(r io.Reader) (*frame, error) {
	scan, err := scanner.Scan(r, lang.GetKeywords())
	if err != nil {
		return nil, err
	}
	return createFrameFromScanner(scan)
}

func (f *frame) Sub(r io.Reader) (*frame, error) {
	scan, err := f.scan.Clone(r)
	if err != nil {
		return nil, err
	}
	return createFrameFromScanner(scan)
}

func (f *frame) Base() string {
	return filepath.Dir(f.file)
}

func (f *frame) Curr() token.Token {
	return f.curr
}

func (f *frame) Peek() token.Token {
	return f.peek
}

func (f *frame) GetCurrLiteral() string {
	return f.curr.Literal
}

func (f *frame) GetPeekLiteral() string {
	return f.peek.Literal
}

func (f *frame) GetCurrType() rune {
	return f.curr.Type
}

func (f *frame) GetPeekType() rune {
	return f.peek.Type
}

func (f *frame) Next() {
	f.curr = f.peek
	f.peek = f.scan.Scan()
}

func (f *frame) Done() bool {
	return f.Is(token.EOF)
}

func (f *frame) Is(kind rune) bool {
	return f.curr.Type == kind
}

func (f *frame) PeekIs(kind rune) bool {
	return f.peek.Type == kind
}
