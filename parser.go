package sweet

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type Parser struct {
	*frame
	stack []*frame

	keywords map[string]func() (Statement, error)
	infix    map[symbol]infixFunc
	prefix   map[symbol]prefixFunc

	set KeywordSet
}

func NewParser(r io.Reader, keywords KeywordSet) (*Parser, error) {
	var p Parser
	frame, err := createFrame(r, keywords)
	if err != nil {
		return nil, err
	}
	p.frame = frame
	p.set = keywords
	p.keywords = map[string]func() (Statement, error){
		"SELECT":      p.parseSelect,
		"DELETE FROM": p.parseDelete,
		"UPDATE":      p.parseUpdate,
		"INSERT INTO": p.parseInsert,
		"WITH":        p.parseWith,
		"IF":          p.parseIf,
		"CASE":        p.parseCase,
		"WHILE":       p.parseWhile,
		"COMMIT":      p.parseCommit,
		"ROLLBACK":    p.parseRollback,
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
	p.registerPrefix("CASE", Keyword, p.parseCase)
	p.registerPrefix("SELECT", Keyword, p.parseSelect)

	return &p, nil
}

func (p *Parser) Parse() (Statement, error) {
	for p.is(Comment) {
		p.next()
	}
	if p.is(Macro) {
		if err := p.parseMacro(); err != nil {
			return nil, err
		}
	}
	stmt, err := p.parseStatement()
	if err != nil {
		return nil, err
	}
	if !p.is(EOL) {
		return nil, p.wantError("statement", ";")
	}
	p.next()
	return stmt, nil
}

func (p *Parser) parseMacro() error {
	var err error
	switch p.curr.Literal {
	case "INCLUDE":
		fmt.Println(p.curr, p.peek)
		err = p.parseIncludeMacro()
	case "DEFINE":
		err = p.parseDefineMacro()
	default:
		err = fmt.Errorf("macro %s unsupported", p.curr.Literal)
	}
	if err != nil {
		return err
	}
	return nil
}

func (p *Parser) parseIncludeMacro() error {
	p.next()

	file := filepath.Join(p.base, p.curr.Literal)
	p.next()

	if !p.is(EOL) {
		return p.wantError("include", ";")
	}
	p.next()

	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	frame, err := createFrame(r, p.set)
	if err != nil {
		return err
	}
	p.stack = append(p.stack, p.frame)
	p.frame = frame

	return nil
}

func (p *Parser) parseDefineMacro() error {
	return nil
}

func (p *Parser) parseStatement() (Statement, error) {
	if p.done() {
		return nil, io.EOF
	}
	if p.curr.Type != Keyword {
		return nil, p.wantError("statement", "keyword")
	}
	fn, ok := p.keywords[p.curr.Literal]
	if !ok {
		return nil, p.unexpected("statement")
	}
	return fn()
}

func (p *Parser) parseCommit() (Statement, error) {
	p.next()
	return Commit{}, nil
}

func (p *Parser) parseRollback() (Statement, error) {
	p.next()
	return Rollback{}, nil
}

func (p *Parser) parseWith() (Statement, error) {
	p.next()
	var (
		stmt WithStatement
		err  error
	)
	for !p.done() && !p.isKeyword("SELECT") {
		cte, err := p.parseSubquery()
		if err != nil {
			return nil, err
		}
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

func (p *Parser) parseSubquery() (Statement, error) {
	var (
		cte CteStatement
		err error
	)
	if !p.is(Ident) {
		return nil, p.unexpected("subquery")
	}
	cte.Ident = p.curr.Literal
	p.next()
	cte.Columns, err = p.parseColumnsList()
	if err != nil {
		return nil, err
	}
	if !p.isKeyword("AS") {
		return nil, p.unexpected("subquery")
	}
	p.next()
	if !p.is(Lparen) {
		return nil, p.unexpected("subquery")
	}
	p.next()
	cte.Statement, err = p.parseSelect()
	if err != nil {
		return nil, err
	}
	if !p.is(Rparen) {
		return nil, p.unexpected("subquery")
	}
	p.next()
	return cte, nil
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
		stmt.Columns, err = p.parseColumnsList()
		if err != nil {
			return nil, err
		}
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
				return nil, p.unexpected("values")
			}
			p.next()

			list, err := p.parseValues()
			if err != nil {
				return nil, err
			}
			all.Values = append(all.Values, list)

			switch {
			case p.is(Comma):
				p.next()
			case p.is(EOL):
			case p.is(Keyword):
			default:
				return nil, p.unexpected("insert(values)")
			}
		}
		stmt.Values = all
	default:
		return nil, p.unexpected("insert(values)")
	}
	if stmt.Return, err = p.parseReturning(); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *Parser) parseValues() (Statement, error) {
	var list List
	for !p.done() && !p.is(Rparen) {
		expr, err := p.parseExpression(powLowest, func() bool {
			return p.is(EOL) || p.is(Rparen)
		})
		if err = wrapError("values", err); err != nil {
			return nil, err
		}
		if err := p.ensureEnd("values", Comma, Rparen); err != nil {
			return nil, err
		}
		list.Values = append(list.Values, expr)
	}
	if !p.is(Rparen) {
		return nil, p.unexpected("values")
	}
	p.next()
	return list, nil
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
		stmt.Cdt, err = p.parseExpression(powLowest, func() bool {
			return p.isKeyword("WHEN")
		})
		if err = wrapError("case", err); err != nil {
			return nil, err
		}
	}
	for p.isKeyword("WHEN") {
		var when WhenStatement
		p.next()
		when.Cdt, err = p.parseExpression(powLowest, func() bool {
			return p.isKeyword("THEN")
		})
		if err = wrapError("when", err); err != nil {
			return nil, err
		}
		p.next()
		when.Body, err = p.parseExpression(powLowest, func() bool {
			return p.isKeyword("WHEN") || p.isKeyword("ELSE") || p.isKeyword("END")
		})
		if err = wrapError("then", err); err != nil {
			return nil, err
		}
		stmt.Body = append(stmt.Body, when)
	}
	if p.isKeyword("ELSE") {
		p.next()
		stmt.Else, err = p.parseExpression(powLowest, func() bool {
			return p.isKeyword("END")
		})
		if err = wrapError("else", err); err != nil {
			return nil, err
		}
	}
	if !p.isKeyword("END") {
		return nil, p.unexpected("case")
	}
	p.next()
	return p.parseAlias(stmt)
}

