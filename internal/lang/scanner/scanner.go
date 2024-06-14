package scanner

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/midbel/sweet/internal/keywords"
	"github.com/midbel/sweet/internal/token"
)

type Scanner struct {
	input []byte
	cursor
	old cursor

	keywords keywords.Set
	str      bytes.Buffer
}

func Scan(r io.Reader, keywords keywords.Set) (*Scanner, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	buf, _ = bytes.CutPrefix(buf, []byte{0xef, 0xbb, 0xbf})
	s := Scanner{
		input:    buf,
		keywords: keywords,
	}
	s.cursor.Position.Line = 1
	s.keywords.Prepare()
	s.read()
	s.skip(isBlank)
	return &s, nil
}

func (s *Scanner) Scan() token.Token {
	defer s.reset()
	s.skip(isBlank)

	var tok token.Token
	tok.Offset = s.curr
	tok.Position = s.cursor.Position
	if s.done() {
		tok.Type = token.EOF
		return tok
	}
	switch {
	case isComment(s.char, s.peek()):
		s.scanComment(&tok)
	case isLetter(s.char):
		s.scanIdent(&tok, false)
	case isIdentQ(s.char):
		s.scanQuotedIdent(&tok)
	case isLiteralQ(s.char):
		s.scanString(&tok)
	case isDigit(s.char):
		s.scanNumber(&tok)
	case isPunct(s.char):
		s.scanPunct(&tok)
	case isOperator(s.char):
		s.scanOperator(&tok)
	case isMacro(s.char):
		s.scanMacro(&tok)
	default:
		tok.Type = token.Invalid
	}
	return tok
}

func (s *Scanner) scanMacro(tok *token.Token) {
	s.read()
	for !s.done() && !isDelim(s.char) {
		s.write()
		s.read()
	}
	tok.Type = token.Macro
	tok.Literal = strings.ToUpper(s.literal())
}

