package lang

import (
	"strconv"

	"github.com/midbel/sweet/internal/lang/ast"
)

type SelectParser interface {
	ParseColumns() ([]ast.Statement, error)
	ParseFrom() ([]ast.Statement, error)
	ParseWhere() (ast.Statement, error)
	ParseGroupBy() ([]ast.Statement, error)
	ParseHaving() (ast.Statement, error)
	ParseOrderBy() ([]ast.Statement, error)
	ParseWindows() ([]ast.Statement, error)
	ParseLimit() (ast.Statement, error)
}

func (p *Parser) ParseValues() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.ValuesStatement
		err  error
	)
	if !p.Is(Lparen) {
		for !p.Done() && !p.Is(EOL) {
			v, err := p.StartExpression()
			if err != nil {
				return nil, err
			}
			if err := p.EnsureEnd("values", Comma, EOL); err != nil {
				return nil, err
			}
			stmt.List = append(stmt.List, v)
		}
		return stmt, nil
	}
	for !p.Done() && !p.Is(EOL) {
		if !p.Is(Lparen) {
			return nil, p.Unexpected("values")
		}
		p.Next()
		var list ast.List
		for !p.Done() && !p.Is(Rparen) {
			v, err := p.StartExpression()
			if err != nil {
				return nil, err
			}
			list.Values = append(list.Values, v)
			switch {
			case p.Is(Comma):
				p.Next()
				if p.Is(Rparen) {
					return nil, p.Unexpected("values")
				}
			case p.Is(Rparen):
			default:
				return nil, p.Unexpected("values")
			}
		}
		if !p.Is(Rparen) {
			return nil, p.Unexpected("values")
		}
		p.Next()
		stmt.List = append(stmt.List, list)
		if !p.Is(Comma) {
			break
		}
		p.Next()
	}
	return stmt, err
}

func (p *Parser) ParseSelect() (ast.Statement, error) {
	return p.ParseSelectStatement(p)
}

func (p *Parser) ParseSelectStatement(sp SelectParser) (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.SelectStatement
		err  error
	)
	if p.IsKeyword("DISTINCT") {
		stmt.Distinct = true
		p.Next()
	}
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
		u := ast.UnionStatement{
			Left: stmt,
		}
		u.All, u.Distinct = allDistinct()
		u.Right, err = p.ParseSelectStatement(sp)
		return u, err
	case p.IsKeyword("INTERSECT"):
		i := ast.IntersectStatement{
			Left: stmt,
		}
		i.All, i.Distinct = allDistinct()
		i.Right, err = p.ParseSelectStatement(sp)
		return i, err
	case p.IsKeyword("EXCEPT"):
		e := ast.ExceptStatement{
			Left: stmt,
		}
		e.All, e.Distinct = allDistinct()
		e.Right, err = p.ParseSelectStatement(sp)
		return e, err
	default:
		return stmt, err
	}
}

