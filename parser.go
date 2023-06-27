package sweet

import (
	"fmt"
	"io"
)

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	keywords map[string]func() (Statement, error)
	infix    map[symbol]infixFunc
	prefix   map[symbol]prefixFunc
}

func NewParser(r io.Reader, keywords KeywordSet) (*Parser, error) {
	scan, err := Scan(r, keywords)
	if err != nil {
		return nil, err
	}
	var p Parser
	p.scan = scan
	p.keywords = map[string]func() (Statement, error){
		"SELECT":      p.parseSelect,
		"DELETE FROM": p.parseDelete,
		"UPDATE":      p.parseUpdate,
		"INSERT INTO": p.parseInsert,
		"WITH":        p.parseWith,
		"IF":          p.parseIf,
		"CASE":        p.parseCase,
		"WHILE":       p.parseWhile,
	}

	p.infix = make(map[symbol]infixFunc)
	p.registerInfix("", Plus, p.parseInfixExpr)
	p.registerInfix("", Minus, p.parseInfixExpr)
	p.registerInfix("", Slash, p.parseInfixExpr)
	p.registerInfix("", Star, p.parseInfixExpr)
	p.registerInfix("", Eq, p.parseInfixExpr)
	p.registerInfix("", Ne, p.parseInfixExpr)
	p.registerInfix("", Lt, p.parseInfixExpr)
	p.registerInfix("", Le, p.parseInfixExpr)
	p.registerInfix("", Gt, p.parseInfixExpr)
	p.registerInfix("", Ge, p.parseInfixExpr)
	p.registerInfix("", Lparen, p.parseCallExpr)
	p.registerInfix("AND", Keyword, p.parseKeywordExpr)
	p.registerInfix("OR", Keyword, p.parseKeywordExpr)
	p.registerInfix("LIKE", Keyword, p.parseKeywordExpr)
	p.registerInfix("ILIKE", Keyword, p.parseKeywordExpr)
	p.registerInfix("BETWEEN", Keyword, p.parseKeywordExpr)
	p.registerInfix("AS", Keyword, p.parseKeywordExpr)
	p.registerInfix("IN", Keyword, p.parseKeywordExpr)

	p.prefix = make(map[symbol]prefixFunc)
	p.registerPrefix("", Ident, p.parseIdent)
	p.registerPrefix("", Star, p.parseIdent)
	p.registerPrefix("", Literal, p.parseLiteral)
	p.registerPrefix("", Number, p.parseLiteral)
	p.registerPrefix("", Lparen, p.parseGroup)
	p.registerPrefix("", Minus, p.parseUnary)
	p.registerPrefix("", Keyword, p.parseUnary)
	p.registerPrefix("NOT", Keyword, p.parseUnary)
	p.registerPrefix("NULL", Keyword, p.parseUnary)
	p.registerPrefix("DEFAULT", Keyword, p.parseUnary)
	p.registerPrefix("CASE", Keyword, p.parseUnary)
	p.registerPrefix("SELECT", Keyword, p.parseSelect)

	p.next()
	p.next()

	return &p, nil
}

func (p *Parser) Parse() (Statement, error) {
	for p.curr.Type == Comment {
		p.next()
	}
	stmt, err := p.parseStatement()
	if err != nil {
		return nil, err
	}
	if p.curr.Type != EOL {
		return nil, fmt.Errorf("want \";\" after statement but got %s", p.curr)
	}
	p.next()
	return stmt, nil
}

func (p *Parser) parseStatement() (Statement, error) {
	if p.done() {
		return nil, io.EOF
	}
	if p.curr.Type != Keyword {
		return nil, fmt.Errorf("keyword expected, got %s", p.curr)
	}
	fn, ok := p.keywords[p.curr.Literal]
	if !ok {
		return nil, fmt.Errorf("unsupported/unrecognized keyword: %s", p.curr.Literal)
	}
	return fn()
}

