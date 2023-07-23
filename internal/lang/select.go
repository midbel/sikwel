package lang

import (
	"strconv"
)

type SelectParser interface {
	ParseColumns() ([]Statement, error)
	ParseFrom() ([]Statement, error)
	ParseWhere() (Statement, error)
	ParseGroupBy() ([]Statement, error)
	ParseHaving() (Statement, error)
	ParseOrderBy() ([]Statement, error)
	ParseWindows() ([]Statement, error)
	ParseLimit() (Statement, error)
}

func (p *Parser) parseWith() (Statement, error) {
	p.Next()
	var (
		stmt WithStatement
		err  error
	)
	for !p.Done() && !p.Is(Keyword) {
		cte, err := p.parseSubquery()
		if err = wrapError("subquery", err); err != nil {
			return nil, err
		}
		stmt.Queries = append(stmt.Queries, cte)
		if err = p.EnsureEnd("with", Comma, Keyword); err != nil {
			return nil, err
		}
	}
	stmt.Statement, err = p.ParseStatement()
	return stmt, wrapError("with", err)
}

func (p *Parser) parseSubquery() (Statement, error) {
	var (
		cte CteStatement
		err error
	)
	if !p.Is(Ident) {
		return nil, p.Unexpected("subquery")
	}
	cte.Ident = p.curr.Literal
	p.Next()

	cte.Columns, err = p.parseColumnsList()
	if err != nil {
		return nil, err
	}

	if !p.IsKeyword("AS") {
		return nil, p.Unexpected("subquery")
	}
	p.Next()
	if !p.Is(Lparen) {
		return nil, p.Unexpected("subquery")
	}
	p.Next()

	cte.Statement, err = p.ParseStatement()
	if err = wrapError("subquery", err); err != nil {
		return nil, err
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("subquery")
	}
	p.Next()
	return cte, nil
}

func (p *Parser) ParseValues() (Statement, error) {
	p.Next()
	var (
		stmt ValuesStatement
		err  error
	)
	for !p.Done() && !p.Is(EOL) {
		expr, err := p.StartExpression()
		if err != nil {
			return nil, err
		}
		if err := p.EnsureEnd("values", Comma, EOL); err != nil {
			return nil, err
		}
		stmt.List = append(stmt.List, expr)
	}
	return stmt, err
}

func (p *Parser) ParseSelect() (Statement, error) {
	return p.ParseSelectStatement(p)
}

func (p *Parser) ParseSelectStatement(sp SelectParser) (Statement, error) {
	p.Next()
	var (
		stmt SelectStatement
		err  error
	)
	if stmt.Columns, err = sp.ParseColumns(); err != nil {
		return nil, err
	}
	if stmt.Tables, err = sp.ParseFrom(); err != nil {
		return nil, err
	}
	if stmt.Where, err = sp.ParseWhere(); err != nil {
		return nil, err
	}
	if stmt.Groups, err = sp.ParseGroupBy(); err != nil {
		return nil, err
	}
	if stmt.Having, err = sp.ParseHaving(); err != nil {
		return nil, err
	}
	if stmt.Windows, err = sp.ParseWindows(); err != nil {
		return nil, err
	}
	if stmt.Orders, err = sp.ParseOrderBy(); err != nil {
		return nil, err
	}
	if stmt.Limit, err = sp.ParseLimit(); err != nil {
		return nil, err
	}
	allDistinct := func() (bool, bool) {
		p.Next()
		var (
			all      = p.IsKeyword("ALL")
			distinct = p.IsKeyword("DISTINCT")
		)
		if all || distinct {
			p.Next()
		}
		return all, distinct
	}
	switch {
	case p.IsKeyword("UNION"):
		u := UnionStatement{
			Left: stmt,
		}
		u.All, u.Distinct = allDistinct()
		u.Right, err = p.ParseSelectStatement(sp)
		return u, err
	case p.IsKeyword("INTERSECT"):
		i := IntersectStatement{
			Left: stmt,
		}
		i.All, i.Distinct = allDistinct()
		i.Right, err = p.ParseSelectStatement(sp)
		return i, err
	case p.IsKeyword("EXCEPT"):
		e := ExceptStatement{
			Left: stmt,
		}
		e.All, e.Distinct = allDistinct()
		e.Right, err = p.ParseSelectStatement(sp)
		return e, err
	default:
		return stmt, err
	}
}