func (s *Scanner) scanComment(tok *token.Token) {
	s.read()
	s.read()
	s.skip(isBlank)
	for !isNL(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = token.Comment
}

func (s *Scanner) scanIdent(tok *token.Token, star bool) {
	if star {
		s.read()
	}
	for !isDelim(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = token.Ident

	if !star {
		s.scanKeyword(tok)
	}
}

func (s *Scanner) scanQuotedIdent(tok *token.Token) {
	s.read()
	for !isIdentQ(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Type = token.Ident
	tok.Literal = s.literal()
	if !isIdentQ(s.char) {
		tok.Type = token.Invalid
	}
	if tok.Type == token.Ident {
		s.read()
	}
}

func (s *Scanner) scanKeyword(tok *token.Token) {
	list := []string{tok.Literal}
	if kw, ok := s.keywords.Is(list); !ok && kw == "" {
		return
	}
	tok.Type = token.Keyword
	tok.Literal = strings.ToUpper(tok.Literal)
	for !s.done() && !(isPunct(s.char) || isOperator(s.char)) {
		s.save()

		s.skip(isBlank)
		s.scanUntil(isDelim)
		if len(s.literal()) == 0 {
			s.restore()
			break
		}
		list = append(list, strings.ToLower(s.literal()))

		res, _ := s.keywords.Is(list)
		if res == "" {
			s.restore()
			return
		}
		tok.Literal = strings.ToUpper(res)
		tok.Type = token.Keyword
	}
}

func (s *Scanner) scanUntil(until func(rune) bool) {
	s.reset()
	for !s.done() && !until(s.char) {
		s.write()
		s.read()
	}
}

func (s *Scanner) scanString(tok *token.Token) {
	s.read()
	for !isLiteralQ(s.char) && !s.done() {
		s.write()
		s.read()
		if s.char == squote && s.peek() == s.char {
			s.write()
			s.read()
			s.write()
			s.read()
		}
	}
	tok.Literal = s.literal()
	tok.Type = token.Literal
	if !isLiteralQ(s.char) {
		tok.Type = token.Invalid
	}
	if tok.Type == token.Literal {
		s.read()
	}
}

func (s *Scanner) scanNumber(tok *token.Token) {
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
	tok.Type = token.Number
}

func (s *Scanner) scanPunct(tok *token.Token) {
	switch s.char {
	case lparen:
		tok.Type = token.Lparen
	case rparen:
		tok.Type = token.Rparen
	case comma:
		tok.Type = token.Comma
	case semicolon:
		tok.Type = token.EOL
	case star:
		tok.Type = token.Star
	case dot:
		tok.Type = token.Dot
	default:
	}
	s.read()
}

func (s *Scanner) scanOperator(tok *token.Token) {
	switch s.char {
	case percent:
		tok.Type = token.Mod
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = token.ModAssign
		}
	case equal:
		tok.Type = token.Eq
		if k := s.peek(); k == rangle {
			s.read()
			tok.Type = token.Arrow
		}
	case langle:
		tok.Type = token.Lt
		if k := s.peek(); k == rangle {
			s.read()
			tok.Type = token.Ne
		} else if k == equal {
			s.read()
			tok.Type = token.Le
		} else if k == langle {
			s.read()
			tok.Type = token.Lshift
		}
	case rangle:
		tok.Type = token.Gt
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = token.Ge
		} else if k == rangle {
			s.read()
			tok.Type = token.Rshift
		}
	case bang:
		tok.Type = token.Invalid
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = token.Ne
		}
	case slash:
		tok.Type = token.Slash
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = token.DivAssign
		}
	case plus:
		tok.Type = token.Plus
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = token.AddAssign
		}
	case minus:
		tok.Type = token.Minus
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = token.MinAssign
		}
	case pipe:
		tok.Type = token.BitOr
		if k := s.peek(); k == pipe {
			s.read()
			tok.Type = token.Concat
		}
	case ampersand:
		tok.Type = token.BitAnd
	case tilde:
		tok.Type = token.BitXor
	default:
	}
	s.read()
}

func (s *Scanner) save() {
	s.old = s.cursor
}

func (s *Scanner) restore() {
	s.cursor = s.old
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
	if s.char == nl {
		s.cursor.Position.Line++
		s.cursor.Position.Column = 0
	}
	s.cursor.Position.Column++
	s.char, s.curr, s.next = r, s.next, s.next+n
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
	token.Position
}

const (
	minus      = '-'
	comma      = ','
	lparen     = '('
	rparen     = ')'
	space      = ' '
	tab        = '\t'
	semicolon  = ';'
	nl         = '\n'
	cr         = '\r'
	dquote     = '"'
	squote     = '\''
	underscore = '_'
	star       = '*'
	dot        = '.'
	equal      = '='
	langle     = '<'
	rangle     = '>'
	bang       = '!'
	pipe       = '|'
	ampersand  = '&'
	slash      = '/'
	plus       = '+'
	arobase    = '@'
	percent    = '%'
	tilde      = '~'
)

func isMacro(r rune) bool {
	return r == arobase
}

func isDelim(r rune) bool {
	return isBlank(r) || isPunct(r) || isOperator(r)
}

func isComment(r, k rune) bool {
	return r == minus && r == k
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isSpace(r rune) bool {
	return r == space || r == tab
}

func isIdentQ(r rune) bool {
	return r == dquote
}

func isLiteralQ(r rune) bool {
	return r == squote
}

func isPunct(r rune) bool {
	return r == comma || r == lparen || r == rparen || r == semicolon || r == star || r == dot
}

func isOperator(r rune) bool {
	return r == equal || r == langle || r == rangle || r == bang || r == slash || r == plus || r == minus || r == pipe || r == percent || r == ampersand || r == tilde
}

func isNL(r rune) bool {
	return r == nl || r == cr
}

func isBlank(r rune) bool {
	return isSpace(r) || isNL(r)
}