func (p *Parser) parseWith() (Statement, error) {
	p.next()
	var (
		stmt WithStatement
		err  error
	)
	for !p.done() && !p.isKeyword("SELECT") {
		var cte CteStatement
		if !p.is(Ident) {
			return nil, p.unexpected("with")
		}
		cte.Ident = p.curr.Literal
		p.next()
		if p.is(Lparen) {
			p.next()
			for !p.done() && !p.is(Rparen) {
				if !p.curr.isValue() {
					return nil, p.unexpected("with")
				}
				cte.Columns = append(cte.Columns, p.curr.Literal)
				p.next()
				if err := p.ensureEnd("with", Comma, Rparen); err != nil {
					return nil, err
				}
			}
			if !p.is(Rparen) {
				return nil, p.unexpected("with")
			}
			p.next()
		}
		if !p.isKeyword("AS") {
			return nil, p.unexpected("with")
		}
		p.next()
		if !p.is(Lparen) {
			return nil, p.unexpected("with")
		}
		p.next()
		cte.Statement, err = p.parseSelect()
		if err != nil {
			return nil, err
		}
		if !p.is(Rparen) {
			return nil, p.unexpected("with")
		}
		p.next()
		stmt.Queries = append(stmt.Queries, cte)
	}
	if !p.is(Keyword) {
		return nil, p.unexpected("with")
	}
	switch p.curr.Literal {
	case "SELECT":
		stmt.Statement, err = p.parseSelect()
	case "DELETE FROM":
		stmt.Statement, err = p.parseDelete()
	case "INSERT INTO":
		stmt.Statement, err = p.parseInsert()
	case "UPDATE":
		stmt.Statement, err = p.parseUpdate()
	default:
		return nil, p.unexpected("with")
	}
	return stmt, err
}

func (p *Parser) parseDelete() (Statement, error) {
	p.next()
	var (
		stmt DeleteStatement
		err  error
	)
	if !p.is(Ident) {
		return nil, p.unexpected("delete")
	}
	stmt.Table = p.curr.Literal
	p.next()
	if stmt.Where, err = p.parseWhere(); err != nil {
		return nil, err
	}
	if stmt.Return, err = p.parseReturning(); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *Parser) parseInsert() (Statement, error) {
	p.next()
	var (
		stmt InsertStatement
		err  error
	)
	if !p.is(Ident) {
		return nil, p.unexpected("insert")
	}
	stmt.Table = p.curr.Literal
	p.next()
	switch {
	case p.is(Lparen):
		p.next()
		for !p.done() && !p.is(Rparen) {
			if !p.curr.isValue() {
				return nil, p.unexpected("insert(columns)")
			}
			stmt.Columns = append(stmt.Columns, p.curr.Literal)
			p.next()
			if err := p.ensureEnd("insert", Comma, Rparen); err != nil {
				return nil, err
			}
		}
		if !p.is(Rparen) {
			return nil, p.unexpected("insert")
		}
		p.next()
	case p.isKeyword("VALUES"):
	case p.isKeyword("SELECT"):
	default:
		return nil, p.unexpected("insert")
	}

	switch {
	case p.isKeyword("SELECT"):
		stmt.Values, err = p.parseSelect()
		return stmt, err
	case p.isKeyword("VALUES"):
		p.next()
		var all List
		for !p.done() && !p.isKeyword("RETURNING") && !p.is(EOL) {
			if !p.is(Lparen) {
				return nil, p.unexpected("insert(values)")
			}
			p.next()
			var list List
			for !p.done() && !p.is(Rparen) {
				expr, err := p.parseExpression("insert(values)", powLowest, func() bool {
					return p.is(EOL) || p.is(Rparen)
				})
				if err != nil {
					return nil, err
				}
				if err := p.ensureEnd("insert(values)", Comma, Rparen); err != nil {
					return nil, err
				}
				list.Values = append(list.Values, expr)
			}
			if !p.is(Rparen) {
				return nil, p.unexpected("insert(values)")
			}
			p.next()
			switch {
			case p.is(Comma):
				p.next()
			case p.is(EOL):
			case p.is(Keyword):
			default:
				return nil, p.unexpected("insert(values)")
			}
			all.Values = append(all.Values, list)
		}
		stmt.Values = all.AsStatement()
	default:
		return nil, p.unexpected("insert(values)")
	}
	if stmt.Return, err = p.parseReturning(); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *Parser) parseUpdate() (Statement, error) {
	p.next()
	var stmt UpdateStatement
	return stmt, nil
}