func (p *Parser) ParseColumns() ([]ast.Statement, error) {
	var list []ast.Statement
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

func (p *Parser) ParseFrom() ([]ast.Statement, error) {
	if !p.IsKeyword("FROM") {
		return nil, p.Unexpected("from")
	}
	p.Next()

	p.setFuncSetForTable()
	defer p.unsetFuncSet()

	var (
		list []ast.Statement
		err  error
	)
	for !p.Done() && !p.QueryEnds() {
		var stmt ast.Statement
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
		j := ast.Join{
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

func (p *Parser) ParseJoinOn() (ast.Statement, error) {
	p.Next()
	p.setDefaultFuncSet()
	p.UnregisterInfix("AS", Keyword)
	defer p.unsetFuncSet()
	return p.StartExpression()
}

func (p *Parser) ParseJoinUsing() (ast.Statement, error) {
	p.Next()
	if !p.Is(Lparen) {
		return nil, p.Unexpected("using")
	}
	p.Next()

	var list ast.List
	for !p.Done() && !p.Is(Rparen) {
		stmt, err := p.ParseIdentifier()
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

func (p *Parser) ParseWhere() (ast.Statement, error) {
	if !p.IsKeyword("WHERE") {
		return nil, nil
	}
	p.Next()
	p.toggleAlias()
	defer p.toggleAlias()
	return p.StartExpression()
}

func (p *Parser) ParseGroupBy() ([]ast.Statement, error) {
	if !p.IsKeyword("GROUP BY") {
		return nil, nil
	}
	p.Next()
	var (
		list []ast.Statement
		err  error
	)
	for !p.Done() && !p.QueryEnds() {
		var stmt ast.Statement
		stmt, err = p.ParseIdentifier()
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

func (p *Parser) ParseHaving() (ast.Statement, error) {
	if !p.IsKeyword("HAVING") {
		return nil, nil
	}
	p.Next()
	p.toggleAlias()
	defer p.toggleAlias()
	return p.StartExpression()
}

func (p *Parser) ParseWindows() ([]ast.Statement, error) {
	if !p.IsKeyword("WINDOW") {
		return nil, nil
	}
	p.Next()
	var (
		list []ast.Statement
		err  error
	)
	for !p.Done() && !p.QueryEnds() {
		var win ast.WindowDefinition
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

func (p *Parser) ParseWindow() (ast.Statement, error) {
	var (
		stmt ast.Window
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

func (p *Parser) parseFrameSpec() (ast.Statement, error) {
	switch {
	case p.IsKeyword("RANGE"):
	case p.IsKeyword("ROWS"):
	case p.IsKeyword("GROUPS"):
	default:
		return nil, nil
	}
	p.Next()
	var stmt ast.BetweenFrameSpec
	if !p.IsKeyword("BETWEEN") {
		stmt.Right.Row = ast.RowCurrent
	}
	p.Next()

	switch {
	case p.IsKeyword("CURRENT ROW"):
		stmt.Left.Row = ast.RowCurrent
	case p.IsKeyword("UNBOUNDED PRECEDING"):
		stmt.Left.Row = ast.RowPreceding | ast.RowUnbounded
	default:
		expr, err := p.StartExpression()
		if err != nil {
			return nil, err
		}
		stmt.Left.Row = ast.RowPreceding
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
			stmt.Right.Row = ast.RowCurrent
		case p.IsKeyword("UNBOUNDED FOLLOWING"):
			stmt.Right.Row = ast.RowFollowing | ast.RowUnbounded
		default:
			expr, err := p.StartExpression()
			if err != nil {
				return nil, err
			}
			stmt.Right.Row = ast.RowFollowing
			stmt.Right.Expr = expr
			if !p.IsKeyword("PRECEDING") && !p.IsKeyword("FOLLOWING") {
				return nil, p.Unexpected("frame spec")
			}
		}
		p.Next()
	}
	switch {
	case p.IsKeyword("EXCLUDE NO OTHERS"):
		stmt.Exclude = ast.ExcludeNoOthers
	case p.IsKeyword("EXCLUDE CURRENT ROW"):
		stmt.Exclude = ast.ExcludeCurrent
	case p.IsKeyword("EXCLUDE GROUP"):
		stmt.Exclude = ast.ExcludeGroup
	case p.IsKeyword("EXCLUDE TIES"):
		stmt.Exclude = ast.ExcludeTies
	default:
	}
	if stmt.Exclude > 0 {
		p.Next()
	}
	return stmt, nil
}

func (p *Parser) ParseOrderBy() ([]ast.Statement, error) {
	if !p.IsKeyword("ORDER BY") {
		return nil, nil
	}
	p.Next()
	var (
		list []ast.Statement
		err  error
	)
	for !p.Done() && !p.QueryEnds() {
		var stmt ast.Statement
		stmt, err = p.ParseIdentifier()
		if err != nil {
			return nil, err
		}
		order := ast.Order{
			Statement: stmt,
			Dir:       "ASC",
		}
		if p.IsKeyword("ASC") || p.IsKeyword("DESC") {
			order.Dir = p.GetCurrLiteral()
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

func (p *Parser) ParseLimit() (ast.Statement, error) {
	switch {
	case p.IsKeyword("LIMIT"):
		var (
			lim ast.Limit
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

func (p *Parser) ParseFetch() (ast.Statement, error) {
	var (
		stmt ast.Offset
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
