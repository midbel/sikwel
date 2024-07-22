package parser

import (
	"fmt"
	"strings"

	"github.com/midbel/sweet/internal/token"
)

const (
	defaultReason      = "one or more errors have been detected in your query"
	missingOpenParen   = "missing opening parenthesis before expression/statement"
	missingCloseParen  = "missing closing parenthesis after expression/statement"
	keywordAfterComma  = "unexpected keyword after comma"
	missingOperator    = "missing operator after identifier"
	identExpected      = "a valid identifier is expected"
	valueExpected      = "a valid value expected (number, boolean, identifier)"
	missingEol         = "missing semicolon at end of statement"
	unknownOperator    = "unknown operator"
	macroOptionUnknown = "macro option unknown"
	syntaxError        = "syntax error"
)

func keywordExpected(kw ...string) string {
	return fmt.Sprintf("expected %s keyword(s)", strings.Join(kw, "|"))
}

type ParseError struct {
	token.Token
	Reason  string
	Context string
	Query   string
}

func (e ParseError) Literal() string {
	return e.Token.Literal
}

func (e ParseError) Position() token.Position {
	return e.Token.Position
}

func (e ParseError) Error() string {
	pos := e.Token.Position
	return fmt.Sprintf("[%s] at %d:%d, %s", e.Context, pos.Line, pos.Column, e.Reason)
}
