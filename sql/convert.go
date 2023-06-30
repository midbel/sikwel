package sql

import (
	"errors"
	"io"

	"github.com/midbel/sweet"
)

func Convert(r io.Reader, w io.Writer) error {
	p, err := NewParser(r)
	if err != nil {
		return err
	}
	ws := sweet.NewWriter(w)
	for {
		stmt, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if err := ws.FormatStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}
