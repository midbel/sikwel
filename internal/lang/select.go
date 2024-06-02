package lang

import (
	"fmt"
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

	p.setFuncSetForTable()
	defer p.unsetFuncSet()

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
	p.setDefaultFuncSet()
	p.UnregisterInfix("AS", Keyword)
	defer p.unsetFuncSet()
	return p.StartExpression()
}

func (p *Parser) ParseJoinUsing() (Statement, error) {
	p.Next()
	if !p.Is(Lparen) {
		return nil, p.Unexpected("using")
	}
	p.Next()

	var list List
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

func (p *Parser) ParseWhere() (Statement, error) {
	if !p.IsKeyword("WHERE") {
		return nil, nil
	}
	p.Next()
	p.toggleAlias()
	defer p.toggleAlias()
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

func (p *Parser) ParseHaving() (Statement, error) {
	if !p.IsKeyword("HAVING") {
		return nil, nil
	}
	p.Next()
	p.toggleAlias()
	defer p.toggleAlias()
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
		stmt, err = p.ParseIdentifier()
		if err != nil {
			return nil, err
		}
		order := Order{
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

func (w *Writer) FormatUnion(stmt UnionStatement) error {
	if err := w.FormatStatement(stmt.Left); err != nil {
		return err
	}
	w.WriteNL()
	w.WriteKeyword("UNION")
	if stmt.All {
		w.WriteBlank()
		w.WriteKeyword("ALL")
	}
	if stmt.Distinct {
		w.WriteBlank()
		w.WriteKeyword("DISTINCT")
	}
	w.WriteNL()
	return w.FormatStatement(stmt.Right)
}

func (w *Writer) FormatExcept(stmt ExceptStatement) error {
	if err := w.FormatStatement(stmt.Left); err != nil {
		return err
	}
	w.WriteNL()
	w.WriteKeyword("EXCEPT")
	if stmt.All {
		w.WriteBlank()
		w.WriteKeyword("ALL")
	}
	if stmt.Distinct {
		w.WriteBlank()
		w.WriteKeyword("DISTINCT")
	}
	w.WriteNL()
	return w.FormatStatement(stmt.Right)
}

func (w *Writer) FormatIntersect(stmt IntersectStatement) error {
	if err := w.FormatStatement(stmt.Left); err != nil {
		return err
	}
	w.WriteNL()
	w.WriteKeyword("INTERSECT")
	if stmt.All {
		w.WriteBlank()
		w.WriteKeyword("ALL")
	}
	if stmt.Distinct {
		w.WriteBlank()
		w.WriteKeyword("DISTINCT")
	}
	w.WriteNL()
	return w.FormatStatement(stmt.Right)
}

func (w *Writer) FormatValues(stmt ValuesStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	return w.formatStmtSlice(stmt.List)
}

func (w *Writer) FormatSelect(stmt SelectStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteNL()
	if err := w.FormatSelectColumns(stmt.Columns); err != nil {
		return err
	}
	w.WriteNL()
	if err := w.FormatFrom(stmt.Tables); err != nil {
		return err
	}
	if stmt.Where != nil {
		w.WriteNL()
		if err := w.FormatWhere(stmt.Where); err != nil {
			return err
		}
	}
	if len(stmt.Groups) > 0 {
		w.WriteNL()
		if err := w.FormatGroupBy(stmt.Groups); err != nil {
			return err
		}
	}
	if stmt.Having != nil {
		w.WriteNL()
		if err := w.FormatHaving(stmt.Having); err != nil {
			return err
		}
	}
	if len(stmt.Windows) > 0 {
		w.WriteNL()
		if err := w.FormatWindows(stmt.Windows); err != nil {
			return err
		}
	}
	if len(stmt.Orders) > 0 {
		w.WriteNL()
		if err := w.FormatOrderBy(stmt.Orders); err != nil {
			return err
		}
	}
	if stmt.Limit != nil {
		w.WriteNL()
		if err := w.FormatLimit(stmt.Limit); err != nil {
			return nil
		}
	}
	return nil
}

func (w *Writer) FormatSelectColumns(columns []Statement) error {
	w.Enter()
	defer w.Leave()
	return w.formatStmtSlice(columns)
}

func (w *Writer) FormatWhere(stmt Statement) error {
	if stmt == nil {
		return nil
	}
	w.WriteStatement("WHERE")
	w.WriteBlank()

	currDepth := w.currDepth
	w.Enter()
	defer func() {
		w.Leave()
		w.currDepth = currDepth
	}()

	return w.FormatExpr(stmt, true)
}

func (w *Writer) formatFromJoin(join Join) error {
	w.WriteString(join.Type)
	w.WriteBlank()

	var err error
	switch s := join.Table.(type) {
	case Name:
		w.FormatName(s)
	case Alias:
		err = w.FormatAlias(s)
	case SelectStatement:
		w.WriteString("(")
		err = w.FormatSelect(s)
		w.WriteString(")")
	default:
		return w.CanNotUse("from", s)
		err = fmt.Errorf("from: unsupported statement (%T)", s)
	}
	if err != nil {
		return err
	}
	switch s := join.Where.(type) {
	case Binary:
		w.WriteBlank()
		w.WriteKeyword("ON")
		w.WriteBlank()
		err = w.formatBinary(s, false)
	case List:
		w.WriteBlank()
		w.WriteKeyword("USING")
		w.WriteBlank()
		err = w.formatList(s)
	default:
		return w.CanNotUse("from", s)
	}
	return err
}

func (w *Writer) FormatFrom(list []Statement) error {
	w.WriteStatement("FROM")
	w.WriteBlank()

	w.Enter()
	defer w.Leave()

	var err error
	for i, s := range list {
		if i > 0 {
			w.WriteNL()
			w.WritePrefix()
		}
		switch s := s.(type) {
		case Name:
			w.FormatName(s)
		case Alias:
			err = w.FormatAlias(s)
		case Join:
			err = w.formatFromJoin(s)
		case SelectStatement:
			w.WriteString("(")
			err = w.FormatStatement(s)
			if err == nil {
				w.WriteNL()
				w.WriteString(")")
				w.WriteNL()
			}
		case Row:
			err = w.formatRow(s, true)
		default:
			err = w.CanNotUse("from", s)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatGroupBy(groups []Statement) error {
	if len(groups) == 0 {
		return nil
	}
	w.WriteStatement("GROUP BY")
	w.WriteBlank()
	for i, s := range groups {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		n, ok := s.(Name)
		if !ok {
			return w.CanNotUse("group by", s)
		}
		w.FormatName(n)
	}
	return nil
}

func (w *Writer) FormatWindows(windows []Statement) error {
	w.WriteStatement("WINDOW")

	w.Enter()
	defer w.Leave()

	if len(windows) > 1 {
		w.WriteNL()
		w.WritePrefix()
	} else {
		w.WriteBlank()
	}

	for i, c := range windows {
		def, ok := c.(WindowDefinition)
		if !ok {
			return fmt.Errorf("window: unexpected statement type %T", c)
		}
		if i > 0 {
			w.WriteString(",")
			w.WriteNL()
			w.WritePrefix()
		}
		if err := w.FormatExpr(def.Ident, false); err != nil {
			return err
		}
		w.WriteBlank()
		w.WriteKeyword("AS")
		w.WriteBlank()
		w.WriteString("(")
		win, ok := def.Window.(Window)
		if !ok {
			return fmt.Errorf("window: unexpected statement type %T", def.Window)
		}
		if win.Ident != nil {
			if err := w.FormatExpr(win.Ident, false); err != nil {
				return err
			}
			w.WriteBlank()
		}
		if win.Ident == nil && len(win.Partitions) > 0 {
			w.WriteKeyword("PARTITION BY")
			w.WriteBlank()
			if err := w.formatStmtSlice(win.Partitions); err != nil {
				return err
			}
		}
		if len(win.Orders) > 0 {
			w.WriteBlank()
			w.WriteKeyword("ORDER BY")
			w.WriteBlank()
			for i, s := range win.Orders {
				if i > 0 {
					w.WriteString(",")
					w.WriteBlank()
				}
				order, ok := s.(Order)
				if !ok {
					return w.CanNotUse("order by", s)
				}
				if err := w.formatOrder(order); err != nil {
					return err
				}
			}
		}
		w.WriteString(")")
	}
	return nil
}

func (w *Writer) FormatHaving(having Statement) error {
	w.Enter()
	defer w.Leave()

	if having == nil {
		return nil
	}
	w.WriteStatement("HAVING")
	w.WriteBlank()
	return w.FormatExpr(having, true)
}

func (w *Writer) FormatOrderBy(orders []Statement) error {
	if len(orders) == 0 {
		return nil
	}
	w.WriteStatement("ORDER BY")
	w.WriteBlank()
	for i, s := range orders {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		order, ok := s.(Order)
		if !ok {
			return w.CanNotUse("order by", s)
		}
		if err := w.formatOrder(order); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) formatOrder(order Order) error {
	n, ok := order.Statement.(Name)
	if !ok {
		return w.CanNotUse("order by", order.Statement)
	}
	w.FormatName(n)
	if order.Dir != "" {
		w.WriteBlank()
		w.WriteString(order.Dir)
	}
	if order.Nulls != "" {
		w.WriteBlank()
		w.WriteKeyword("NULLS")
		w.WriteBlank()
		w.WriteString(order.Nulls)
	}
	return nil
}

func (w *Writer) FormatLimit(limit Statement) error {
	if limit == nil {
		return nil
	}
	lim, ok := limit.(Limit)
	if !ok {
		return w.FormatOffset(limit)
	}
	w.WriteStatement("LIMIT")
	w.WriteBlank()
	w.WriteString(strconv.Itoa(lim.Count))
	if lim.Offset > 0 {
		w.WriteBlank()
		w.WriteKeyword("OFFSET")
		w.WriteBlank()
		w.WriteString(strconv.Itoa(lim.Offset))
	}
	return nil
}

func (w *Writer) FormatOffset(limit Statement) error {
	lim, ok := limit.(Offset)
	if !ok {
		return w.CanNotUse("fetch", limit)
	}
	w.WritePrefix()
	if lim.Offset > 0 {
		w.WriteKeyword("OFFSET")
		w.WriteBlank()
		w.WriteString(strconv.Itoa(lim.Offset))
		w.WriteBlank()
		w.WriteKeyword("ROWS")
		w.WriteBlank()
	}
	w.WriteKeyword("FETCH")
	w.WriteBlank()
	if lim.Next {
		w.WriteKeyword("NEXT")
	} else {
		w.WriteKeyword("FIRST")
	}
	w.WriteBlank()
	w.WriteString(strconv.Itoa(lim.Count))
	w.WriteBlank()
	w.WriteKeyword("ROWS ONLY")
	return nil
}
