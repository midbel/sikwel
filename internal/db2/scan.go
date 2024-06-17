package db2

import (
	"io"

	"github.com/midbel/sweet/internal/scanner"
	"github.com/midbel/sweet/internal/token"
)

type Scanner struct {
	inner *scanner.Scanner
}

func Scan(r io.Reader) (*Scanner, error) {
	inner, err := scanner.Scan(r, GetKeywords())
	if err != nil {
		return nil, err
	}
	s := Scanner{
		inner: inner,
	}
	return &s, err
}

func (s *Scanner) Scan() token.Token {
	return s.inner.Scan()
}

func (s *Scanner) scanStarIdent(tok *token.Token) {

}
