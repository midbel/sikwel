package lang

import (
	"fmt"
)

type prefixFunc func() (Statement, error)

type infixFunc func(Statement) (Statement, error)

func (p *Parser) parseExpression(power int) (Statement, error) {
	fn, err := p.getPrefixExpr()
	if err != nil {
		return nil, err
	}
	left, err := fn()
	if err != nil {
		return nil, err
	}
	for !p.QueryEnds() && !p.Is(Comma) && !p.Done() && power < p.currBinding() {
		fn, err := p.getInfixExpr()
		if err != nil {
			return nil, err
		}
		if left, err = fn(left); err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *Parser) getPrefixExpr() (prefixFunc, error) {
	fn, ok := p.prefix[p.curr.asSymbol()]
	if !ok {
		return nil, p.Unexpected("prefix")
	}
	return fn, nil
}

func (p *Parser) getInfixExpr() (infixFunc, error) {
	fn, ok := p.infix[p.curr.asSymbol()]
	if !ok {
		return nil, p.Unexpected("infix")
	}
	return fn, nil
}

func (p *Parser) parseInfixExpr(left Statement) (Statement, error) {
	stmt := Binary{
		Left: left,
	}
	var (
		pow = p.currBinding()
		err error
	)
	stmt.Op = operandMapping.Get(p.curr.Type)
	if stmt.Op == "" {
		return nil, p.Unexpected("operand")
	}
	p.Next()

	stmt.Right, err = p.parseExpression(pow)
	return stmt, wrapError("infix", err)
}

func (p *Parser) parseCollateExpr(left Statement) (Statement, error) {
	stmt := Collate{
		Statement: left,
	}
	p.Next()
	if !p.Is(Literal) {
		return nil, p.Unexpected("collate")
	}
	stmt.Collation = p.GetCurrLiteral()
	p.Next()
	return stmt, nil
}

func (p *Parser) parseKeywordExpr(left Statement) (Statement, error) {
	not := p.GetCurrLiteral() == "NOT" && p.Is(Keyword)
	reverse := func(stmt Statement) Statement { return stmt }
	if not {
		p.Next()
		reverse = func(stmt Statement) Statement {
			return Not{
				Statement: stmt,
			}
		}
	}
	switch p.GetCurrLiteral() {
	case "AND", "OR":
		stmt := Binary{
			Left: left,
			Op:   p.GetCurrLiteral(),
		}
		var (
			pow = p.currBinding()
			err error
		)
		p.Next()
		stmt.Right, err = p.parseExpression(pow)
		return stmt, wrapError("infix", err)
	case "LIKE", "ILIKE", "SIMILAR":
		stmt := Binary{
			Left: left,
			Op:   p.GetCurrLiteral(),
		}
		var (
			pow = p.currBinding()
			err error
		)
		p.Next()
		stmt.Right, err = p.parseExpression(pow)
		return reverse(stmt), wrapError("infix", err)
	case "ANY", "SOME":
	case "ALL":
	case "EXISTS":
		p.Next()
		if !p.Is(Lparen) {
			return nil, p.Unexpected("expression")
		}
		p.Next()
		var (
			expr Exists
			err  error
		)
		expr.Statement, err = p.ParseStatement()
		if err != nil {
			return nil, err
		}
		if !p.Is(Rparen) {
			return nil, p.Unexpected("expression")
		}
		p.Next()
		return reverse(expr), nil
	case "BETWEEN":
		p.Next()
		expr := Between{
			Ident: left,
		}
		left, err := p.StartExpression()
		if err != nil {
			return nil, err
		}
		if !p.IsKeyword("AND") {
			return nil, p.Unexpected("expression")
		}
		p.Next()
		right, err := p.StartExpression()
		if err != nil {
			return nil, err
		}
		expr.Lower = left
		expr.Upper = right
		return reverse(expr), nil
	case "IN":
		var stmt Statement
		return reverse(stmt), nil
	case "IS":
		p.Next()
		not := p.GetCurrLiteral() == "NOT" && p.Is(Keyword)
		if not {
			p.Next()
		}
		expr := Is{
			Ident: left,
		}
		val, err := p.ParseConstant()
		if err != nil {
			return nil, err
		}
		expr.Value = val
		return reverse(expr), nil
	case "ISNULL":
	case "NOTNULL":
	default:
		return nil, p.Unexpected("expression")
	}
	return nil, fmt.Errorf("not yet implemented")
}

func (p *Parser) parseCallExpr(left Statement) (Statement, error) {
	p.Next()
	stmt := Call{
		Ident:    left,
		Distinct: p.IsKeyword("DISTINCT"),
	}
	if stmt.Distinct {
		p.Next()
	}
	for !p.Done() && !p.Is(Rparen) {
		arg, err := p.StartExpression()
		if err = wrapError("call", err); err != nil {
			return nil, err
		}
		if err := p.EnsureEnd("call", Comma, Rparen); err != nil {
			return nil, err
		}
		stmt.Args = append(stmt.Args, arg)
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("call")
	}
	p.Next()
	if p.IsKeyword("FILTER") {
		p.Next()
		if !p.Is(Lparen) {
			return nil, p.Unexpected("call/filter")
		}
		p.Next()
		if !p.IsKeyword("WHERE") {
			return nil, p.Unexpected("call/filter")
		}
		p.Next()
		filter, err := p.StartExpression()
		if err != nil {
			return nil, err
		}
		stmt.Filter = filter
		if !p.Is(Rparen) {
			return nil, p.Unexpected("call/filter")
		}
		p.Next()
	}
	over, err := p.parseOver()
	if err != nil {
		return nil, err
	}
	stmt.Over = over
	return p.ParseAlias(stmt)
}

func (p *Parser) parseOver() (Statement, error) {
	if !p.IsKeyword("OVER") {
		return nil, nil
	}
	p.UnregisterInfix("AS", Keyword)
	defer p.RegisterInfix("AS", Keyword, p.parseKeywordExpr)
	p.Next()
	if !p.Is(Lparen) {
		return p.ParseIdentifier()
	}
	return p.ParseWindow()
}

func (p *Parser) parseUnary() (Statement, error) {
	var (
		stmt Statement
		err  error
	)
	switch {
	case p.Is(Minus):
		p.Next()
		stmt, err = p.StartExpression()
		if err = wrapError("reverse", err); err != nil {
			return nil, err
		}
		stmt = Unary{
			Right: stmt,
			Op:    "-",
		}
	case p.IsKeyword("NOT"):
		p.Next()
		stmt, err = p.StartExpression()
		if err = wrapError("not", err); err != nil {
			return nil, err
		}
		stmt = Unary{
			Right: stmt,
			Op:    "NOT",
		}
	case p.IsKeyword("CASE"):
		stmt, err = p.ParseCase()
	case p.IsKeyword("NULL") || p.IsKeyword("DEFAULT"):
		stmt = Value{
			Literal: p.curr.Literal,
		}
		p.Next()
	case p.IsKeyword("EXISTS"):
		p.Next()
		if !p.Is(Lparen) {
			return nil, p.Unexpected("exists")
		}
		stmt, err = p.StartExpression()
		if err == nil {
			stmt = Exists{
				Statement: stmt,
			}
		}
	default:
		err = p.Unexpected("unary")
	}
	return stmt, nil
}

func (p *Parser) parseGroupExpr() (Statement, error) {
	p.Next()
	if p.IsKeyword("SELECT") {
		stmt, err := p.ParseStatement()
		if err != nil {
			return nil, err
		}
		if !p.Is(Rparen) {
			return nil, p.Unexpected("group")
		}
		p.Next()
		return p.ParseAlias(stmt)
	}
	stmt, err := p.StartExpression()
	if err = wrapError("group", err); err != nil {
		return nil, err
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("group")
	}
	p.Next()
	return stmt, nil
}

type OpSet map[rune]string

var operandMapping = OpSet{
	Plus:   "+",
	Minus:  "-",
	Slash:  "/",
	Star:   "*",
	Eq:     "=",
	Ne:     "<>",
	Gt:     ">",
	Ge:     ">=",
	Lt:     "<",
	Le:     "<=",
	Concat: "||",
}

func (o OpSet) Get(r rune) string {
	return o[r]
}

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
	symbolFor(Keyword, "IS"):      powKw,
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
