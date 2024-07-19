package parser

import (
	"fmt"

	"github.com/midbel/sweet/internal/token"
)

type ParseError struct {
	Dialect string
	Query   string
	Err     error
}

func (e ParseError) Literal() string {
	p := e.Err.(TokenError)
	return p.Literal
}

func (e ParseError) Position() token.Position {
	p := e.Err.(TokenError)
	return p.Position
}

func (e ParseError) Error() string {
	return e.Err.Error()
}

type TokenError struct {
	token.Token
	Type string
}

func (e TokenError) Error() string {
	var (
		line   = e.Position.Line
		column = e.Position.Column
	)
	return fmt.Sprintf("at %d:%d, %s %s", line, column, e.Type, e.Token)
}

func unexpected(tok token.Token) error {
	return fmt.Errorf("unexpected token %s at (%d:%d)", tok, tok.Line, tok.Column)
}

func wrapErrorWithDialect(dialect, ctx string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s(%s): %w", dialect, ctx, err)
}

func wrapError(ctx string, err error) error {
	return wrapErrorWithDialect("lang", ctx, err)
}