func (p *Parser) parseWhile() (Statement, error) {
	var (
		stmt WhileStatement
		err  error
	)
	stmt.Cdt, err = p.parseExpression(powLowest, func() bool {
		return p.isKeyword("DO")
	})
	if err = wrapError("while", err); err != nil {
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
	if stmt.Limit, err = p.parseLimit(); err != nil {
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
		stmt, err := p.parseExpression(powLowest, done)
		if err = wrapError("fields", err); err != nil {
			return nil, err
		}
		switch {
		case p.is(Comma):
			p.next()
			if p.isKeyword("FROM") {
				return nil, p.unexpected("fields")
			}
		case p.isKeyword("FROM"):
		default:
			return nil, p.unexpected("fields")
		}
		list = append(list, stmt)
	}
	if !p.isKeyword("FROM") {
		return nil, p.unexpected("fields")
	}
	return list, nil
}

func (p *Parser) parseFrom() ([]Statement, error) {
	if !p.isKeyword("FROM") {
		return nil, p.unexpected("from")
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
			j.Table, err = p.parseAlias(j.Table)
			if err != nil {
				return nil, err
			}
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
	return p.parseExpression(powLowest, func() bool {
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
		stmt, err := p.parseExpression(powLowest, func() bool {
			return p.is(Comma) || p.is(Rparen)
		})
		if err = wrapError("using", err); err != nil {
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
	return p.parseExpression(powLowest, func() bool {
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
	return p.parseExpression(powLowest, func() bool {
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

func (p *Parser) parseLimit() (Statement, error) {
	if !p.isKeyword("LIMIT") {
		return nil, nil
	}
	var (
		lim Limit
		err error
	)
	p.next()
	lim.Count, err = strconv.Atoi(p.curr.Literal)
	if err != nil {
		return nil, p.unexpected("limit")
	}
	p.next()
	if !p.is(Comma) && !p.isKeyword("OFFSET") {
		return lim, nil
	}
	p.next()
	lim.Offset, err = strconv.Atoi(p.curr.Literal)
	if err != nil {
		return nil, p.unexpected("offset")
	}
	p.next()
	return lim, nil
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
	return p.parseExpression(powLowest, func() bool {
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

func (p *Parser) parseExpression(power int, end func() bool) (Statement, error) {
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
		return nil, p.unexpected("infix")
	}
	pow := p.currBinding()
	p.next()
	right, err := p.parseExpression(pow, end)
	if err = wrapError("infix", err); err != nil {
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
	right, err := p.parseExpression(pow, end)
	if err = wrapError("expression", err); err != nil {
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
		arg, err := p.parseExpression(powLowest, done)
		if err = wrapError("call", err); err != nil {
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
		stmt, err = p.parseExpression(powLowest, nil)
		if err = wrapError("unary", err); err != nil {
			return nil, err
		}
		stmt = Unary{
			Right: stmt,
			Op:    "-",
		}
	case p.isKeyword("NOT"):
		stmt, err = p.parseExpression(powLowest, nil)
		if err = wrapError("when", err); err != nil {
			return nil, err
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
	stmt, err := p.parseExpression(powLowest, func() bool {
		return p.curr.Type == Rparen
	})
	if err = wrapError("group", err); err != nil {
		return nil, err
	}
	if !p.is(Rparen) {
		return nil, p.unexpected("group")
	}
	p.next()
	return stmt, nil
}

func (p *Parser) parseColumnsList() ([]string, error) {
	if !p.is(Lparen) {
		return nil, nil
	}
	p.next()

	var (
		list []string
		err  error
	)

	for !p.done() && !p.is(Rparen) {
		if !p.curr.isValue() {
			return nil, p.unexpected("columns")
		}
		list = append(list, p.curr.Literal)
		p.next()
		if err := p.ensureEnd("columns", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	if !p.is(Rparen) {
		return nil, p.unexpected("columns")
	}
	p.next()

	return list, err
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
		switch {
		case p.is(Lparen):
			p.next()
			stmt, err = p.parseSelect()
			if err != nil {
				return nil, err
			}
			if !p.is(Rparen) {
				return nil, p.unexpected("list")
			}
			p.next()
		case p.is(Ident):
			if p.is(Ident) && p.peekIs(Dot) {
				name.Prefix = p.curr.Literal
				p.next()
				p.next()
			}
			name.Ident = p.curr.Literal
			stmt = name
			p.next()
		default:
			return nil, p.unexpected("list")
		}
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

func (p *Parser) isKeyword(kw string) bool {
	return p.curr.Type == Keyword && p.curr.Literal == kw
}

func (p *Parser) currBinding() int {
	return bindings[p.curr.asSymbol()]
}

func (p *Parser) peekBinding() int {
	return bindings[p.peek.asSymbol()]
}

func (p *Parser) wantError(ctx, str string) error {
	return fmt.Errorf("%s: expected %q! got %s", ctx, str, p.curr.Literal)
}

func (p *Parser) unexpected(ctx string) error {
	return wrapError(ctx, unexpected(p.curr))
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
	if p.frame.done() {
		if n := len(p.stack); n > 0 {
			p.frame = p.stack[n-1]
			p.stack = p.stack[:n-1]
		}
	}
	return p.frame.done()
}

func unexpected(tok Token) error {
	return fmt.Errorf("unexpected token %s", tok)
}

func wrapError(ctx string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", ctx, err)
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

type frame struct {
	*Scanner
	base string
	curr Token
	peek Token
}

func createFrame(r io.Reader, keywords KeywordSet) (*frame, error) {
	scan, err := Scan(r, keywords)
	if err != nil {
		return nil, err
	}
	f := frame{
		Scanner: scan,
	}
	if n, ok := r.(interface{ Name() string }); ok {
		f.base = filepath.Dir(n.Name())
	}
	f.next()
	f.next()
	return &f, nil
}

func (f *frame) next() {
	f.curr = f.peek
	f.peek = f.Scan()
}

func (f *frame) done() bool {
	return f.is(EOF)
}

func (f *frame) is(kind rune) bool {
	return f.curr.Type == kind
}

func (f *frame) peekIs(kind rune) bool {
	return f.peek.Type == kind
}
