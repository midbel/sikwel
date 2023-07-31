package lang

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"
)

type Scanner struct {
	input []byte
	cursor
	old cursor

	keywords     KeywordSet
	str          bytes.Buffer
	starredIdent bool
}

func Scan(r io.Reader, keywords KeywordSet) (*Scanner, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	buf, _ = bytes.CutPrefix(buf, []byte{0xef, 0xbb, 0xbf})
	s := Scanner{
		input:    buf,
		keywords: keywords,
	}
	s.cursor.Line = 1
	s.keywords.prepare()
	s.read()
	s.skip(isBlank)
	return &s, nil
}

func (s *Scanner) Scan() Token {
	defer s.reset()
	s.skip(isBlank)

	var tok Token
	tok.Offset = s.curr
	tok.Position = s.cursor.Position
	if s.done() {
		tok.Type = EOF
		return tok
	}
	switch {
	case isComment(s.char, s.peek()):
		s.scanComment(&tok)
	case isLetter(s.char):
		s.scanIdent(&tok, false)
	case s.starredIdent && isStarLetter(s.char, s.peek()):
		s.scanIdent(&tok, true)
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
		tok.Type = Invalid
	}
	return tok
}

func (s *Scanner) scanMacro(tok *Token) {
	s.read()
	for !s.done() && !isDelim(s.char) {
		s.write()
		s.read()
	}
	tok.Type = Macro
	tok.Literal = strings.ToUpper(s.literal())
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

func (s *Scanner) scanIdent(tok *Token, star bool) {
	if star {
		s.read()
	}
	for !isDelim(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = Ident

	if !star {
		s.scanKeyword(tok)
	}
}

func (s *Scanner) scanQuotedIdent(tok *Token) {
	s.read()
	for !isIdentQ(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Type = Ident
	tok.Literal = s.literal()
	if !isIdentQ(s.char) {
		tok.Type = Invalid
	}
	if tok.Type == Ident {
		s.read()
	}
}

func (s *Scanner) scanKeyword(tok *Token) {
	list := []string{tok.Literal}
	if kw, ok := s.keywords.Is(list); !ok && kw == "" {
		return
	}
	tok.Type = Keyword
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
		tok.Type = Keyword
	}
}

func (s *Scanner) scanUntil(until func(rune) bool) {
	s.reset()
	for !s.done() && !until(s.char) {
		s.write()
		s.read()
	}
}

func (s *Scanner) scanString(tok *Token) {
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
	tok.Type = Literal
	if !isLiteralQ(s.char) {
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
	case lparen:
		tok.Type = Lparen
	case rparen:
		tok.Type = Rparen
	case comma:
		tok.Type = Comma
	case semicolon:
		tok.Type = EOL
	case star:
		tok.Type = Star
	case dot:
		tok.Type = Dot
	default:
	}
	s.read()
}

func (s *Scanner) scanOperator(tok *Token) {
	switch s.char {
	case percent:
		tok.Type = Mod
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = ModAssign
		}
	case equal:
		tok.Type = Eq
		if k := s.peek(); k == rangle {
			s.read()
			tok.Type = Arrow
		}
	case langle:
		tok.Type = Lt
		if k := s.peek(); k == rangle {
			s.read()
			tok.Type = Ne
		} else if k == equal {
			s.read()
			tok.Type = Le
		} else if k == langle {
			s.read()
			tok.Type = Lshift
		}
	case rangle:
		tok.Type = Gt
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = Ge
		} else if k == rangle {
			s.read()
			tok.Type = Rshift
		}
	case bang:
		tok.Type = Invalid
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = Ne
		}
	case slash:
		tok.Type = Slash
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = DivAssign
		}
	case plus:
		tok.Type = Plus
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = AddAssign
		}
	case minus:
		tok.Type = Minus
		if k := s.peek(); k == equal {
			s.read()
			tok.Type = MinAssign
		}
	case pipe:
		tok.Type = BitOr
		if k := s.peek(); k == pipe {
			s.read()
			tok.Type = Concat
		}
	case ampersand:
		tok.Type = BitAnd
	case tilde:
		tok.Type = BitXor
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
		s.cursor.Line++
		s.cursor.Column = 0
	}
	s.cursor.Column++
	s.char, s.curr, s.next = r, s.next, s.next+n
}

// func (s *Scanner) unread() {
// 	c, z := utf8.DecodeRune(s.input[s.curr:])
// 	s.char, s.curr, s.next = c, s.curr-z, s.curr
// }

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

func isStarLetter(r, k rune) bool {
	return r == star && isLetter(k)
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// func isAlpha(r rune) bool {
// 	return isLetter(r) || isDigit(r) || r == underscore
// }

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
