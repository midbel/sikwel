package lang

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Parser struct {
	*frame
	stack []*frame

	level int

	keywords map[string]func() (Statement, error)
	infix    *stack[infixFunc]
	prefix   *stack[prefixFunc]

	inlineCte bool
	withAlias bool

	queries map[string]Statement
	values  map[string]Statement
}

func NewParser(r io.Reader) (*Parser, error) {
	return NewParserWithKeywords(r, keywords)
}

func NewParserWithKeywords(r io.Reader, set KeywordSet) (*Parser, error) {
	var p Parser

	frame, err := createFrame(r, set)
	if err != nil {
		return nil, err
	}
	p.frame = frame
	p.queries = make(map[string]Statement)
	p.values = make(map[string]Statement)
	p.infix = emptyStack[infixFunc]()
	p.prefix = emptyStack[prefixFunc]()

	p.setParseFunc()
	p.setDefaultFuncSet()
	p.toggleAlias()

	return &p, nil
}

func (p *Parser) DefineVars(file string) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	return nil
}

func (p *Parser) SetInline(inline bool) {
	if p.level != 0 {
		return
	}
	p.inlineCte = inline
}

func (p *Parser) Parse() (Statement, error) {
	p.reset()
	stmt, err := p.parse()
	if err != nil {
		p.restore()
	}
	return stmt, err
}

func (p *Parser) ParseStatement() (Statement, error) {
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
	if !p.Is(Keyword) {
		return nil, p.wantError("statement", "keyword")
	}
	fn, ok := p.keywords[p.curr.Literal]
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
		return p.Is(Rparen)
	}
	return p.Is(EOL) || p.Done()
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
	for !p.Done() && !p.Is(EOL) {
		p.Next()
	}
}

func (p *Parser) parse() (Statement, error) {
	var (
		com Commented
		err error
	)
	for p.Is(Comment) {
		com.Before = append(com.Before, p.GetCurrLiteral())
		p.Next()
	}
	if p.Is(Macro) {
		if err := p.ParseMacro(); err != nil {
			return nil, err
		}
		return p.Parse()
	}
	if com.Statement, err = p.ParseStatement(); err != nil {
		return nil, err
	}
	if !p.Is(EOL) {
		return nil, p.wantError("statement", ";")
	}
	eol := p.curr
	p.Next()
	if p.Is(Comment) && eol.Line == p.curr.Line {
		com.After = p.GetCurrLiteral()
		p.Next()
	}
	return com.Statement, nil
}

func (p *Parser) RegisterParseFunc(kw string, fn func() (Statement, error)) {
	kw = strings.ToUpper(kw)
	p.keywords[kw] = fn
}

func (p *Parser) UnregisterParseFunc(kw string) {
	kw = strings.ToUpper(kw)
	delete(p.keywords, kw)
}

func (p *Parser) UnregisterAllParseFunc() {
	p.keywords = make(map[string]func() (Statement, error))
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
	if !p.Is(Lparen) {
		return nil, nil
	}
	p.Next()

	var (
		list []string
		err  error
	)

	for !p.Done() && !p.Is(Rparen) {
		if !p.curr.isValue() {
			return nil, p.Unexpected("columns")
		}
		list = append(list, p.GetCurrLiteral())
		p.Next()
		if err := p.EnsureEnd("columns", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("columns")
	}
	p.Next()

	return list, err
}

func (p *Parser) IsKeyword(kw string) bool {
	return p.curr.Type == Keyword && p.curr.Literal == kw
}

func (p *Parser) wantError(ctx, str string) error {
	return fmt.Errorf("%s: expected %q at %d:%d! got %s", ctx, str, p.curr.Line, p.curr.Column, p.curr.Literal)
}

func (p *Parser) Unexpected(ctx string) error {
	return p.UnexpectedDialect(ctx, "lang")
}

func (p *Parser) UnexpectedDialect(ctx, dialect string) error {
	return wrapErrorWithDialect(dialect, ctx, unexpected(p.curr))
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
		return i < len(kind) && kind[i] == p.curr.Type
	}
}

func (p *Parser) KwCheck(str ...string) func() bool {
	sort.Strings(str)
	return func() bool {
		if !p.Is(Keyword) {
			return false
		}
		if len(str) == 1 {
			return str[0] == p.curr.Literal
		}
		i := sort.SearchStrings(str, p.curr.Literal)
		return i < len(str) && str[i] == p.curr.Literal
	}
}

