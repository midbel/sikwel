package sql

import (
	"fmt"
	"io"

	"github.com/midbel/sweet"
)

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	functions map[string]func() (sweet.Statement, error)
}

func NewParser(r io.Reader) (*Parser, error) {
	scan, err := Scan(r)
	if err != nil {
		return nil, err
	}
	p := Parser{
		scan: scan,
	}
	p.functions = map[string]func() (sweet.Statement, error){
		"select": p.parseSelect,
		"insert": p.parseInsert,
		"update": p.parseUpdate,
		"delete": p.parseDelete,
	}
	p.next()
	p.next()

	return &p, nil
}

func (p *Parser) Parse() (sweet.Statement, error) {
	if p.done() {
		return nil, io.EOF
	}
	return p.parse()
}

func (p *Parser) parse() (sweet.Statement, error) {
	if !p.is(Ident) {
		return nil, p.unexpected()
	}
	parse, ok := p.functions[p.curr.Literal]
	if !ok {
		return nil, p.unexpected()
	}
	return withParens[sweet.Statement](p, parse)
}

func (p *Parser) parseSelect() (sweet.Statement, error) {
	var (
		stmt      sweet.SelectStatement
		functions = map[string]parseFunc{
			"alias": p.parseAlias,
			"from":  p.parseFrom,
			"where": p.parseWhere,
		}
	)
	for !p.done() && !p.is(Rparen) {
		switch {
		case p.is(Literal) || p.is(Number):
		case p.is(Ident):
			parse, ok := functions[p.curr.Literal]
			if !ok {
				parse = p.parseIdent
			}
			_, err := withParens(p, parse)
			if err != nil {
				return nil, err
			}
		default:
			return nil, p.unexpected()
		}
		if err := p.ensureEOL(); err != nil {
			return nil, err
		}
	}
	return stmt, nil
}

func (p *Parser) parseIdent() (sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) parseAlias() (sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) parseFrom() ([]sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) parseWhere() (sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) parseUpdate() (sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) parseInsert() (sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) parseDelete() (sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) unexpected() error {
	return fmt.Errorf("unexpected token %s", p.curr)
}

func (p *Parser) is(kind rune) bool {
	return p.curr.Type == kind
}

func (p *Parser) done() bool {
	return p.is(EOF)
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

func (p *Parser) ensureEOL() error {
	switch {
	case p.is(Comma):
		p.next()
	case p.is(Rparen):
	default:
		return p.unexpected()
	}
	return nil
}

type parseFunc[T sweet.Statement | []sweet.Statement] func() (T, error)

func withParens[T sweet.Statement | []sweet.Statement](p *Parser, parse parseFunc) (ret T, err error) {
	p.next()
	if !p.is(Lparen) {
		return ret, p.unexpected()
	}
	p.next()

	if ret, err = parse(); err != nil {
		return
	}
	if !p.is(Rparen) {
		return ret, p.unexpected()
	}
	p.next()
	return
}
