package lang

import (
	"fmt"
	"os"
	"path/filepath"
)

func (p *Parser) ParseMacro() error {
	var err error
	switch p.curr.Literal {
	case "INCLUDE":
		err = p.ParseIncludeMacro()
	case "DEFINE":
		err = p.ParseDefineMacro()
	case "USE":
		err = p.ParseUseMacro()
	case "INLINE":
	default:
		err = fmt.Errorf("macro %s unsupported", p.curr.Literal)
	}
	if err != nil {
		return err
	}
	return nil
}

func (p *Parser) ParseIncludeMacro() error {
	p.Next()

	file := filepath.Join(p.base, p.curr.Literal)
	p.Next()

	if !p.Is(EOL) {
		return p.wantError("include", ";")
	}
	p.Next()

	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	frame, err := createFrame(r, p.frame.set)
	if err != nil {
		return err
	}
	p.stack = append(p.stack, p.frame)
	p.frame = frame

	return nil
}

func (p *Parser) ParseDefineMacro() error {
	return nil
}

func (p *Parser) ParseUseMacro() error {
	return nil
}