func (p *Parser) parseIf() (Statement, error) {
	var stmt IfStatement
	return stmt, nil
}

func (p *Parser) parseCase() (Statement, error) {
	p.next()
	var (
		stmt CaseStatement
		err  error
	)
	if !p.isKeyword("WHEN") {
		stmt.Cdt, err = p.parseExpression("case", powLowest, func() bool {
			return p.isKeyword("WHEN")
		})
		if err != nil {
			return nil, err
		}
	}
	for p.isKeyword("WHEN") {
		var when WhenStatement
		p.next()
		when.Cdt, err = p.parseExpression("when", powLowest, func() bool {
			return p.isKeyword("THEN")
		})
		if err != nil {
			return nil, err
		}
		p.next()
		when.Body, err = p.parseExpression("then", powLowest, func() bool {
			return p.isKeyword("WHEN") || p.isKeyword("ELSE") || p.isKeyword("END")
		})
		stmt.Body = append(stmt.Body, when)
	}
	if p.isKeyword("ELSE") {
		p.next()
		stmt.Else, err = p.parseExpression("else", powLowest, func() bool {
			return p.isKeyword("END")
		})
		if err != nil {
			return nil, err
		}
	}
	if !p.isKeyword("END") {
		return nil, p.unexpected("case")
	}
	p.next()
	return stmt, nil
}

func (p *Parser) parseWhile() (Statement, error) {
	var (
		stmt WhileStatement
		err  error
	)
	stmt.Cdt, err = p.parseExpression("while", powLowest, func() bool {
		return p.isKeyword("DO")
	})
	if err != nil {
		return nil, err
	}
	p.next()
	return stmt, nil
}

func (p *Parser) parseSelect() (Statement, error) {
	p.next()
	var (
		stmt SelectStatement
		err  error
	)
	if stmt.Columns, err = p.parseColumns(); err != nil {
		return nil, err
	}
	if stmt.Tables, err = p.parseFrom(); err != nil {
		return nil, err
	}
	if stmt.Where, err = p.parseWhere(); err != nil {
		return nil, err
	}
	if stmt.Groups, err = p.parseGroupBy(); err != nil {
		return nil, err
	}
	if stmt.Having, err = p.parseHaving(); err != nil {
		return nil, err
	}
	if stmt.Orders, err = p.parseOrderBy(); err != nil {
		return nil, err
	}
	if stmt.Limit, stmt.Offset, err = p.parseLimit(); err != nil {
		return nil, err
	}
	allDistinct := func() (bool, bool) {
		p.next()
		var (
			all      = p.isKeyword("ALL")
			distinct = p.isKeyword("DISTINCT")
		)
		if all || distinct {
			p.next()
		}
		return all, distinct
	}
	switch {
	case p.isKeyword("UNION"):
		u := UnionStatement{
			Left: stmt,
		}
		u.All, u.Distinct = allDistinct()
		u.Right, err = p.parseSelect()
		return u, err
	case p.isKeyword("INTERSECT"):
		i := IntersectStatement{
			Left: stmt,
		}
		i.All, i.Distinct = allDistinct()
		i.Right, err = p.parseSelect()
		return i, err
	case p.isKeyword("EXCEPT"):
		e := ExceptStatement{
			Left: stmt,
		}
		e.All, e.Distinct = allDistinct()
		e.Right, err = p.parseSelect()
		return e, err
	default:
		return stmt, err
	}
}

