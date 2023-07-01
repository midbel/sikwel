package sql

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/midbel/sweet"
)

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	statements map[string]sweet.Statement
	functions  map[string]func(string) (sweet.Statement, error)
}

func NewParser(r io.Reader) (*Parser, error) {
	scan, err := Scan(r)
	if err != nil {
		return nil, err
	}
	p := Parser{
		scan:       scan,
		statements: make(map[string]sweet.Statement),
	}
	p.functions = map[string]func(string) (sweet.Statement, error){
		"select":    p.parseSelect,
		"insert":    p.parseInsert,
		"update":    p.parseUpdate,
		"delete":    p.parseDelete,
		"with":      p.parseWith,
		"union":     p.parseUnion,
		"intersect": p.parseIntersect,
		"except":    p.parseExcept,
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
	defer func() {
		if !p.is(Comma) {
			return
		}
		p.next()
	}()
	parse, ok := p.functions[p.curr.Literal]
	if !ok {
		err := p.parseMacro()
		if err != nil {
			return nil, err
		}
		return p.parse()
	}
	return withParens[sweet.Statement](p, parse)
}

func (p *Parser) parseMacro() error {
	switch {
	case p.check("include"):
		return p.parseInclude()
	case p.check("define"):
		return p.parseDefine()
	default:
		return p.unexpected()
	}
}

func (p *Parser) parseInclude() error {
	return nil
}

func (p *Parser) parseDefine() error {
	return nil
}

func (p *Parser) parseWith(_ string) (sweet.Statement, error) {
	var (
		with sweet.WithStatement
		err  error
	)
	for !p.done() && !p.is(Rparen) {
		switch {
		case p.is(Ident) && p.check("select", "insert", "update", "delete"):
			if with.Statement != nil {
				return nil, p.unexpected()
			}
			with.Statement, err = p.parse()
		case p.is(Ident):
			s, err1 := withParens(p, p.parseSubquery)
			if err1 != nil {
				err = err1
				break
			}
			with.Queries = append(with.Queries, s)
		default:
			return nil, p.unexpected()
		}
		if err != nil {
			return nil, err
		}
		if err = p.ensureEOL(); err != nil {
			return nil, err
		}
	}
	return with, err
}

func (p *Parser) parseSubquery(ident string) (sweet.Statement, error) {
	var (
		cte = sweet.CteStatement{
			Ident: ident,
		}
		err error
	)
	for !p.done() {
		if !p.is(Ident) {
			return nil, p.unexpected()
		}
		cte.Columns = append(cte.Columns, p.curr.Literal)
		p.next()
		if !p.is(Comma) {
			return nil, p.unexpected()
		}
		p.next()
		if ok := p.check("insert", "update", "delete", "select"); ok {
			break
		}
	}
	cte.Statement, err = p.parse()
	if err != nil {
		return nil, err
	}
	return cte, err
}

func (p *Parser) parseUnion(_ string) (sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) parseIntersect(_ string) (sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) parseExcept(_ string) (sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) parseSelect(_ string) (sweet.Statement, error) {
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
		case p.is(Ident) && p.check("call"):
			s, err1 := withParens(p, p.parseCall)
			if err1 != nil {
				err = err1
				break
			}
			stmt.Columns = append(stmt.Columns, s)
		case p.is(Ident) && p.check("select"):
			s, err1 := p.parse()
			if err1 != nil {
				err = err1
				break
			}
			stmt.Columns = append(stmt.Columns, s)
		case p.is(Ident) && p.check("all"):
			s, err1 := withParens(p, p.parseAll)
			if err1 != nil {
				err = err1
				break
			}
			stmt.Columns = append(stmt.Columns, s...)
		case p.is(Ident) && p.check("alias"):
			s, err1 := withParens(p, p.parseAlias)
			if err1 != nil {
				err = err1
				break
			}
			stmt.Columns = append(stmt.Columns, s)
		case p.is(Ident) && p.check("from"):
			stmt.Tables, err = withParens(p, p.parseFrom)
		case p.is(Ident) && p.check("where"):
			stmt.Where, err = withParens(p, p.parseWhere)
		case p.is(Ident) && p.check("having"):
			stmt.Having, err = withParens(p, p.parseHaving)
		case p.is(Ident) && p.check("limit"):
			stmt.Limit, err = withParens(p, p.parseLimit)
		case p.is(Ident) && p.check("asc", "desc"):
			s, err1 := withParens(p, p.parseOrderBy)
			if err1 != nil {
				err = err1
				break
			}
			stmt.Orders = append(stmt.Orders, s)
		case p.is(Ident) && p.check("groupby"):
			stmt.Groups, err = withParens(p, p.parseGroupBy)
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

func (p *Parser) parseAll(_ string) ([]sweet.Statement, error) {
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

func (p *Parser) parseCall(_ string) (sweet.Statement, error) {
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

func (p *Parser) parseAlias(_ string) (sweet.Statement, error) {
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
	case p.is(Ident) && p.check("call"):
		stmt, err = withParens(p, p.parseCall)
	case p.is(Ident) && p.check("select"):
		stmt, err = p.parse()
	case p.is(Ident):
		stmt, err = p.parseIdent()
	default:
		err = p.unexpected()
	}
	return stmt, err
}

func (p *Parser) parseLimit(_ string) (sweet.Statement, error) {
	var (
		lim sweet.Limit
		err error
	)
	lim.Count, err = strconv.Atoi(p.curr.Literal)
	if err != nil {
		return nil, p.unexpected()
	}
	if !p.is(Comma) {
		return lim, nil
	}
	p.next()
	lim.Offset, err = strconv.Atoi(p.curr.Literal)
	if err != nil {
		return nil, p.unexpected()
	}
	return lim, err
}

func (p *Parser) parseOrderBy(orient string) (sweet.Statement, error) {
	var (
		ord sweet.Order
		err error
	)
	ord.Orient = orient
	ord.Statement, err = p.parseIdent()
	if err != nil {
		return nil, err
	}
	if !p.is(Comma) {
		return ord, nil
	}
	p.next()
	if !p.is(Ident) && !p.check("first", "last") {
		return nil, p.unexpected()
	}
	ord.Nulls = p.curr.Literal
	p.next()
	return ord, err
}

func (p *Parser) parseGroupBy(_ string) ([]sweet.Statement, error) {
	var list []sweet.Statement
	for !p.done() && !p.is(Rparen) {
		stmt, err := p.parseIdent()
		if err != nil {
			return nil, err
		}
		if err = p.ensureEOL(); err != nil {
			return nil, err
		}
		list = append(list, stmt)
	}
	return list, nil
}

func (p *Parser) parseFrom(_ string) ([]sweet.Statement, error) {
	var (
		tables []sweet.Statement
		joins  []sweet.Statement
		err    error
	)
	for !p.done() && !p.is(Rparen) {
		var (
			stmt sweet.Statement
			join bool
		)
		switch {
		case p.is(Ident) && p.check("select"):
			stmt, err = withParens(p, p.parseSelect)
		case p.is(Ident) && p.check("alias"):
			stmt, err = withParens(p, p.parseAlias)
		case p.is(Ident) && p.check("join", "leftjoin", "rightjoin", "fulljoin"):
			stmt, err = withParens(p, p.parseJoin)
		case p.is(Ident):
			stmt, err = p.parseIdent()
		default:
			return nil, p.unexpected()
		}
		if err != nil {
			return nil, err
		}
		if err = p.ensureEOL(); err != nil {
			return nil, p.unexpected()
		}
		if join {
			joins = append(joins, stmt)
		} else {
			tables = append(tables, stmt)
		}
	}
	return append(tables, joins...), nil
}

func (p *Parser) parseJoin(kind string) (sweet.Statement, error) {
	var (
		stmt sweet.Join
		err  error
	)
	stmt.Type = joinMapping[kind]
	switch {
	case p.is(Ident) && p.check("select"):
		stmt.Table, err = withParens(p, p.parseSelect)
	case p.is(Ident) && p.check("alias"):
		stmt.Table, err = withParens(p, p.parseAlias)
	case p.is(Ident):
		stmt.Table, err = p.parseIdent()
	default:
		err = p.unexpected()
	}
	if err != nil {
		return nil, err
	}
	if !p.is(Comma) {
		return nil, p.unexpected()
	}
	p.next()
	stmt.Where, err = p.parseRel("and")
	return stmt, err
}

func (p *Parser) parseRel(op string) (sweet.Statement, error) {
	var (
		parse func(sweet.Binary) (sweet.Statement, error)
		err   error
	)
	parse = func(left sweet.Binary) (sweet.Statement, error) {
		if !p.is(Comma) {
			return nil, p.unexpected()
		}
		p.next()
		var err error
		if left.Right, err = p.parseExpr(); err != nil {
			return nil, err
		}
		switch {
		case p.is(Comma):
			b := sweet.Binary{
				Op:   strings.ToUpper(op),
				Left: left.Right,
			}
			left.Right, err = parse(b)
		case p.is(Rparen):
		default:
			return nil, p.unexpected()
		}
		return left, err
	}
	bin := sweet.Binary{
		Op: strings.ToUpper(op),
	}
	bin.Left, err = p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.is(Rparen) {
		return bin.Left, nil
	}
	return parse(bin)
}

func (p *Parser) parseExpr() (sweet.Statement, error) {
	if !p.is(Ident) {
		return nil, p.unexpected()
	}
	var (
		op    = p.curr.Literal
		parse parseFunc[sweet.Statement]
	)
	switch op {
	case "eq", "ne", "lt", "le", "gt", "ge", "like", "ilike":
		parse = p.parseBinary
	case "and", "or":
		parse = p.parseRel
	case "between":
		parse = p.parseBetween
	default:
		return nil, p.unexpected()
	}
	return withParens(p, parse)
}

func (p *Parser) parseBinary(op string) (sweet.Statement, error) {
	var (
		bin sweet.Binary
		err error
		ok  bool
	)
	if bin.Op, ok = opMapping[op]; !ok {
		bin.Op = strings.ToUpper(op)
	}
	bin.Left, err = p.parseIdentOrValue()
	if err != nil {
		return nil, err
	}
	if !p.is(Comma) {
		return nil, p.unexpected()
	}
	p.next()
	bin.Right, err = p.parseIdentOrValue()
	if err != nil {
		return nil, err
	}
	return bin, nil
}

func (p *Parser) parseBetween(_ string) (sweet.Statement, error) {
	var (
		stmt sweet.Between
		err  error
	)
	stmt.Ident, err = p.parseIdentOrValue()
	if err != nil {
		return nil, p.unexpected()
	}
	if !p.is(Comma) {
		return nil, p.unexpected()
	}
	p.next()
	stmt.Lower, err = p.parseIdentOrValue()
	if err != nil {
		return nil, err
	}
	if !p.is(Comma) {
		return nil, p.unexpected()
	}
	p.next()
	stmt.Upper, err = p.parseIdentOrValue()
	if err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *Parser) parseWhere(_ string) (sweet.Statement, error) {
	return p.parseRel("and")
}

func (p *Parser) parseHaving(_ string) (sweet.Statement, error) {
	return p.parseRel("and")
}

func (p *Parser) parseUpdate(_ string) (sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) parseInsert(_ string) (sweet.Statement, error) {
	var (
		ins sweet.InsertStatement
		err error
	)
	if !p.is(Ident) {
		return nil, p.unexpected()
	}
	ins.Table = p.curr.Literal
	p.next()
	for !p.done() {
		if !p.is(Ident) {
			return nil, p.unexpected()
		}
		ins.Columns = append(ins.Columns, p.curr.Literal)
		p.next()
		if err := p.ensureEOL(); err != nil {
			return nil, err
		}
		if ok := p.check("select", "values"); ok && p.is(Ident) {
			break
		}
	}
	if !p.is(Ident) {
		return nil, p.unexpected()
	}
	switch {
	case p.check("values"):
	case p.check("select"):
		ins.Values, err =  withParens(p, p.parseSelect)
	default:
		return nil, p.unexpected()
	}
	return ins, err
}

func (p *Parser) parseDelete(_ string) (sweet.Statement, error) {
	return nil, nil
}

func (p *Parser) unexpected() error {
	return fmt.Errorf("unexpected token %s at %d:%d", p.curr, p.curr.Line, p.curr.Column)
}

func (p *Parser) check(str ...string) bool {
	for _, s := range str {
		if p.curr.Literal == s {
			return true
		}
	}
	return false
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

type parseFunc[T sweet.Statement | []sweet.Statement] func(string) (T, error)

func withParens[T sweet.Statement | []sweet.Statement](p *Parser, parse parseFunc[T]) (ret T, err error) {
	ident := p.curr.Literal
	p.next()
	if !p.is(Lparen) {
		return ret, p.unexpected()
	}
	p.next()

	if ret, err = parse(ident); err != nil {
		return
	}
	if !p.is(Rparen) {
		return ret, p.unexpected()
	}
	p.next()
	return
}

var opMapping = map[string]string{
	"eq": "=",
	"ne": "<>",
	"lt": "<",
	"le": "<=",
	"gt": ">",
	"ge": ">=",
}

var joinMapping = map[string]string{
	"join":      "JOIN",
	"leftjoin":  "LEFT JOIN",
	"rightjoin": "RIGHT JOIN",
	"fulljoin":  "FULL JOIN",
}
