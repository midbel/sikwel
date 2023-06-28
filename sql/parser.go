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
		stmt sweet.SelectStatement
		err  error
	)
	for !p.done() && !p.is(Rparen) {
		switch {
		case p.is(Literal) || p.is(Number):
			s := sweet.Value{
				Literal: p.curr.Literal,
			}
			stmt.Columns = append(stmt.Columns, s)
			p.next()
		case p.is(Star):
			s := sweet.Value{
				Literal: "*",
			}
			stmt.Columns = append(stmt.Columns, s)
			p.next()
		case p.is(Ident) && p.curr.Literal == "call":
			s, err1 := withParens(p, p.parseCall)
			if err1 != nil {
				err = err1
				break
			}
			stmt.Columns = append(stmt.Columns, s)
		case p.is(Ident) && p.curr.Literal == "select":
			s, err1 := p.parse()
			if err1 != nil {
				err = err1
				break
			}
			stmt.Columns = append(stmt.Columns, s)
		case p.is(Ident) && p.curr.Literal == "all":
			s, err1 := withParens(p, p.parseAll)
			if err1 != nil {
				err = err1
				break
			}
			stmt.Columns = append(stmt.Columns, s...)
		case p.is(Ident) && p.curr.Literal == "alias":
			s, err1 := withParens(p, p.parseAlias)
			if err1 != nil {
				err = err1
				break
			}
			stmt.Columns = append(stmt.Columns, s)
		case p.is(Ident) && p.curr.Literal == "from":
			stmt.Tables, err = withParens(p, p.parseFrom)
		case p.is(Ident) && p.curr.Literal == "where":
			stmt.Where, err = withParens(p, p.parseWhere)
		case p.is(Ident) && p.curr.Literal == "limit":
			// stmt.Limit, err = withParens(p, p.parseLimit)
		case p.is(Ident):
			s, err := p.parseIdent()
			if err == nil {
				stmt.Columns = append(stmt.Columns, s)
			}
		default:
			err = p.unexpected()
		}
		if err != nil {
			return nil, err
		}
		if err = p.ensureEOL(); err != nil {
			return nil, err
		}
	}
	return stmt, nil
}

func (p *Parser) parseIdent() (sweet.Statement, error) {
	var n sweet.Name
	if p.peekis(Dot) {
		n.Prefix = p.curr.Literal
		p.next()
		p.next()
	}
	switch {
	case p.is(Star):
		n.Ident = "*"
	case p.is(Ident):
		n.Ident = p.curr.Literal
	default:
		return nil, p.unexpected()
	}
	p.next()
	return n, nil
}

func (p *Parser) parseAll() ([]sweet.Statement, error) {
	var list []sweet.Statement
	for !p.done() && !p.is(Rparen) {
		if !p.is(Ident) {
			return nil, p.unexpected()
		}
		n := sweet.Name{
			Prefix: p.curr.Literal,
			Ident:  "*",
		}
		list = append(list, n)
		p.next()
		if err := p.ensureEOL(); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (p *Parser) parseCall() (sweet.Statement, error) {
	var stmt sweet.Call
	if !p.is(Ident) {
		return nil, p.unexpected()
	}
	stmt.Ident = sweet.Name{
		Ident: p.curr.Literal,
	}
	p.next()
	if !p.is(Comma) {
		return nil, p.unexpected()
	}
	p.next()
	for !p.done() && !p.is(Rparen) {
		arg, err := p.parseIdentOrValue()
		if err != nil {
			return nil, err
		}
		stmt.Args = append(stmt.Args, arg)
		if err = p.ensureEOL(); err != nil {
			return nil, err
		}
	}
	return stmt, nil
}

func (p *Parser) parseAlias() (sweet.Statement, error) {
	stmt, err := p.parseIdentOrValue()
	if err != nil {
		return nil, err
	}
	if !p.is(Comma) {
		return nil, p.unexpected()
	}
	p.next()
	if !p.is(Ident) {
		return nil, p.unexpected()
	}
	stmt = sweet.Alias{
		Statement: stmt,
		Alias:     p.curr.Literal,
	}
	p.next()
	return stmt, nil
}

func (p *Parser) parseIdentOrValue() (sweet.Statement, error) {
	var (
		stmt sweet.Statement
		err  error
	)

	switch {
	case p.is(Literal) || p.is(Number):
		stmt = sweet.Value{
			Literal: p.curr.Literal,
		}
		p.next()
	case p.is(Star):
		stmt = sweet.Value{
			Literal: "*",
		}
		p.next()
	case p.is(Ident) && p.curr.Literal == "call":
		stmt, err = withParens(p, p.parseCall)
	case p.is(Ident) && p.curr.Literal == "select":
		stmt, err = p.parse()
	case p.is(Ident):
		stmt, err = p.parseIdent()
	default:
		err = p.unexpected()
	}
	return stmt, err
}

func (p *Parser) parseFrom() ([]sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) parseWhere() (sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) parseLimit() (sweet.Statement, error) {
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
	return fmt.Errorf("unexpected token %s at %d:%d", p.curr, p.curr.Line, p.curr.Column)
}

func (p *Parser) is(kind rune) bool {
	return p.curr.Type == kind
}

func (p *Parser) peekis(kind rune) bool {
	return p.peek.Type == kind
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

func withParens[T sweet.Statement | []sweet.Statement](p *Parser, parse parseFunc[T]) (ret T, err error) {
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
