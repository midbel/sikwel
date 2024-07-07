package scanner

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
	question   = '?'
	colon      = ':'
	dollar     = '$'
)

func IsPlaceholder(r rune) bool {
	return r == question || r == colon || r == dollar
}

func IsMacro(r rune) bool {
	return r == arobase
}

func IsDelim(r rune) bool {
	return IsBlank(r) || IsPunct(r) || IsOperator(r)
}

func IsComment(r, k rune) bool {
	return r == minus && r == k
}

func IsLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func IsDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func IsSpace(r rune) bool {
	return r == space || r == tab
}

func IsIdentQ(r rune) bool {
	return r == dquote
}

func IsLiteralQ(r rune) bool {
	return r == squote
}

func IsPunct(r rune) bool {
	return r == comma || r == lparen || r == rparen || r == semicolon || r == star || r == dot
}

func IsOperator(r rune) bool {
	return r == equal || r == langle || r == rangle || r == bang || r == slash || r == plus || r == minus || r == pipe || r == percent || r == ampersand || r == tilde
}

func IsNL(r rune) bool {
	return r == nl || r == cr
}

func IsBlank(r rune) bool {
	return IsSpace(r) || IsNL(r)
}
