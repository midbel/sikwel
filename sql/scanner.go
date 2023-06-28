package sql

import (
	"io"
	"fmt"
	"bytes"
	"unicode/utf8"
)

const (
	EOF rune = -(iota + 1)
	Comment
	Ident
	Literal
	Number
	Comma
	Lparen
	Rparen
	Dot
	Star
	Invalid
)

type Token struct {
	Type    rune
	Literal string
	Offset  int
	Position
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	case EOF:
		return "<eof>"
	case Comma:
		return "<comma>"
	case Lparen:
		return "<lparen>"
	case Rparen:
		return "<rparen>"
	case Dot:
		return "<dot>"
	case Star:
		return "<star>"
	case Ident:
		prefix = "identifier"
	case Literal:
		prefix = "literal"
	case Number:
		prefix = "number"
	case Comment:
		prefix = "comment"
	case Invalid:
		return "<invalid>"
	default:
		prefix = "unknown"
	}
	return fmt.Sprintf("%s(%s)", prefix, t.Literal)
}

type Position struct {
	Line   int
	Column int
}

type cursor struct {
	char rune
	curr int
	next int
	Position
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
	s.cursor.Line = 1
	s.read()
	s.skip(isBlank)
	return &s, nil
}

func (s *Scanner) Scan() Token {
	defer s.reset()
	s.skip(isSpace)

	var tok Token
	tok.Offset = s.curr
	tok.Position = s.cursor.Position
	if s.done() {
		tok.Type = EOF
		return tok
	}
	curr := s.char
	switch {
	case isNL(s.char):
		s.scanNL(&tok)
	case isComment(s.char):
		s.scanComment(&tok)
	case isLetter(s.char):
		s.scanIdent(&tok)
	case isQuote(s.char):
		s.scanString(&tok)
	case isDigit(s.char):
		s.scanNumber(&tok)
	case isPunct(s.char):
		s.scanPunct(&tok)
	default:
		tok.Type = Invalid
	}
	fmt.Printf("%[1]c (%02[1]x: %[2]s\n", curr, tok)
	return tok
}

func (s *Scanner) scanNL(tok *Token) {
	s.skip(isBlank)
	tok.Type = Comma
}

func (s *Scanner) scanComment(tok *Token) {
	s.read()
	s.read()
	s.skip(isBlank)
	for !isNL(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = Comment
}

func (s *Scanner) scanIdent(tok *Token) {
	for !isDelim(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = Ident
}

func (s *Scanner) scanString(tok *Token) {
	quote := s.char
	s.read()
	for !isQuote(s.char) && s.char != quote && !s.done() {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = Literal
	if !isQuote(s.char) && s.char != quote {
		tok.Type = Invalid
	}
	if tok.Type == Literal {
		s.read()
	}
}

func (s *Scanner) scanNumber(tok *Token) {
	for isDigit(s.char) && !s.done() {
		s.write()
		s.read()
	}
	if s.char == dot {
		s.write()
		s.read()
		for isDigit(s.char) && !s.done() {
			s.write()
			s.read()
		}
	}
	tok.Literal = s.literal()
	tok.Type = Number
}

func (s *Scanner) scanPunct(tok *Token) {
	switch s.char {
	case dot:
		tok.Type = Dot
	case star:
		tok.Type = Star
	case lparen:
		tok.Type = Lparen
	case rparen:
		tok.Type = Rparen
	case comma:
		tok.Type = Comma
	default:
		tok.Type = Invalid
	}
	s.read()
	s.save()
	s.skip(isBlank)
	if tok.Type == Rparen && s.char != rparen {
		s.restore()
	}
}

func (s *Scanner) done() bool {
	return s.char == utf8.RuneError || s.char == 0
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
	s.old.Position = s.cursor.Position
	if r == nl {
		s.cursor.Line++
		s.cursor.Column = 0
	}
	s.cursor.Column++
	s.char, s.curr, s.next = r, s.next, s.next+n
}

func (s *Scanner) unread() {
	c, z := utf8.DecodeRune(s.input[s.curr:])
	s.char, s.curr, s.next = c, s.curr-z, s.curr
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

func (s *Scanner) save() {
	s.old = s.cursor
}

func (s *Scanner) restore() {
	s.cursor = s.old
}

const (
	minus      = '-'
	comma      = ','
	lparen     = '('
	rparen     = ')'
	space      = ' '
	tab        = '\t'
	nl         = '\n'
	cr         = '\r'
	squote     = '\''
	dquote     = '"'
	underscore = '_'
	pound      = '#'
	dot        = '.'
	star       = '*'
)

func isDelim(r rune) bool {
	return isBlank(r) || isPunct(r)
}

func isComment(r rune) bool {
	return r == pound
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isAlpha(r rune) bool {
	return isLetter(r) || isDigit(r) || r == underscore
}

func isSpace(r rune) bool {
	return r == space || r == tab
}

func isQuote(r rune) bool {
	return r == squote || r == dquote
}

func isPunct(r rune) bool {
	return r == comma || r == lparen || r == rparen || r == dot || r == star
}

func isNL(r rune) bool {
	return r == nl || r == cr
}

func isBlank(r rune) bool {
	return isSpace(r) || isNL(r)
}