func (p *Parser) ParseColumns() ([]Statement, error) {
	var list []Statement
	for !p.Done() && !p.IsKeyword("FROM") {
		stmt, err := p.StartExpression()
		if err = wrapError("fields", err); err != nil {
			return nil, err
		}
		switch {
		case p.Is(Comma):
			p.Next()
			if p.IsKeyword("FROM") {
				return nil, p.Unexpected("fields")
			}
		case p.IsKeyword("FROM"):
		default:
			return nil, p.Unexpected("fields")
		}
		list = append(list, stmt)
	}
	if !p.IsKeyword("FROM") {
		return nil, p.Unexpected("fields")
	}
	return list, nil
}

func (p *Parser) ParseFrom() ([]Statement, error) {
	if !p.IsKeyword("FROM") {
		return nil, p.Unexpected("from")
	}
	p.Next()

	var (
		list []Statement
		err  error
	)
	for !p.Done() && !p.QueryEnds() {
		var stmt Statement
		stmt, err = p.StartExpression()
		if err != nil {
			return nil, err
		}
		list = append(list, stmt)
		if !p.Is(Comma) {
			break
		}
		p.Next()
		if p.QueryEnds() || p.Is(Keyword) {
			return nil, p.Unexpected("from")
		}
	}
	for !p.Done() && !p.QueryEnds() && isJoin(p.curr) {
		j := Join{
			Type: p.GetCurrLiteral(),
		}
		p.Next()
		j.Table, err = p.StartExpression()
		if err != nil {
			return nil, err
		}
		switch {
		case p.IsKeyword("ON"):
			j.Where, err = p.ParseJoinOn()
		case p.IsKeyword("USING"):
			j.Where, err = p.ParseJoinUsing()
		default:
			return nil, p.Unexpected("join")
		}
		if err = wrapError("join", err); err != nil {
			return nil, err
		}
		list = append(list, j)
	}
	return list, nil
}

func (p *Parser) ParseJoinOn() (Statement, error) {
	p.Next()
	p.UnregisterInfix("AS", Keyword)
	defer p.RegisterInfix("AS", Keyword, p.parseKeywordExpr)
	return p.StartExpression()
}

