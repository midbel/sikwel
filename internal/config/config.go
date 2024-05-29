package config

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"
)

type Parser struct {
	scan *Scanner
	curr Token
	peek Token
}

func New(r io.Reader) (*Parser, error) {
	sc, err := Scan(r)
	if err != nil {
		return nil, err
	}
	ps := &Parser{
		scan: sc,
	}
	ps.next()
	ps.next()
	return ps, nil
}

func (p *Parser) Parse() error {
	for !p.isDone() {
		p.skip(Comment)
		if err := p.parse(); err != nil {
			return nil
		}
	}
	return nil
}

func (p *Parser) parse() error {
	if !p.isIdent() {
		return p.unexpected()
	}
	ident := p.curr
	p.next()
	if p.is(Equal) {
		return p.parseEqual(ident)
	}
	return p.parseObject(ident, false)
}

func (p *Parser) parseEqual(ident Token) error {
	p.next()
	switch {
	case p.is(BegArr):
		return p.parseArray(ident)
	case p.is(BegObj):
		return p.parseObject(ident, true)
	default:
		return p.parseLiteral(ident)
	}
}

func (p *Parser) parseLiteral(ident Token) error {
	for p.isValue() {
		p.next()
	}
	if !p.isEOL() {
		return p.unexpected()
	}
	p.next()
	return nil
}

func (p *Parser) parseArray(ident Token) error {
	p.next()
	for !p.isDone() && !p.is(EndArr) {
		if !p.isValue() {
			return p.unexpected()
		}
		p.next()
		switch {
		case p.is(Comma):
			p.next()
		case p.is(EndArr):
		default:
			return p.unexpected()
		}
	}
	if !p.is(EndArr) {
		return p.unexpected()
	}
	p.next()
	return nil
}

func (p *Parser) parseObject(ident Token, inline bool) error {
	if !inline {
		for p.isIdent() {
			p.next()
		}
	}
	if !p.is(BegObj) {
		return p.unexpected()
	}
	p.next()
	for !p.isDone() && !p.is(EndObj) {
		p.skip(Comment)
		if err := p.parse(); err != nil {
			return err
		}
	}
	if !p.is(EndObj) {
		return p.unexpected()
	}
	p.next()
	return nil
}

func (p *Parser) isDone() bool {
	return p.curr.Type == EOF
}

func (p *Parser) isEOL() bool {
	return p.is(EOL) || p.is(Comment)
}

func (p *Parser) isIdent() bool {
	return p.is(Ident) || p.is(String)
}

func (p *Parser) isValue() bool {
	return p.is(Ident) || p.is(String) || p.is(Number) || p.is(Bool)
}

func (p *Parser) is(kind rune) bool {
	return p.curr.Type == kind
}

func (p *Parser) skip(kind rune) {
	for p.is(kind) {
		p.next()
	}
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

func (p *Parser) unexpected() error {
	return fmt.Errorf("unexpected token: %s", p.curr)
}

type Position struct {
	Line   int
	Column int
}

const (
	EOF rune = -(iota + 1)
	EOL
	Blank
	Comment
	Ident
	String
	Number
	Bool
	Equal
	Comma
	BegArr
	EndArr
	BegObj
	EndObj
	Invalid
)

type Token struct {
	Literal string
	Type    rune
	Position
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	default:
		return "invalid"
	case EOF:
		return "<eof>"
	case EOL:
		return "<eol>"
	case Blank:
		return "<blank>"
	case Comment:
		prefix = "comment"
	case Ident:
		prefix = "identifier"
	case String:
		prefix = "string"
	case Number:
		prefix = "number"
	case Bool:
		prefix = "boolean"
	case Equal:
		return "<equal>"
	case Comma:
		return "<comma>"
	case BegArr:
		return "<beg-array>"
	case EndArr:
		return "<end-array>"
	case BegObj:
		return "<beg-object>"
	case EndObj:
		return "<end-object>"
	}
	return fmt.Sprintf("%s(%s)", prefix, t.Literal)
}

type Scanner struct {
	input []byte
	cursor
	old cursor

	str bytes.Buffer
}

func Scan(r io.Reader) (*Scanner, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	buf, _ = bytes.CutPrefix(buf, []byte{0xef, 0xbb, 0xbf})
	s := Scanner{
		input: buf,
	}
	s.Line = 1
	s.read()
	s.skip(isBlank)
	return &s, nil
}

func (s *Scanner) Scan() Token {
	defer s.reset()
	s.skip(isSpace)

	var tok Token
	tok.Position = s.Position
	if s.done() {
		tok.Type = EOF
		return tok
	}
	switch {
	case isComment(s.char):
		s.scanComment(&tok)
	case isLetter(s.char):
		s.scanIdent(&tok)
	case isDigit(s.char):
		s.scanNumber(&tok)
	case isQuote(s.char):
		s.scanString(&tok)
	case isGroup(s.char):
		s.scanGroup(&tok)
	case isEqual(s.char):
		s.scanEqual(&tok)
	case isPunct(s.char):
		s.scanPunct(&tok)
	case isNL(s.char):
		s.scanNL(&tok)
	default:
		tok.Type = Invalid
	}
	return tok
}