func (p *Parser) parseColumns() ([]Statement, error) {
	var (
		list []Statement
		done = func() bool {
			return p.is(Comma) || p.isKeyword("FROM")
		}
	)
	for !p.done() && !p.isKeyword("FROM") {
		stmt, err := p.parseExpression("list", powLowest, done)
		if err != nil {
			return nil, err
		}
		switch {
		case p.is(Comma):
			p.next()
			if p.isKeyword("FROM") {
				return nil, p.unexpected("list")
			}
		case p.isKeyword("FROM"):
		default:
			return nil, p.unexpected("list")
		}
		list = append(list, stmt)
	}
	if !p.isKeyword("FROM") {
		return nil, p.unexpected("list")
	}
	return list, nil
}

func (p *Parser) parseFrom() ([]Statement, error) {
	if !p.isKeyword("FROM") {
		return nil, p.unexpected("tables")
	}
	p.next()

	list, err := p.parseStatementList("FROM", p.parseAlias)
	if err != nil {
		return nil, err
	}
	if p.is(EOL) {
		return list, nil
	}
	for !p.done() && p.curr.isJoin() {
		j := Join{
			Type: p.curr.Literal,
		}
		p.next()
		switch {
		case p.is(Ident):
			j.Table, err = p.parseIdent()
		case p.is(Lparen):
			p.next()
			j.Table, err = p.parseSelect()
			if err != nil {
				break
			}
			if !p.is(Rparen) {
				err = p.unexpected("join")
				break
			}
			p.next()
		default:
			return nil, p.unexpected("join")
		}
		if err != nil {
			return nil, err
		}
		switch {
		case p.isKeyword("ON"):
			j.Where, err = p.parseJoinOn()
		case p.isKeyword("USING"):
			j.Where, err = p.parseJoinUsing()
		default:
			return nil, p.unexpected("join")
		}
		if err != nil {
			return nil, err
		}
		list = append(list, j)
	}
	return list, nil
}

func (p *Parser) parseJoinOn() (Statement, error) {
	p.next()
	p.unregisterInfix("AS", Keyword)
	defer p.registerInfix("AS", Keyword, p.parseKeywordExpr)
	return p.parseExpression("on", powLowest, func() bool {
		if p.is(EOL) {
			return true
		}
		if !p.is(Keyword) {
			return false
		}
		switch p.curr.Literal {
		default:
			return false
		case "WHERE", "GROUP BY", "HAVING", "ORDER BY", "LIMIT", "UNION", "INTERSECT", "EXCEPT":
			return true
		}
	})
}

