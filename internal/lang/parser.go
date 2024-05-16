package lang

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

type Parser struct {
	*frame
	stack []*frame

	level int

	keywords map[string]func() (Statement, error)
	infix    map[symbol]infixFunc
	prefix   map[symbol]prefixFunc
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
	p.keywords = make(map[string]func() (Statement, error))

	p.RegisterParseFunc("SELECT", p.ParseSelect)
	p.RegisterParseFunc("VALUES", p.ParseValues)
	p.RegisterParseFunc("DELETE FROM", p.ParseDelete)
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

	p.infix = make(map[symbol]infixFunc)
	p.RegisterInfix("", Plus, p.parseInfixExpr)
	p.RegisterInfix("", Minus, p.parseInfixExpr)
	p.RegisterInfix("", Slash, p.parseInfixExpr)
	p.RegisterInfix("", Star, p.parseInfixExpr)
	p.RegisterInfix("", Concat, p.parseInfixExpr)
	p.RegisterInfix("", Eq, p.parseInfixExpr)
	p.RegisterInfix("", Ne, p.parseInfixExpr)
	p.RegisterInfix("", Lt, p.parseInfixExpr)
	p.RegisterInfix("", Le, p.parseInfixExpr)
	p.RegisterInfix("", Gt, p.parseInfixExpr)
	p.RegisterInfix("", Ge, p.parseInfixExpr)
	p.RegisterInfix("", Lparen, p.parseCallExpr)
	p.RegisterInfix("AND", Keyword, p.parseKeywordExpr)
	p.RegisterInfix("OR", Keyword, p.parseKeywordExpr)
	p.RegisterInfix("LIKE", Keyword, p.parseKeywordExpr)
	p.RegisterInfix("ILIKE", Keyword, p.parseKeywordExpr)
	p.RegisterInfix("BETWEEN", Keyword, p.parseKeywordExpr)
	p.RegisterInfix("COLLATE", Keyword, p.parseCollateExpr)
	p.RegisterInfix("AS", Keyword, p.parseKeywordExpr)
	p.RegisterInfix("IN", Keyword, p.parseKeywordExpr)
	p.RegisterInfix("IS", Keyword, p.parseKeywordExpr)
	p.RegisterInfix("NOT", Keyword, p.parseKeywordExpr)

	p.prefix = make(map[symbol]prefixFunc)
	p.RegisterPrefix("", Ident, p.ParseIdent)
	p.RegisterPrefix("", Star, p.ParseIdentifier)
	p.RegisterPrefix("", Literal, p.ParseLiteral)
	p.RegisterPrefix("", Number, p.ParseLiteral)
	p.RegisterPrefix("", Lparen, p.parseGroupExpr)
	p.RegisterPrefix("", Minus, p.parseUnary)
	p.RegisterPrefix("", Keyword, p.parseUnary)
	p.RegisterPrefix("NOT", Keyword, p.parseUnary)
	p.RegisterPrefix("NULL", Keyword, p.parseUnary)
	p.RegisterPrefix("DEFAULT", Keyword, p.parseUnary)
	p.RegisterPrefix("CASE", Keyword, p.ParseCase)
	p.RegisterPrefix("SELECT", Keyword, p.ParseStatement)
	p.RegisterPrefix("EXISTS", Keyword, p.parseUnary)
	p.RegisterPrefix("CAST", Keyword, p.ParseCast)
	p.RegisterPrefix("ROW", Keyword, p.ParseRow)

	return &p, nil
}

func (p *Parser) Parse() (Statement, error) {
	stmt, err := p.parse()
	if err != nil {
		p.restore()
	}
	return stmt, err
}

func (p *Parser) ParseStatement() (Statement, error) {
	p.Enter()
	defer p.Leave()

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

func (p *Parser) StartExpression() (Statement, error) {
	return p.parseExpression(powLowest)
}

func (p *Parser) Enter() {
	p.level++
}

func (p *Parser) Leave() {
	p.level--
}

func (p *Parser) Nested() bool {
	return p.level >= 1
}

func (p *Parser) QueryEnds() bool {
	if p.Nested() {
		return p.Is(Rparen)
	}
	return p.Is(EOL)
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
	p.prefix[symbolFor(kind, literal)] = fn
}

func (p *Parser) UnregisterPrefix(literal string, kind rune) {
	delete(p.prefix, symbolFor(kind, literal))
}

func (p *Parser) RegisterInfix(literal string, kind rune, fn infixFunc) {
	p.infix[symbolFor(kind, literal)] = fn
}

func (p *Parser) UnregisterInfix(literal string, kind rune) {
	delete(p.infix, symbolFor(kind, literal))
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
		list = append(list, p.curr.Literal)
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

func (p *Parser) currBinding() int {
	return bindings[p.curr.asSymbol()]
}

func (p *Parser) peekBinding() int {
	return bindings[p.peek.asSymbol()]
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
