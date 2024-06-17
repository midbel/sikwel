package db2

import (
	"io"

	"github.com/midbel/sweet/internal/lang/parser"
)

type factory struct{}

func (_ factory) Create(r io.Reader) (*scanner.Frame, error) {
	scan, err := Scan(r, GetKeywords())
	if err != nil {
		return nil, err
	}
	_ = scan
	return nil, nil
}

type Parser struct {
	*parser.Parser
}

func Parse(r io.Reader) (*Parser, error) {
	p, err := parser.ParseWithFactory(r, nil)
	if err != nil {
		return nil, err
	}
	ps := Parser{
		Parser: p,
	}
	return &ps, nil
}