func (p *Parser) ParseJoinUsing() (Statement, error) {
	p.Next()
	if !p.Is(Lparen) {
		return nil, p.Unexpected("using")
	}
	p.Next()
	p.UnregisterInfix("AS", Keyword)
	defer p.RegisterInfix("AS", Keyword, p.parseKeywordExpr)

	var list List
	for !p.Done() && !p.Is(Rparen) {
		stmt, err := p.StartExpression()
		if err = wrapError("using", err); err != nil {
			return nil, err
		}
		list.Values = append(list.Values, stmt)
		if err := p.EnsureEnd("using", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("using")
	}
	p.Next()
	return list, nil
}

func (p *Parser) ParseWhere() (Statement, error) {
	if !p.IsKeyword("WHERE") {
		return nil, nil
	}
	p.Next()
	p.UnregisterInfix("AS", Keyword)
	defer p.RegisterInfix("AS", Keyword, p.parseKeywordExpr)
	return p.StartExpression()
}

func (p *Parser) ParseGroupBy() ([]Statement, error) {
	if !p.IsKeyword("GROUP BY") {
		return nil, nil
	}
	p.Next()
	var (
		list []Statement
		err  error
	)
	for !p.Done() && !p.QueryEnds() {
		var stmt Statement
		stmt, err = p.StartExpression()
		if err != nil {
			return nil, err
		}
		list = append(list, stmt)
		if !p.Is(Comma) {
			break
		}
		p.Next()
		if p.QueryEnds() && !p.Is(Keyword) {
			return nil, p.Unexpected("group by")
		}
	}
	return list, err
}

func (p *Parser) ParseHaving() (Statement, error) {
	if !p.IsKeyword("HAVING") {
		return nil, nil
	}
	p.Next()
	p.UnregisterInfix("AS", Keyword)
	defer p.RegisterInfix("AS", Keyword, p.parseKeywordExpr)
	return p.StartExpression()
}

func (p *Parser) ParseWindows() ([]Statement, error) {
	if !p.IsKeyword("WINDOW") {
		return nil, nil
	}
	p.Next()
	var (
		list []Statement
		err  error
	)
	for !p.Done() && !p.QueryEnds() {
		var win WindowDefinition
		if win.Ident, err = p.ParseIdentifier(); err != nil {
			return nil, err
		}
		if !p.IsKeyword("AS") {
			return nil, p.Unexpected("windoow")
		}
		p.Next()
		if win.Window, err = p.ParseWindow(); err != nil {
			return nil, err
		}
		list = append(list, win)
		if !p.Is(Comma) {
			break
		}
		p.Next()
		if p.Is(Keyword) || p.QueryEnds() {
			return nil, p.Unexpected("window")
		}
	}
	return list, err
}

func (p *Parser) ParseWindow() (Statement, error) {
	var (
		stmt Window
		err  error
	)
	if !p.Is(Lparen) {
		return nil, p.Unexpected("window")
	}
	p.Next()
	switch {
	case p.Is(Ident):
		stmt.Ident, err = p.ParseIdentifier()
	case p.IsKeyword("PARTITION BY"):
		p.Next()
		for !p.Done() && !p.IsKeyword("ORDER BY") && !p.Is(Rparen) {
			expr, err := p.StartExpression()
			if err != nil {
				return nil, err
			}
			stmt.Partitions = append(stmt.Partitions, expr)
			switch {
			case p.Is(Comma):
				p.Next()
				if p.IsKeyword("ORDER BY") || p.Is(Rparen) {
					return nil, p.Unexpected("window")
				}
			case p.IsKeyword("ORDER BY"):
			case p.Is(Rparen):
			default:
				return nil, p.Unexpected("window")
			}
		}
	default:
		return nil, p.Unexpected("window")
	}
	if err != nil {
		return nil, err
	}
	if stmt.Orders, err = p.ParseOrderBy(); err != nil {
		return nil, err
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("window")
	}
	p.Next()
	return stmt, err
}

func (p *Parser) parseFrameSpec() (Statement, error) {
	switch {
	case p.IsKeyword("RANGE"):
	case p.IsKeyword("ROWS"):
	case p.IsKeyword("GROUPS"):
	default:
		return nil, nil
	}
	p.Next()
	var stmt BetweenFrameSpec
	if !p.IsKeyword("BETWEEN") {
		stmt.Right.Row = RowCurrent
	}
	p.Next()

	switch {
	case p.IsKeyword("CURRENT ROW"):
		stmt.Left.Row = RowCurrent
	case p.IsKeyword("UNBOUNDED PRECEDING"):
		stmt.Left.Row = RowPreceding | RowUnbounded
	default:
		expr, err := p.StartExpression()
		if err != nil {
			return nil, err
		}
		stmt.Left.Row = RowPreceding
		stmt.Left.Expr = expr
		if !p.IsKeyword("PRECEDING") && !p.IsKeyword("FOLLOWING") {
			return nil, p.Unexpected("frame spec")
		}
	}
	p.Next()
	if stmt.Right.Row == 0 {
		if !p.IsKeyword("AND") {
			return nil, p.Unexpected("frame spec")
		}
		p.Next()
		switch {
		case p.IsKeyword("CURRENT ROW"):
			stmt.Right.Row = RowCurrent
		case p.IsKeyword("UNBOUNDED FOLLOWING"):
			stmt.Right.Row = RowFollowing | RowUnbounded
		default:
			expr, err := p.StartExpression()
			if err != nil {
				return nil, err
			}
			stmt.Right.Row = RowFollowing
			stmt.Right.Expr = expr
			if !p.IsKeyword("PRECEDING") && !p.IsKeyword("FOLLOWING") {
				return nil, p.Unexpected("frame spec")
			}
		}
		p.Next()
	}
	switch {
	case p.IsKeyword("EXCLUDE NO OTHERS"):
		stmt.Exclude = ExcludeNoOthers
	case p.IsKeyword("EXCLUDE CURRENT ROW"):
		stmt.Exclude = ExcludeCurrent
	case p.IsKeyword("EXCLUDE GROUP"):
		stmt.Exclude = ExcludeGroup
	case p.IsKeyword("EXCLUDE TIES"):
		stmt.Exclude = ExcludeTies
	default:
	}
	if stmt.Exclude > 0 {
		p.Next()
	}
	return stmt, nil
}

func (p *Parser) ParseOrderBy() ([]Statement, error) {
	if !p.IsKeyword("ORDER BY") {
		return nil, nil
	}
	p.Next()
	var (
		list []Statement
		err  error
	)
	for !p.Done() && !p.QueryEnds() {
		var stmt Statement
		stmt, err = p.StartExpression()
		if err != nil {
			return nil, err
		}
		order := Order{
			Statement: stmt,
		}
		if p.IsKeyword("ASC") || p.IsKeyword("DESC") {
			order.Orient = p.GetCurrLiteral()
			p.Next()
		}
		if p.IsKeyword("NULLS") {
			p.Next()
			if !p.IsKeyword("FIRST") && !p.IsKeyword("LAST") {
				return nil, p.Unexpected("order by")
			}
			order.Nulls = p.GetCurrLiteral()
			p.Next()
		}
		list = append(list, order)
		if !p.Is(Comma) {
			break
		}
		p.Next()
		if p.QueryEnds() || p.Is(Rparen) || p.Is(Keyword) {
			return nil, p.Unexpected("order by")
		}
	}
	return list, err
}

func (p *Parser) ParseLimit() (Statement, error) {
	switch {
	case p.IsKeyword("LIMIT"):
		var (
			lim Limit
			err error
		)
		p.Next()
		lim.Count, err = strconv.Atoi(p.GetCurrLiteral())
		if err != nil {
			return nil, p.Unexpected("limit")
		}
		p.Next()
		if !p.Is(Comma) && !p.IsKeyword("OFFSET") {
			return lim, nil
		}
		p.Next()
		lim.Offset, err = strconv.Atoi(p.GetCurrLiteral())
		if err != nil {
			return nil, p.Unexpected("offset")
		}
		p.Next()
		return lim, nil
	case p.IsKeyword("OFFSET") || p.IsKeyword("FETCH"):
		return p.ParseFetch()
	default:
		return nil, nil
	}
}

func (p *Parser) ParseFetch() (Statement, error) {
	var (
		stmt Offset
		err  error
	)
	if p.IsKeyword("OFFSET") {
		p.Next()
		stmt.Offset, err = strconv.Atoi(p.GetCurrLiteral())
		if err != nil {
			return nil, p.Unexpected("fetch")
		}
		p.Next()
		if !p.IsKeyword("ROW") && !p.IsKeyword("ROWS") {
			return nil, p.Unexpected("fetch")
		}
		p.Next()
	}
	if !p.IsKeyword("FETCH") {
		return nil, p.Unexpected("fetch")
	}
	p.Next()
	if p.IsKeyword("NEXT") {
		stmt.Next = true
	} else if p.IsKeyword("FIRST") {
		stmt.Next = false
	} else {
		return nil, p.Unexpected("fetch")
	}
	p.Next()
	stmt.Count, err = strconv.Atoi(p.GetCurrLiteral())
	if err != nil {
		return nil, p.Unexpected("fetch")
	}
	p.Next()
	if !p.IsKeyword("ROW") && !p.IsKeyword("ROWS") {
		return nil, p.Unexpected("fetch")
	}
	p.Next()
	if !p.IsKeyword("ONLY") {
		return nil, p.Unexpected("fetch")
	}
	p.Next()
	return stmt, err
}