func (p *Parser) setParseFunc() {
	p.keywords = make(map[string]func() (Statement, error))
	p.RegisterParseFunc("SELECT", p.ParseSelect)
	p.RegisterParseFunc("VALUES", p.ParseValues)
	p.RegisterParseFunc("DELETE FROM", p.ParseDelete)
	p.RegisterParseFunc("TRUNCATE", p.ParseTruncate)
	p.RegisterParseFunc("TRUNCATE TABLE", p.ParseTruncate)
	p.RegisterParseFunc("UPDATE", p.ParseUpdate)
	p.RegisterParseFunc("MERGE", p.ParseMerge)
	p.RegisterParseFunc("INSERT INTO", p.ParseInsert)
	p.RegisterParseFunc("WITH", p.parseWith)
	p.RegisterParseFunc("CASE", p.ParseCase)
	p.RegisterParseFunc("IF", p.parseIf)
	p.RegisterParseFunc("WHILE", p.parseWhile)
	p.RegisterParseFunc("DECLARE", p.parseDeclare)
	p.RegisterParseFunc("SET", p.parseSet)
	p.RegisterParseFunc("RETURN", p.parseReturn)
	p.RegisterParseFunc("BEGIN", p.ParseBegin)
	p.RegisterParseFunc("START TRANSACTION", p.parseStartTransaction)
	p.RegisterParseFunc("CALL", p.ParseCall)
	p.RegisterParseFunc("CREATE TABLE", p.ParseCreateTable)
	p.RegisterParseFunc("CREATE TEMP TABLE", p.ParseCreateTable)
	p.RegisterParseFunc("CREATE TEMPORARY TABLE", p.ParseCreateTable)
	p.RegisterParseFunc("CREATE PROCEDURE", p.ParseCreateProcedure)
	p.RegisterParseFunc("CREATE OR REPLACE PROCEDURE", p.ParseCreateProcedure)
	p.RegisterParseFunc("ALTER TABLE", p.ParseAlterTable)
	p.RegisterParseFunc("DROP", p.ParseDropTable)
	p.RegisterParseFunc("DROP TABLE", p.ParseDropTable)
}

func (p *Parser) setFuncSetForTable() {
	prefix := newFuncSet[prefixFunc]()
	prefix.Register("", Ident, p.ParseIdent)
	prefix.Register("", Lparen, p.parseGroupExpr)
	prefix.Register("ROW", Keyword, p.ParseRow)

	p.prefix.Push(prefix)

	infix := newFuncSet[infixFunc]()
	p.infix.Push(infix)
}

func (p *Parser) setDefaultFuncSet() {
	infix := newFuncSet[infixFunc]()
	infix.Register("", Plus, p.parseInfixExpr)
	infix.Register("", Minus, p.parseInfixExpr)
	infix.Register("", Slash, p.parseInfixExpr)
	infix.Register("", Star, p.parseInfixExpr)
	infix.Register("", Concat, p.parseInfixExpr)
	infix.Register("", Eq, p.parseInfixExpr)
	infix.Register("", Ne, p.parseInfixExpr)
	infix.Register("", Lt, p.parseInfixExpr)
	infix.Register("", Le, p.parseInfixExpr)
	infix.Register("", Gt, p.parseInfixExpr)
	infix.Register("", Ge, p.parseInfixExpr)
	infix.Register("", Lparen, p.parseCallExpr)
	infix.Register("AND", Keyword, p.parseKeywordExpr)
	infix.Register("OR", Keyword, p.parseKeywordExpr)
	infix.Register("NOT", Keyword, p.parseKeywordExpr)
	infix.Register("LIKE", Keyword, p.parseKeywordExpr)
	infix.Register("SIMILAR", Keyword, p.parseKeywordExpr)
	infix.Register("ILIKE", Keyword, p.parseKeywordExpr)
	infix.Register("BETWEEN", Keyword, p.parseKeywordExpr)
	infix.Register("COLLATE", Keyword, p.parseCollateExpr)
	infix.Register("IN", Keyword, p.parseKeywordExpr)
	infix.Register("IS", Keyword, p.parseKeywordExpr)
	infix.Register("ISNULL", Keyword, p.parseKeywordExpr)
	infix.Register("NOTNULL", Keyword, p.parseKeywordExpr)
	infix.Register("ALL", Keyword, p.parseKeywordExpr)

	p.infix.Push(infix)

	prefix := newFuncSet[prefixFunc]()
	prefix.Register("", Ident, p.ParseIdentifier)
	prefix.Register("", Star, p.ParseIdentifier)
	prefix.Register("", Literal, p.ParseLiteral)
	prefix.Register("", Number, p.ParseLiteral)
	prefix.Register("", Lparen, p.parseGroupExpr)
	prefix.Register("", Minus, p.parseUnary)
	prefix.Register("", Keyword, p.parseUnary)
	prefix.Register("NOT", Keyword, p.parseUnary)
	prefix.Register("NULL", Keyword, p.ParseConstant)
	prefix.Register("DEFAULT", Keyword, p.ParseConstant)
	prefix.Register("TRUE", Keyword, p.ParseConstant)
	prefix.Register("FALSE", Keyword, p.ParseConstant)
	prefix.Register("CASE", Keyword, p.ParseCase)
	prefix.Register("SELECT", Keyword, p.ParseStatement)
	prefix.Register("CAST", Keyword, p.ParseCast)
	prefix.Register("ROW", Keyword, p.ParseRow)
	prefix.Register("EXISTS", Keyword, p.parseExists)

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
	scan *Scanner
	set  KeywordSet

	base string
	curr Token
	peek Token
}

func createFrame(r io.Reader, set KeywordSet) (*frame, error) {
	scan, err := Scan(r, set)
	if err != nil {
		return nil, err
	}
	f := frame{
		scan: scan,
		set:  set,
	}
	if n, ok := r.(interface{ Name() string }); ok {
		f.base = filepath.Dir(n.Name())
	}
	f.Next()
	f.Next()
	return &f, nil
}

func (f *frame) Curr() Token {
	return f.curr
}

func (f *frame) Peek() Token {
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
	return f.Is(EOF)
}

func (f *frame) Is(kind rune) bool {
	return f.curr.Type == kind
}

func (f *frame) peekIs(kind rune) bool {
	return f.peek.Type == kind
}
