package parser

import (
	"fmt"

	"github.com/midbel/sweet/internal/token"
)

const (
	defaultReason     = "one or more errors have been detected in your query"
	missingOpenParen  = "missing ( before parameters"
	missingCloseParen = "missing ) after paramters"
	keywordAfterComma = "unexpected keyword after comma"
	missingOperator   = "missing operator after identifier"
	identExpected     = "identifier expected"
	missingEol        = "missing semicolon at end of statement"
)

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
	return fmt.Sprintf("at %d:%d, unexpected token %s", pos.Line, pos.Column, e.Token)
}
