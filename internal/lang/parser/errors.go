package parser

import (
	"fmt"

	"github.com/midbel/sweet/internal/token"
)

type ParseError struct {
	Dialect string
	Query   string
	token.Position
	Err error
}

func (e ParseError) Error() string {
	return e.Err.Error()
}

func unexpected(tok token.Token) error {
	return fmt.Errorf("unexpected token %s at (%d:%d)", tok, tok.Line, tok.Column)
}

func wrapErrorWithDialect(dialect, ctx string, err error) error {
	if err == nil {
		return nil
	}
	err = fmt.Errorf("%s(%s): %w", dialect, ctx, err)
	return ParseError{
		Dialect: dialect,
		Query:   "TBD",
		Err:     err,
	}
	// return fmt.Errorf("%s(%s): %w", dialect, ctx, err)
}

func wrapError(ctx string, err error) error {
	return wrapErrorWithDialect("lang", ctx, err)
}
