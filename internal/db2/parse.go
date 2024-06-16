package db2

import "io"

type Parser struct {
	*lang.Parser
}

func Parse(r io.Reader) (*Parser, error) {
	p, err := lang.NewParser(r)
	if err != nil {
		return nil, err
	}
	ps := Parser{
		Parser: p,
	}
	return &ps, nil
}