func (p *Parser) parseJoinUsing() (Statement, error) {
	p.next()
	if !p.is(Lparen) {
		return nil, p.unexpected("using")
	}
	p.next()
	p.unregisterInfix("AS", Keyword)
	defer p.registerInfix("AS", Keyword, p.parseKeywordExpr)

	var list List
	for !p.done() && !p.is(Rparen) {
		stmt, err := p.parseExpression("using", powLowest, func() bool {
			return p.is(Comma) || p.is(Rparen)
		})
		if err != nil {
			return nil, err
		}
		list.Values = append(list.Values, stmt)
		if err := p.ensureEnd("using", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	if !p.is(Rparen) {
		return nil, p.unexpected("using")
	}
	p.next()
	return list, nil
}

func (p *Parser) parseWhere() (Statement, error) {
	if !p.isKeyword("WHERE") {
		return nil, nil
	}
	p.next()
	p.unregisterInfix("AS", Keyword)
	defer p.registerInfix("AS", Keyword, p.parseKeywordExpr)
	return p.parseExpression("where", powLowest, func() bool {
		if p.is(EOL) {
			return true
		}
		if !p.is(Keyword) {
			return false
		}
		switch p.curr.Literal {
		default:
			return false
		case "GROUP BY", "HAVING", "ORDER BY", "LIMIT", "UNION", "INTERSECT", "EXCEPT":
			return true
		}
	})
}

func (p *Parser) parseGroupBy() ([]Statement, error) {
	if !p.isKeyword("GROUP BY") {
		return nil, nil
	}
	p.next()
	return p.parseStatementList("group by", nil)
}

func (p *Parser) parseHaving() (Statement, error) {
	if !p.isKeyword("HAVING") {
		return nil, nil
	}
	p.next()
	p.unregisterInfix("AS", Keyword)
	defer p.registerInfix("AS", Keyword, p.parseKeywordExpr)
	return p.parseExpression("having", powLowest, func() bool {
		if p.is(EOL) {
			return true
		}
		if !p.is(Keyword) {
			return false
		}
		switch p.curr.Literal {
		default:
			return false
		case "ORDER BY", "LIMIT", "UNION", "INTERSECT", "EXCEPT":
			return true
		}
	})
}

func (p *Parser) parseOrderBy() ([]Statement, error) {
	if !p.isKeyword("ORDER BY") {
		return nil, nil
	}
	p.next()
	do := func(stmt Statement) (Statement, error) {
		order := Order{
			Statement: stmt,
			Orient:    "ASC",
			Nulls:     "FIRST",
		}
		if p.isKeyword("ASC") || p.isKeyword("DESC") {
			order.Orient = p.curr.Literal
			p.next()
		}
		if p.isKeyword("NULLS") {
			p.next()
			if !p.isKeyword("FIRST") && !p.isKeyword("LAST") {
				return nil, p.unexpected("order by")
			}
			order.Nulls = p.curr.Literal
			p.next()
		}
		return order, nil
	}
	return p.parseStatementList("order by", do)
}

func (p *Parser) parseLimit() (string, string, error) {
	if !p.isKeyword("LIMIT") {
		return "", "", nil
	}
	p.next()
	if !p.is(Number) {
		return "", "", p.unexpected("limit")
	}
	limit := p.curr.Literal
	p.next()
	if !p.is(Comma) && !p.isKeyword("OFFSET") {
		return limit, "", nil
	}
	p.next()
	if !p.is(Number) {
		return "", "", p.unexpected("offset")
	}
	offset := p.curr.Literal
	p.next()
	return limit, offset, nil
}

func (p *Parser) parseReturning() (Statement, error) {
	if !p.isKeyword("RETURNING") {
		return nil, nil
	}
	p.next()
	if p.is(Star) {
		defer p.next()
		return Value{
			Literal: "*",
		}, nil
	}
	return p.parseExpression("returning", powLowest, func() bool {
		return p.is(EOL)
	})
}

func (p *Parser) registerPrefix(literal string, kind rune, fn prefixFunc) {
	p.prefix[symbolFor(kind, literal)] = fn
}

func (p *Parser) registerInfix(literal string, kind rune, fn infixFunc) {
	p.infix[symbolFor(kind, literal)] = fn
}

func (p *Parser) unregisterInfix(literal string, kind rune) {
	delete(p.infix, symbolFor(kind, literal))
}

func (p *Parser) getPrefixExpr() (Statement, error) {
	fn, ok := p.prefix[p.curr.asSymbol()]
	if !ok {
		return nil, p.unexpected("prefix")
	}
	return fn()
}

func (p *Parser) getInfixExpr(left Statement, end func() bool) (Statement, error) {
	fn, ok := p.infix[p.curr.asSymbol()]
	if !ok {
		return nil, p.unexpected("infix")
	}
	return fn(left, end)
}

func (p *Parser) parseExpression(ctx string, power int, end func() bool) (Statement, error) {
	left, err := p.getPrefixExpr()
	if err != nil {
		return nil, err
	}
	for !end() && power < p.currBinding() {
		left, err = p.getInfixExpr(left, end)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *Parser) parseInfixExpr(left Statement, end func() bool) (Statement, error) {
	stmt := Binary{
		Left: left,
	}
	switch {
	case p.is(Plus):
		stmt.Op = "+"
	case p.is(Minus):
		stmt.Op = "-"
	case p.is(Slash):
		stmt.Op = "/"
	case p.is(Star):
		stmt.Op = "*"
	case p.is(Eq):
		stmt.Op = "="
	case p.is(Ne):
		stmt.Op = "<>"
	case p.is(Lt):
		stmt.Op = "<"
	case p.is(Le):
		stmt.Op = "<="
	case p.is(Gt):
		stmt.Op = ">"
	case p.is(Ge):
		stmt.Op = ">="
	default:
		return nil, p.unexpected("infix expression")
	}
	pow := p.currBinding()
	p.next()
	right, err := p.parseExpression("infix", pow, end)
	if err != nil {
		return nil, err
	}
	stmt.Right = right
	return stmt, nil
}

func (p *Parser) parseKeywordExpr(left Statement, end func() bool) (Statement, error) {
	stmt := Binary{
		Left: left,
		Op:   p.curr.Literal,
	}
	pow := p.currBinding()
	p.next()
	right, err := p.parseExpression("expression", pow, end)
	if err != nil {
		return nil, err
	}
	stmt.Right = right
	return stmt, nil
}

func (p *Parser) parseCallExpr(left Statement, _ func() bool) (Statement, error) {
	p.next()
	stmt := Call{
		Ident: left,
	}
	done := func() bool {
		return p.is(Comma) || p.is(Rparen)
	}
	for !p.done() && !p.is(Rparen) {
		arg, err := p.parseExpression("call", powLowest, done)
		if err != nil {
			return nil, err
		}
		switch p.curr.Type {
		case Comma:
			p.next()
			if p.is(Rparen) {
				return nil, p.unexpected("call")
			}
		case Rparen:
		default:
			return nil, p.unexpected("call")
		}
		stmt.Args = append(stmt.Args, arg)
	}
	if !p.is(Rparen) {
		return nil, p.unexpected("call")
	}
	p.next()
	return p.parseAlias(stmt)
}

func (p *Parser) parseUnary() (Statement, error) {
	var (
		stmt Statement
		err  error
	)
	switch {
	case p.is(Minus):
		stmt, err = p.parseExpression("unary", powLowest, nil)
		if err != nil {
			break
		}
		stmt = Unary{
			Right: stmt,
			Op:    "-",
		}
	case p.isKeyword("NOT"):
		stmt, err = p.parseExpression("unary", powLowest, nil)
		if err != nil {
			break
		}
		stmt = Unary{
			Right: stmt,
			Op:    "NOT",
		}
	case p.isKeyword("CASE"):
		stmt, err = p.parseCase()
	case p.isKeyword("NULL") || p.isKeyword("DEFAULT"):
		stmt = Value{
			Literal: p.curr.Literal,
		}
		p.next()
	default:
		err = p.unexpected("unary")
	}
	return stmt, nil
}

func (p *Parser) parseIdent() (Statement, error) {
	var name Name
	if p.peekIs(Dot) {
		name.Prefix = p.curr.Literal
		p.next()
		p.next()
	}
	name.Ident = p.curr.Literal
	if p.is(Star) {
		name.Ident = "*"
	}
	p.next()
	return p.parseAlias(name)
}

func (p *Parser) parseLiteral() (Statement, error) {
	stmt := Value{
		Literal: p.curr.Literal,
	}
	p.next()
	return stmt, nil
}

func (p *Parser) parseGroup() (Statement, error) {
	p.next()
	if p.isKeyword("SELECT") {
		stmt, err := p.parseSelect()
		if err != nil {
			return nil, err
		}
		if !p.is(Rparen) {
			return nil, p.unexpected("group")
		}
		p.next()
		return p.parseAlias(stmt)
	}
	stmt, err := p.parseExpression("group", powLowest, func() bool {
		return p.curr.Type == Rparen
	})
	if err != nil {
		return nil, err
	}
	if !p.is(Rparen) {
		return nil, p.unexpected("group")
	}
	p.next()
	return stmt, nil
}

func (p *Parser) parseStatementList(ctx string, fn func(Statement) (Statement, error)) ([]Statement, error) {
	var (
		list []Statement
		err  error
	)
	for !p.done() && !p.is(Keyword) && !p.is(EOL) && !p.is(Rparen) {
		var (
			name Name
			stmt Statement
		)
		if !p.is(Ident) {
			return nil, p.unexpected(ctx)
		}
		if p.is(Ident) && p.peekIs(Dot) {
			name.Prefix = p.curr.Literal
			p.next()
			p.next()
		}
		name.Ident = p.curr.Literal
		stmt = name
		p.next()
		if fn != nil {
			if stmt, err = fn(stmt); err != nil {
				return nil, err
			}
		}
		list = append(list, stmt)
		switch {
		case p.is(Comma):
			p.next()
			if p.is(Keyword) || p.is(EOL) {
				return nil, p.unexpected(ctx)
			}
		case p.is(Keyword):
		case p.is(EOL):
		case p.is(Rparen):
		default:
			return nil, p.unexpected(ctx)
		}
	}
	return list, nil
}

func (p *Parser) parseAlias(stmt Statement) (Statement, error) {
	mandatory := p.isKeyword("AS")
	if mandatory {
		p.next()
	}
	switch p.curr.Type {
	case Ident, Literal, Number:
		stmt = Alias{
			Statement: stmt,
			Alias:     p.curr.Literal,
		}
		p.next()
	default:
		if mandatory {
			return nil, p.unexpected("alias")
		}
	}
	return stmt, nil
}

func (p *Parser) is(r rune) bool {
	return p.curr.Type == r
}

func (p *Parser) peekIs(r rune) bool {
	return p.peek.Type == r
}

func (p *Parser) isKeyword(kw string) bool {
	return p.curr.Type == Keyword && p.curr.Literal == kw
}

func (p *Parser) currBinding() int {
	return bindings[p.curr.asSymbol()]
}

func (p *Parser) peekBinding() int {
	return bindings[p.peek.asSymbol()]
}

func (p *Parser) unexpected(ctx string) error {
	return fmt.Errorf("%s: unexpected token %s", ctx, p.curr)
}

func (p *Parser) ensureEnd(ctx string, sep, end rune) error {
	switch {
	case p.is(sep):
		p.next()
		if p.is(end) {
			return p.unexpected(ctx)
		}
	case p.is(end):
	default:
		return p.unexpected(ctx)
	}
	return nil
}

func (p *Parser) done() bool {
	return p.curr.Type == EOF
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

type prefixFunc func() (Statement, error)

type infixFunc func(Statement, func() bool) (Statement, error)

const (
	powLowest int = iota
	powRel
	powCmp
	powKw
	powNot
	powConcat
	powAdd
	powMul
	powUnary
	powCall
)

var bindings = map[symbol]int{
	symbolFor(Keyword, "AND"):     powRel,
	symbolFor(Keyword, "OR"):      powRel,
	symbolFor(Keyword, "NOT"):     powNot,
	symbolFor(Keyword, "LIKE"):    powCmp,
	symbolFor(Keyword, "ILIKE"):   powCmp,
	symbolFor(Keyword, "BETWEEN"): powCmp,
	symbolFor(Keyword, "IN"):      powCmp,
	symbolFor(Keyword, "AS"):      powKw,
	symbolFor(Lt, ""):             powCmp,
	symbolFor(Le, ""):             powCmp,
	symbolFor(Gt, ""):             powCmp,
	symbolFor(Ge, ""):             powCmp,
	symbolFor(Eq, ""):             powCmp,
	symbolFor(Ne, ""):             powCmp,
	symbolFor(Plus, ""):           powAdd,
	symbolFor(Minus, ""):          powAdd,
	symbolFor(Star, ""):           powMul,
	symbolFor(Slash, ""):          powMul,
	symbolFor(Lparen, ""):         powCall,
	symbolFor(Concat, ""):         powConcat,
}