func (s *Scanner) scanComment(tok *Token) {
	s.read()
	s.skip(isSpace)
	for !s.done() && !isNL(s.char) {
		s.write()
		s.read()
	}
	tok.Type = Comment
	tok.Literal = s.literal()
	s.skip(isBlank)
}

func (s *Scanner) scanIdent(tok *Token) {
	for !s.done() && isAlpha(s.char) {
		s.write()
		s.read()
	}
	tok.Type = Ident
	tok.Literal = s.literal()
	switch tok.Literal {
	case "true", "false", "on", "off":
		tok.Type = Bool
	default:
	}
}

func (s *Scanner) scanNumber(tok *Token) {
	for !s.done() && isDigit(s.char) {
		s.write()
		s.read()
	}
	tok.Type = Number
	tok.Literal = s.literal()
}

func (s *Scanner) scanString(tok *Token) {
	quote := s.char
	s.read()
	for !s.done() && s.char != quote {
		s.write()
		s.read()
	}
	tok.Type = String
	tok.Literal = s.literal()
	if s.char != quote {
		tok.Type = Invalid
	} else {
		s.read()
	}
}

func (s *Scanner) scanGroup(tok *Token) {
	switch s.char {
	case lsquare:
		tok.Type = BegArr
	case rsquare:
		tok.Type = EndArr
	case lcurly:
		tok.Type = BegObj
	case rcurly:
		tok.Type = EndObj
	default:
		tok.Type = Invalid
	}
	s.read()
	s.skip(isBlank)
}

func (s *Scanner) scanEqual(tok *Token) {
	tok.Type = Equal
	s.read()
}

func (s *Scanner) scanPunct(tok *Token) {
	switch s.char {
	case comma:
		tok.Type = Comma
	case semicolon:
		tok.Type = EOL
	default:
		tok.Type = Invalid
	}
	s.read()
	s.skip(isBlank)
}

func (s *Scanner) scanNL(tok *Token) {
	tok.Type = EOL
	s.skip(isBlank)
}

func (s *Scanner) done() bool {
	return s.char == utf8.RuneError || s.char == 0
}

func (s *Scanner) unread() {
	s.old = s.cursor
	r, n := utf8.DecodeRune(s.input[s.curr:])
	s.char, s.next, s.curr = r, s.curr, s.curr-n
}

func (s *Scanner) read() {
	if s.curr >= len(s.input) {
		s.char = utf8.RuneError
		return
	}
	r, n := utf8.DecodeRune(s.input[s.next:])
	if r == utf8.RuneError {
		s.char = r
		s.next = len(s.input)
		return
	}
	if s.char == nl {
		s.cursor.Line++
		s.cursor.Column = 0
	}
	s.cursor.Column++
	s.char, s.curr, s.next = r, s.next, s.next+n
}

func (s *Scanner) save() {
	s.old = s.cursor
}

func (s *Scanner) restore() {
	s.cursor = s.old
}

func (s *Scanner) peek() rune {
	r, _ := utf8.DecodeRune(s.input[s.next:])
	return r
}

func (s *Scanner) reset() {
	s.str.Reset()
}

func (s *Scanner) write() {
	s.str.WriteRune(s.char)
}

func (s *Scanner) literal() string {
	return s.str.String()
}

func (s *Scanner) skip(accept func(rune) bool) {
	if s.done() {
		return
	}
	for accept(s.char) && !s.done() {
		s.read()
	}
}

type cursor struct {
	char rune
	curr int
	next int
	Position
}

const (
	comma      = ','
	space      = ' '
	tab        = '\t'
	semicolon  = ';'
	nl         = '\n'
	cr         = '\r'
	dquote     = '"'
	squote     = '\''
	underscore = '_'
	dot        = '.'
	equal      = '='
	colon      = ':'
	lsquare    = '['
	rsquare    = ']'
	lcurly     = '{'
	rcurly     = '}'
	pound      = '#'
)

func isComment(r rune) bool {
	return r == pound
}

func isEqual(r rune) bool {
	return r == equal || r == colon
}

func isPunct(r rune) bool {
	return r == comma || r == semicolon
}

func isGroup(r rune) bool {
	return r == lcurly || r == rcurly || r == lsquare || r == rsquare
}

func isQuote(r rune) bool {
	return r == squote || r == dquote
}

func isAlpha(r rune) bool {
	return isLetter(r) || isDigit(r)
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == underscore
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isSpace(r rune) bool {
	return r == space || r == tab
}

func isNL(r rune) bool {
	return r == nl || r == cr
}

func isBlank(r rune) bool {
	return isSpace(r) || isNL(r)
}