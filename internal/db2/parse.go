package db2

import (
	"io"

	"github.com/midbel/sweet/internal/lang/parser"
)

func Parse(r io.Reader) (*parser.Parser, error) {
	scan, err := Scan(r)
	if err != nil {
		return nil, err
	}
	ps, err := parser.ParseWithScanner(scan)
	return ps, err
}
