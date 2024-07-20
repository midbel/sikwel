package parser

import (
	"fmt"

	"github.com/midbel/sweet/internal/token"
)

type ParseError struct {
	token.Token
	Type    string
	Context string
	Dialect string
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
	return fmt.Sprintf("at %d:%d, %s %s", pos.Line, pos.Column, e.Type, e.Token)
}
