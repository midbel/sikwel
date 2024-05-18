package lang

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

func (p *Parser) parseRelational(ident Statement) (Statement, error) {
	stmt := Binary{
		Left: ident,
		Op:   p.GetCurrLiteral(),
	}
	var (
		pow = p.currBinding()
		err error
	)
	p.Next()
	stmt.Right, err = p.parseExpression(pow)
	return stmt, err
}

func (p *Parser) parseLike(ident Statement) (Statement, error) {
	stmt := Binary{
		Left: ident,
		Op:   p.GetCurrLiteral(),
	}
	var (
		pow = p.currBinding()
		err error
	)
	p.Next()
	stmt.Right, err = p.parseExpression(pow)
	return stmt, err
}

func (p *Parser) parseIs(ident Statement) (Statement, error) {
	p.Next()
	not := p.GetCurrLiteral() == "NOT" && p.Is(Keyword)
	if not {
		p.Next()
	}
	stmt := Is{
		Ident: ident,
	}
	val, err := p.ParseConstant()
	if err != nil {
		return nil, err
	}
	stmt.Value = val
	if not {
		return Not{
			Statement: stmt,
		}, nil
	}
	return stmt, nil
}

func (p *Parser) parseIsNull(ident Statement) (Statement, error) {
	p.Next()
	val := Value{
		Literal: "NULL",
	}
	stmt := Is{
		Ident: ident,
		Value: val,
	}
	return stmt, nil
}

func (p *Parser) parseNotNull(ident Statement) (Statement, error) {
	p.Next()
	val := Value{
		Literal: "NULL",
	}
	stmt := Is{
		Ident: ident,
		Value: val,
	}
	not := Not{
		Statement: stmt,
	}
	return not, nil
}

func (p *Parser) parseExists(ident Statement) (Statement, error) {
	p.Next()
	if !p.Is(Lparen) {
		return nil, p.Unexpected("expression")
	}
	p.Next()
	var (
		stmt Exists
		err  error
	)
	stmt.Statement, err = p.ParseStatement()
	if err != nil {
		return nil, err
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("expression")
	}
	p.Next()
	return stmt, nil
}

func (p *Parser) parseBetween(ident Statement) (Statement, error) {
	p.Next()
	stmt := Between{
		Ident: ident,
	}
	left, err := p.parseExpression(powRel)
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("AND") {
		return nil, p.Unexpected("expression")
	}
	p.Next()
	right, err := p.parseExpression(powRel)
	if err != nil {
		return nil, err
	}
	stmt.Lower = left
	stmt.Upper = right
	return stmt, nil
}

func (p *Parser) parseIn(ident Statement) (Statement, error) {
	p.Next()
	in := In{
		Ident: ident,
	}
	var err error
	if p.Is(Lparen) && p.peekIs(Keyword) && p.GetPeekLiteral() == "SELECT" {
		in.Value, err = p.parseExpression(powLowest)
	} else if p.Is(Lparen) {
		p.Next()
		var (
			list List
			val  Statement
		)
		for !p.Done() && !p.Is(Rparen) {
			val, err = p.parseExpression(powLowest)
			if err != nil {
				return nil, err
			}
			switch {
			case p.Is(Comma):
				p.Next()
				if p.Is(Rparen) {
					return nil, p.Unexpected("in")
				}
			case p.Is(Rparen):
			default:
				return nil, p.Unexpected("in")
			}
			list.Values = append(list.Values, val)
		}
		if !p.Is(Rparen) {
			return nil, p.Unexpected("in")
		}
		in.Value = list
		p.Next()
	} else {
		in.Value, err = p.ParseIdentifier()
	}
	return in, err
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
	reverse := func(stmt Statement) Statement { return stmt }
	if p.GetCurrLiteral() == "NOT" && p.Is(Keyword) {
		p.Next()
		reverse = func(stmt Statement) Statement {
			if stmt == nil {
				return stmt
			}
			return Not{
				Statement: stmt,
			}
		}
	}
	var (
		stmt Statement
		err  error
	)
	switch p.GetCurrLiteral() {
	case "AND", "OR":
		stmt, err = p.parseRelational(left)
	case "LIKE", "ILIKE", "SIMILAR":
		stmt, err = p.parseLike(left)
	case "EXISTS":
		stmt, err = p.parseExists(left)
	case "BETWEEN":
		stmt, err = p.parseBetween(left)
		return reverse(stmt), err
	case "IN":
		stmt, err = p.parseIn(left)
	case "IS":
		stmt, err = p.parseIs(left)
	case "ISNULL":
		stmt, err = p.parseIsNull(left)
	case "NOTNULL":
		stmt, err = p.parseNotNull(left)
	default:
		err = p.Unexpected("expression")
	}
	return reverse(stmt), wrapError("keyword", err)
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
