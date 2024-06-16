package parser

import (
	"os"
	"path/filepath"

	"github.com/midbel/sweet/internal/token"
)

func (p *Parser) ParseMacro() error {
	var err error
	switch p.GetCurrLiteral() {
	case "INCLUDE":
		err = p.ParseIncludeMacro()
	case "DEFINE":
		err = p.ParseDefineMacro()
	case "USE":
		err = p.ParseUseMacro()
	case "ENV":
		err = p.ParseEnvMacro()
	case "VAR":
		err = p.ParseVarMacro()
	default:
		err = p.Unexpected("macro")
	}
	if err != nil {
		return err
	}
	return nil
}

func (p *Parser) ParseIncludeMacro() error {
	p.Next()

	file := filepath.Join(p.Base(), p.GetCurrLiteral())
	p.Next()

	if !p.Is(token.EOL) {
		return p.wantError("include", ";")
	}
	p.Next()

	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	frame, err := p.factory.Create(r)
	if err != nil {
		return err
	}
	p.stack = append(p.stack, p.Frame)
	p.Frame = frame

	return nil
}

// define a query in a SQL script and reuse it via the use macro
func (p *Parser) ParseDefineMacro() error {
	return nil
}

// use a query define via the define macro
func (p *Parser) ParseUseMacro() error {
	return nil
}

// use value from a variable given to a sql script
func (p *Parser) ParseVarMacro() error {
	return nil
}

// use value from an environment variable
func (p *Parser) ParseEnvMacro() error {
	return nil
}
