package lang

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type SelectParser interface {
	ParseColumns() ([]Statement, error)
	ParseFrom() ([]Statement, error)
	ParseWhere() (Statement, error)
	ParseGroupBy() ([]Statement, error)
	ParseHaving() (Statement, error)
	ParseOrderBy() ([]Statement, error)
	ParseLimit() (Statement, error)
}

type Parser struct {
	*frame
	stack []*frame

	keywords map[string]func() (Statement, error)
	infix    map[symbol]infixFunc
	prefix   map[symbol]prefixFunc
}

func NewParser(r io.Reader) (*Parser, error) {
	return NewParserWithKeywords(r, keywords)
}

func NewParserWithKeywords(r io.Reader, set KeywordSet) (*Parser, error) {
	var p Parser

	frame, err := createFrame(r, set)
	if err != nil {
		return nil, err
	}
	p.frame = frame
	p.keywords = make(map[string]func() (Statement, error))

	p.RegisterParseFunc("SELECT", p.ParseSelect)
	p.RegisterParseFunc("VALUES", p.parseValues)
	p.RegisterParseFunc("DELETE FROM", p.parseDelete)
	p.RegisterParseFunc("UPDATE", p.ParseUpdate)
	p.RegisterParseFunc("INSERT INTO", p.ParseInsert)
	p.RegisterParseFunc("WITH", p.parseWith)
	p.RegisterParseFunc("IF", p.parseIf)
	p.RegisterParseFunc("CASE", p.parseCase)
	p.RegisterParseFunc("WHILE", p.parseWhile)
	p.RegisterParseFunc("COMMIT", p.parseCommit)
	p.RegisterParseFunc("ROLLBACK", p.parseRollback)
	p.RegisterParseFunc("DECLARE", p.parseDeclare)
	p.RegisterParseFunc("SET", p.parseSet)
	p.RegisterParseFunc("RETURN", p.parseReturn)

	p.infix = make(map[symbol]infixFunc)
	p.registerInfix("", Plus, p.parseInfixExpr)
	p.registerInfix("", Minus, p.parseInfixExpr)
	p.registerInfix("", Slash, p.parseInfixExpr)
	p.registerInfix("", Star, p.parseInfixExpr)
	p.registerInfix("", Concat, p.parseInfixExpr)
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
	p.registerInfix("IS", Keyword, p.parseKeywordExpr)

	p.prefix = make(map[symbol]prefixFunc)
	p.registerPrefix("", Ident, p.ParseIdent)
	p.registerPrefix("", Star, p.ParseIdent)
	p.registerPrefix("", Literal, p.ParseLiteral)
	p.registerPrefix("", Number, p.ParseLiteral)
	p.registerPrefix("", Lparen, p.parseGroupExpr)
	p.registerPrefix("", Minus, p.parseUnary)
	p.registerPrefix("", Keyword, p.parseUnary)
	p.registerPrefix("NOT", Keyword, p.parseUnary)
	p.registerPrefix("NULL", Keyword, p.parseUnary)
	p.registerPrefix("DEFAULT", Keyword, p.parseUnary)
	p.registerPrefix("CASE", Keyword, p.parseCase)
	p.registerPrefix("SELECT", Keyword, p.ParseSelect)

	return &p, nil
}

func (p *Parser) RegisterParseFunc(kw string, fn func() (Statement, error)) {
	kw = strings.ToUpper(kw)
	p.keywords[kw] = fn
}

func (p *Parser) UnregisterParseFunc(kw string) {
	kw = strings.ToUpper(kw)
	delete(p.keywords, kw)
}

func (p *Parser) Parse() (Statement, error) {
	for p.Is(Comment) {
		p.Next()
	}
	if p.Is(Macro) {
		if err := p.ParseMacro(); err != nil {
			return nil, err
		}
		return p.Parse()
	}
	stmt, err := p.parseStatement()
	if err != nil {
		return nil, err
	}
	if !p.Is(EOL) {
		return nil, p.wantError("statement", ";")
	}
	p.Next()
	return stmt, nil
}

func (p *Parser) ParseMacro() error {
	var err error
	switch p.curr.Literal {
	case "INCLUDE":
		err = p.ParseIncludeMacro()
	case "DEFINE":
		err = p.ParseDefineMacro()
	case "USE":
		err = p.ParseUseMacro()
	default:
		err = fmt.Errorf("macro %s unsupported", p.curr.Literal)
	}
	if err != nil {
		return err
	}
	return nil
}

func (p *Parser) ParseIncludeMacro() error {
	p.Next()

	file := filepath.Join(p.base, p.curr.Literal)
	p.Next()

	if !p.Is(EOL) {
		return p.wantError("include", ";")
	}
	p.Next()

	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	frame, err := createFrame(r, p.frame.set)
	if err != nil {
		return err
	}
	p.stack = append(p.stack, p.frame)
	p.frame = frame

	return nil
}

func (p *Parser) ParseDefineMacro() error {
	return nil
}

func (p *Parser) ParseUseMacro() error {
	return nil
}

func (p *Parser) parseStatement() (Statement, error) {
	if p.Done() {
		return nil, io.EOF
	}
	if !p.Is(Keyword) {
		return nil, p.wantError("statement", "keyword")
	}
	fn, ok := p.keywords[p.curr.Literal]
	if !ok {
		return nil, p.Unexpected("statement")
	}
	return fn()
}

func (p *Parser) parseDeclare() (Statement, error) {
	p.Next()

	var (
		stmt Declare
		err  error
	)
	if !p.Is(Ident) {
		return nil, p.Unexpected("declare")
	}
	stmt.Ident = p.curr.Literal
	p.Next()

	stmt.Type, err = p.parseType()
	if err != nil {
		return nil, err
	}

	if p.IsKeyword("DEFAULT") {
		p.Next()
		stmt.Value, err = p.parseExpression(powLowest, p.tokCheck(EOL))
		if err != nil {
			return nil, err
		}
	}
	return stmt, nil
}

func (p *Parser) parseReturn() (Statement, error) {
	p.Next()
	var (
		ret Return
		err error
	)
	ret.Statement, err = p.parseExpression(powLowest, p.tokCheck(EOL))
	return ret, err
}

func (p *Parser) parseType() (Type, error) {
	var t Type
	if !p.Is(Ident) {
		return t, p.Unexpected("type")
	}
	t.Name = p.curr.Literal
	p.Next()
	if p.Is(Lparen) {
		p.Next()
		if !p.Is(Number) {
			return t, p.Unexpected("type")
		}
		size, err := strconv.Atoi(p.curr.Literal)
		if err != nil {
			return t, err
		}
		t.Length = size
		p.Next()
		if !p.Is(Rparen) {
			return t, p.Unexpected("type")
		}
		p.Next()
	}
	return t, nil
}

func (p *Parser) parseSet() (Statement, error) {
	p.Next()
	var (
		stmt Assignment
		err  error
	)
	if stmt.Field, err = p.ParseIdent(); err != nil {
		return nil, wrapError("set", err)
	}
	if !p.Is(Eq) {
		return nil, p.Unexpected("set")
	}
	p.Next()

	stmt.Value, err = p.parseExpression(powLowest, p.tokCheck(EOL))
	return stmt, err
}

func (p *Parser) parseIf() (Statement, error) {
	p.Next()

	var (
		stmt IfStatement
		err  error
	)
	if stmt.Cdt, err = p.parseExpression(powLowest, p.kwCheck("THEN")); err != nil {
		return nil, err
	}
	if !p.IsKeyword("THEN") {
		return nil, p.Unexpected("if")
	}
	p.Next()
	stmt.Csq, err = p.parseBody(p.kwCheck("ELSE", "ELSIF", "END IF"))
	if err != nil {
		return nil, err
	}
	switch {
	case p.IsKeyword("ELSE"):
		p.Next()
		stmt.Alt, err = p.parseBody(p.kwCheck("END IF"))
	case p.IsKeyword("ELSIF"):
		stmt.Alt, err = p.parseIf()
	case p.IsKeyword("END IF"):
	default:
		return nil, p.Unexpected("if")
	}
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("END IF") {
		return nil, p.Unexpected("if")
	}
	p.Next()
	return stmt, nil
}

func (p *Parser) parseWhile() (Statement, error) {
	var (
		stmt WhileStatement
		err  error
	)
	p.Next()

	stmt.Cdt, err = p.parseExpression(powLowest, p.kwCheck("DO"))
	if err = wrapError("while", err); err != nil {
		return nil, err
	}
	if !p.IsKeyword("DO") {
		return nil, p.Unexpected("while")
	}
	p.Next()
	stmt.Body, err = p.parseBody(p.kwCheck("END WHILE"))
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("END WHILE") {
		return nil, p.Unexpected("while")
	}
	p.Next()
	return stmt, nil
}

func (p *Parser) parseBody(done func() bool) (Statement, error) {
	var list List
	for !p.Done() && !done() {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if !p.Is(EOL) {
			return nil, p.Unexpected("body")
		}
		p.Next()
		list.Values = append(list.Values, stmt)
	}
	if !done() {
		return nil, p.Unexpected("body")
	}
	return list, nil
}

func (p *Parser) parseCommit() (Statement, error) {
	p.Next()
	return Commit{}, nil
}

func (p *Parser) parseRollback() (Statement, error) {
	p.Next()
	return Rollback{}, nil
}

func (p *Parser) parseBegin() (Statement, error) {
	return nil, nil
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
	stmt.Statement, err = p.parseStatement()
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

	cte.Statement, err = p.parseStatement()
	if err = wrapError("subquery", err); err != nil {
		return nil, err
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("subquery")
	}
	p.Next()
	return cte, nil
}

func (p *Parser) parseDelete() (Statement, error) {
	p.Next()
	var (
		stmt DeleteStatement
		err  error
	)
	if !p.Is(Ident) {
		return nil, p.Unexpected("delete")
	}
	stmt.Table = p.curr.Literal
	p.Next()

	if stmt.Where, err = p.ParseWhere(); err != nil {
		return nil, wrapError("delete", err)
	}
	if stmt.Return, err = p.ParseReturning(); err != nil {
		return nil, wrapError("delete", err)
	}
	return stmt, nil
}

func (p *Parser) ParseUpdate() (Statement, error) {
	p.Next()
	var (
		stmt UpdateStatement
		err  error
	)
	stmt.Table, err = p.ParseIdent()
	if err != nil {
		return nil, err
	}

	if !p.IsKeyword("SET") {
		return nil, p.Unexpected("update")
	}
	p.Next()

	if stmt.List, err = p.ParseUpdateList(); err != nil {
		return nil, err
	}

	if p.IsKeyword("FROM") {
		_, err = p.ParseFrom()
		if err != nil {
			return nil, err
		}
	}
	if stmt.Where, err = p.ParseWhere(); err != nil {
		return nil, err
	}
	stmt.Return, err = p.ParseReturning()
	return stmt, wrapError("update", err)
}

func (p *Parser) ParseUpdateList() ([]Statement, error) {
	var list []Statement
	for !p.Done() && !p.Is(EOL) && !p.IsKeyword("WHERE") && !p.IsKeyword("FROM") && !p.IsKeyword("RETURNING") {
		stmt, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}
		if p.Is(EOL) {
			break
		}
		if err := p.EnsureEnd("update", Comma, Keyword); err != nil {
			return nil, err
		}
		list = append(list, stmt)
	}
	return list, nil
}

func (p *Parser) parseAssignment() (Statement, error) {
	var (
		ass Assignment
		err error
	)
	switch {
	case p.Is(Ident):
		ass.Field, err = p.ParseIdent()
		if err != nil {
			return nil, err
		}
	case p.Is(Lparen):
		p.Next()
		var list List
		for !p.Done() && !p.Is(Rparen) {
			stmt, err := p.ParseIdent()
			if err != nil {
				return nil, err
			}
			list.Values = append(list.Values, stmt)
			if err = p.EnsureEnd("update", Comma, Rparen); err != nil {
				return nil, err
			}
		}
		if !p.Is(Rparen) {
			return nil, err
		}
		p.Next()
		ass.Field = list
	default:
		return nil, p.Unexpected("update")
	}
	if !p.Is(Eq) {
		return nil, p.Unexpected("update")
	}
	p.Next()
	if p.Is(Lparen) {
		p.Next()
		var list List
		for !p.Done() && !p.Is(Rparen) {
			expr, err := p.parseExpression(powLowest, p.tokCheck(Comma, Rparen))
			if err != nil {
				return nil, err
			}
			if err = p.EnsureEnd("update", Comma, Rparen); err != nil {
				return nil, err
			}
			list.Values = append(list.Values, expr)
		}
		if !p.Is(Rparen) {
			return nil, p.Unexpected("update")
		}
		p.Next()
	} else {
		ass.Value, err = p.parseExpression(powLowest, p.tokCheck(Comma, Keyword))
		if err != nil {
			return nil, p.Unexpected("update")
		}
	}
	return ass, nil
}

func (p *Parser) ParseInsert() (Statement, error) {
	p.Next()
	var (
		stmt InsertStatement
		err  error
	)
	stmt.Table, err = p.ParseIdent()
	if err != nil {
		return nil, err
	}

	stmt.Columns, err = p.parseColumnsList()
	if err = wrapError("insert", err); err != nil {
		return nil, err
	}

	switch {
	case p.IsKeyword("SELECT"):
		stmt.Values, err = p.ParseSelect()
	case p.IsKeyword("VALUES"):
		p.Next()
		var all List
		for !p.Done() && !p.IsKeyword("RETURNING") && !p.IsKeyword("ON") && !p.Is(EOL) {
			if !p.Is(Lparen) {
				return nil, p.Unexpected("values")
			}
			p.Next()

			list, err := p.parseListValues()
			if err = wrapError("insert", err); err != nil {
				return nil, err
			}
			all.Values = append(all.Values, list)

			switch {
			case p.Is(Comma):
				p.Next()
			case p.Is(EOL):
			case p.Is(Keyword):
			default:
				return nil, p.Unexpected("values")
			}
		}
		stmt.Values = all
	default:
		return nil, p.Unexpected("values")
	}
	if err = wrapError("insert", err); err != nil {
		return nil, err
	}
	if stmt.Upsert, err = p.ParseUpsert(); err != nil {
		return nil, err
	}
	stmt.Return, err = p.ParseReturning()
	return stmt, wrapError("insert", err)
}

func (p *Parser) ParseUpsert() (Statement, error) {
	if !p.IsKeyword("ON") {
		return nil, nil
	}
	p.Next()
	if !p.IsKeyword("CONFLICT") {
		return nil, p.Unexpected("upsert")
	}
	p.Next()

	var (
		stmt UpsertStatement
		err  error
	)

	if !p.IsKeyword("DO") {
		stmt.Columns, err = p.parseColumnsList()
		if err != nil {
			return nil, err
		}
	}
	if !p.IsKeyword("DO") {
		return nil, p.Unexpected("upsert")
	}
	p.Next()
	if p.IsKeyword("NOTHING") {
		p.Next()
		return stmt, nil
	}
	if !p.IsKeyword("UPDATE") {
		return nil, p.Unexpected("upsert")
	}
	p.Next()
	if !p.IsKeyword("SET") {
		return nil, p.Unexpected("upsert")
	}
	p.Next()
	if stmt.List, err = p.ParseUpsertList(); err != nil {
		return nil, err
	}
	stmt.Where, err = p.ParseWhere()
	return stmt, wrapError("upsert", err)
}

func (p *Parser) ParseUpsertList() ([]Statement, error) {
	var list []Statement
	for !p.Done() && !p.Is(EOL) && !p.IsKeyword("WHERE") && !p.IsKeyword("RETURNING") {
		stmt, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}
		if p.Is(EOL) {
			break
		}
		if err := p.EnsureEnd("update", Comma, Keyword); err != nil {
			return nil, err
		}
		list = append(list, stmt)
	}
	return list, nil
}

func (p *Parser) parseListValues() (Statement, error) {
	var list List
	for !p.Done() && !p.Is(Rparen) {
		expr, err := p.parseExpression(powLowest, p.tokCheck(EOL, Rparen))
		if err = wrapError("values", err); err != nil {
			return nil, err
		}
		if err := p.EnsureEnd("values", Comma, Rparen); err != nil {
			return nil, err
		}
		list.Values = append(list.Values, expr)
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("values")
	}
	p.Next()
	return list, nil
}

func (p *Parser) parseCase() (Statement, error) {
	p.Next()
	var (
		stmt CaseStatement
		err  error
	)
	if !p.IsKeyword("WHEN") {
		stmt.Cdt, err = p.parseExpression(powLowest, p.kwCheck("WHEN"))
		if err = wrapError("case", err); err != nil {
			return nil, err
		}
	}
	for p.IsKeyword("WHEN") {
		var when WhenStatement
		p.Next()
		when.Cdt, err = p.parseExpression(powLowest, p.kwCheck("THEN"))
		if err = wrapError("when", err); err != nil {
			return nil, err
		}
		p.Next()
		when.Body, err = p.parseExpression(powLowest, p.kwCheck("WHEN", "ELSE", "END"))
		if err = wrapError("then", err); err != nil {
			return nil, err
		}
		stmt.Body = append(stmt.Body, when)
	}
	if p.IsKeyword("ELSE") {
		p.Next()
		stmt.Else, err = p.parseExpression(powLowest, p.kwCheck("END"))
		if err = wrapError("else", err); err != nil {
			return nil, err
		}
	}
	if !p.IsKeyword("END") {
		return nil, p.Unexpected("case")
	}
	p.Next()
	return p.ParseAlias(stmt)
}

func (p *Parser) parseValues() (Statement, error) {
	p.Next()
	var (
		stmt ValuesStatement
		err  error
	)
	for !p.Done() && !p.Is(EOL) {
		expr, err := p.parseExpression(powLowest, p.tokCheck(EOL, Comma))
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
	var (
		list []Statement
		done = func() bool {
			return p.Is(Comma) || p.IsKeyword("FROM")
		}
	)
	for !p.Done() && !p.IsKeyword("FROM") {
		stmt, err := p.parseExpression(powLowest, done)
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

	list, err := p.ParseStatementList("FROM", p.ParseAlias)
	if err != nil {
		return nil, err
	}
	if p.Is(EOL) {
		return list, nil
	}
	for !p.Done() && p.curr.isJoin() {
		j := Join{
			Type: p.curr.Literal,
		}
		p.Next()
		switch {
		case p.Is(Ident):
			j.Table, err = p.ParseIdent()
		case p.Is(Lparen):
			p.Next()
			j.Table, err = p.ParseSelect()
			if err != nil {
				break
			}
			if !p.Is(Rparen) {
				err = p.Unexpected("join")
				break
			}
			p.Next()
			j.Table, err = p.ParseAlias(j.Table)
			if err != nil {
				return nil, err
			}
		default:
			return nil, p.Unexpected("join")
		}
		if err = wrapError("join", err); err != nil {
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
	p.unregisterInfix("AS", Keyword)
	defer p.registerInfix("AS", Keyword, p.parseKeywordExpr)

	done := p.kwCheck("WHERE", "GROUP BY", "HAVING", "ORDER BY", "LIMIT", "UNION", "INTERSECT", "EXCEPT")

	return p.parseExpression(powLowest, func() bool {
		return p.Is(EOL) || done()
	})
}

func (p *Parser) ParseJoinUsing() (Statement, error) {
	p.Next()
	if !p.Is(Lparen) {
		return nil, p.Unexpected("using")
	}
	p.Next()
	p.unregisterInfix("AS", Keyword)
	defer p.registerInfix("AS", Keyword, p.parseKeywordExpr)

	var list List
	for !p.Done() && !p.Is(Rparen) {
		stmt, err := p.parseExpression(powLowest, p.tokCheck(Comma, Rparen))
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
	p.unregisterInfix("AS", Keyword)
	defer p.registerInfix("AS", Keyword, p.parseKeywordExpr)

	done := p.kwCheck("GROUP BY", "HAVING", "ORDER BY", "LIMIT", "UNION", "INTERSECT", "EXCEPT")

	return p.parseExpression(powLowest, func() bool {
		return p.Is(EOL) || done()
	})
}

func (p *Parser) ParseGroupBy() ([]Statement, error) {
	if !p.IsKeyword("GROUP BY") {
		return nil, nil
	}
	p.Next()
	return p.ParseStatementList("group by", nil)
}

func (p *Parser) ParseHaving() (Statement, error) {
	if !p.IsKeyword("HAVING") {
		return nil, nil
	}
	p.Next()
	p.unregisterInfix("AS", Keyword)
	defer p.registerInfix("AS", Keyword, p.parseKeywordExpr)

	done := p.kwCheck("ORDER BY", "LIMIT", "UNION", "INTERSECT", "EXCEPT")

	return p.parseExpression(powLowest, func() bool {
		return p.Is(EOL) || done()
	})
}

func (p *Parser) ParseOrderBy() ([]Statement, error) {
	if !p.IsKeyword("ORDER BY") {
		return nil, nil
	}
	p.Next()
	do := func(stmt Statement) (Statement, error) {
		order := Order{
			Statement: stmt,
		}
		if p.IsKeyword("ASC") || p.IsKeyword("DESC") {
			order.Orient = p.curr.Literal
			p.Next()
		}
		if p.IsKeyword("NULLS") {
			p.Next()
			if !p.IsKeyword("FIRST") && !p.IsKeyword("LAST") {
				return nil, p.Unexpected("order by")
			}
			order.Nulls = p.curr.Literal
			p.Next()
		}
		return order, nil
	}
	return p.ParseStatementList("order by", do)
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

func (p *Parser) ParseReturning() (Statement, error) {
	if !p.IsKeyword("RETURNING") {
		return nil, nil
	}
	p.Next()
	if p.Is(Star) {
		stmt := Value{
			Literal: "*",
		}
		p.Next()
		if !p.Is(EOL) {
			return nil, p.Unexpected("returning")
		}
		return stmt, nil
	}
	var list List
	for !p.Done() && !p.Is(EOL) {
		stmt, err := p.parseExpression(powLowest, p.tokCheck(EOL, Comma))
		if err != nil {
			return nil, err
		}
		list.Values = append(list.Values, stmt)
		if err = p.EnsureEnd("returning", Comma, EOL); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (p *Parser) registerPrefix(literal string, kind rune, fn prefixFunc) {
	p.prefix[symbolFor(kind, literal)] = fn
}

func (p *Parser) unregisterPrefix(literal string, kind rune) {
	delete(p.prefix, symbolFor(kind, literal))
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
		return nil, p.Unexpected("prefix")
	}
	return fn()
}

func (p *Parser) getInfixExpr(left Statement, end func() bool) (Statement, error) {
	fn, ok := p.infix[p.curr.asSymbol()]
	if !ok {
		return nil, p.Unexpected("infix")
	}
	return fn(left, end)
}

func (p *Parser) parseExpression(power int, end func() bool) (Statement, error) {
	left, err := p.getPrefixExpr()
	if err != nil {
		return nil, err
	}
	for !p.Is(EOL) && !p.Done() && !end() && power < p.currBinding() {
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
	var (
		pow = p.currBinding()
		err error
		ok  bool
	)
	stmt.Op, ok = operandMapping[p.curr.Type]
	if !ok {
		return nil, p.Unexpected("operand")
	}
	p.Next()

	stmt.Right, err = p.parseExpression(pow, end)
	return stmt, wrapError("infix", err)
}

func (p *Parser) parseKeywordExpr(left Statement, end func() bool) (Statement, error) {
	stmt := Binary{
		Left: left,
		Op:   p.curr.Literal,
	}
	var (
		pow = p.currBinding()
		err error
	)
	p.Next()
	stmt.Right, err = p.parseExpression(pow, end)
	return stmt, wrapError("infix", err)
}

func (p *Parser) parseCallExpr(left Statement, _ func() bool) (Statement, error) {
	p.Next()
	stmt := Call{
		Ident: left,
	}
	for !p.Done() && !p.Is(Rparen) {
		arg, err := p.parseExpression(powLowest, p.tokCheck(Comma, Rparen))
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
	return p.ParseAlias(stmt)
}

func (p *Parser) parseUnary() (Statement, error) {
	var (
		stmt Statement
		err  error
	)
	switch {
	case p.Is(Minus):
		p.Next()
		stmt, err = p.parseExpression(powLowest, nil)
		if err = wrapError("reverse", err); err != nil {
			return nil, err
		}
		stmt = Unary{
			Right: stmt,
			Op:    "-",
		}
	case p.IsKeyword("NOT"):
		p.Next()
		stmt, err = p.parseExpression(powLowest, nil)
		if err = wrapError("not", err); err != nil {
			return nil, err
		}
		stmt = Unary{
			Right: stmt,
			Op:    "NOT",
		}
	case p.IsKeyword("CASE"):
		stmt, err = p.parseCase()
	case p.IsKeyword("NULL") || p.IsKeyword("DEFAULT"):
		stmt = Value{
			Literal: p.curr.Literal,
		}
		p.Next()
	default:
		err = p.Unexpected("unary")
	}
	return stmt, nil
}

func (p *Parser) ParseIdent() (Statement, error) {
	var name Name
	if p.peekIs(Dot) {
		name.Prefix = p.curr.Literal
		p.Next()
		p.Next()
	}
	if !p.Is(Ident) && !p.Is(Star) {
		return nil, p.Unexpected("identifier")
	}
	name.Ident = p.curr.Literal
	if p.Is(Star) {
		name.Ident = "*"
	}
	p.Next()
	return p.ParseAlias(name)
}

func (p *Parser) ParseLiteral() (Statement, error) {
	stmt := Value{
		Literal: p.curr.Literal,
	}
	p.Next()
	return stmt, nil
}

func (p *Parser) parseGroupExpr() (Statement, error) {
	p.Next()
	if p.IsKeyword("SELECT") {
		stmt, err := p.ParseSelect()
		if err != nil {
			return nil, err
		}
		if !p.Is(Rparen) {
			return nil, p.Unexpected("group")
		}
		p.Next()
		return p.ParseAlias(stmt)
	}
	stmt, err := p.parseExpression(powLowest, p.tokCheck(Rparen))
	if err = wrapError("group", err); err != nil {
		return nil, err
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("group")
	}
	p.Next()
	return stmt, nil
}

func (p *Parser) parseColumnsList() ([]string, error) {
	if !p.Is(Lparen) {
		return nil, nil
	}
	p.Next()

	var (
		list []string
		err  error
	)

	for !p.Done() && !p.Is(Rparen) {
		if !p.curr.isValue() {
			return nil, p.Unexpected("columns")
		}
		list = append(list, p.curr.Literal)
		p.Next()
		if err := p.EnsureEnd("columns", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("columns")
	}
	p.Next()

	return list, err
}

func (p *Parser) ParseStatementList(ctx string, fn func(Statement) (Statement, error)) ([]Statement, error) {
	var (
		list []Statement
		err  error
	)
	for !p.Done() && !p.Is(Keyword) && !p.Is(EOL) && !p.Is(Rparen) {
		var (
			name Name
			stmt Statement
		)
		switch {
		case p.Is(Lparen):
			p.Next()
			stmt, err = p.ParseSelect()
			if err != nil {
				return nil, err
			}
			if !p.Is(Rparen) {
				return nil, p.Unexpected("list")
			}
			p.Next()
		case p.Is(Ident):
			if p.Is(Ident) && p.peekIs(Dot) {
				name.Prefix = p.curr.Literal
				p.Next()
				p.Next()
			}
			name.Ident = p.curr.Literal
			stmt = name
			p.Next()
		default:
			return nil, p.Unexpected("list")
		}
		if fn != nil {
			if stmt, err = fn(stmt); err != nil {
				return nil, err
			}
		}
		list = append(list, stmt)

		switch {
		case p.Is(Comma):
			p.Next()
			if p.Is(Keyword) || p.Is(EOL) || p.Is(Rparen) || p.Done() {
				return nil, p.Unexpected(ctx)
			}
		case p.Is(Keyword):
		case p.Is(EOL):
		case p.Is(Rparen):
		default:
			return nil, p.Unexpected(ctx)
		}
	}
	return list, nil
}

func (p *Parser) ParseAlias(stmt Statement) (Statement, error) {
	mandatory := p.IsKeyword("AS")
	if mandatory {
		p.Next()
	}
	switch p.curr.Type {
	case Ident, Literal, Number:
		stmt = Alias{
			Statement: stmt,
			Alias:     p.curr.Literal,
		}
		p.Next()
	default:
		if mandatory {
			return nil, p.Unexpected("alias")
		}
	}
	return stmt, nil
}

func (p *Parser) IsKeyword(kw string) bool {
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

func (p *Parser) Unexpected(ctx string) error {
	return p.UnexpectedDialect(ctx, "lang")
}

func (p *Parser) UnexpectedDialect(ctx, dialect string) error {
	return wrapErrorWithDialect(dialect, ctx, unexpected(p.curr))
}

func (p *Parser) EnsureEnd(ctx string, sep, end rune) error {
	switch {
	case p.Is(sep):
		p.Next()
		if p.Is(end) {
			return p.Unexpected(ctx)
		}
	case p.Is(end):
	default:
		return p.Unexpected(ctx)
	}
	return nil
}

func (p *Parser) tokCheck(kind ...rune) func() bool {
	sort.Slice(kind, func(i, j int) bool {
		return kind[i] < kind[j]
	})
	return func() bool {
		i := sort.Search(len(kind), func(i int) bool {
			return p.Is(kind[i])
		})
		return i < len(kind) && kind[i] == p.curr.Type
	}
}

func (p *Parser) kwCheck(str ...string) func() bool {
	sort.Strings(str)
	return func() bool {
		if !p.Is(Keyword) {
			return false
		}
		if len(str) == 1 {
			return str[0] == p.curr.Literal
		}
		i := sort.SearchStrings(str, p.curr.Literal)
		return i < len(str) && str[i] == p.curr.Literal
	}
}

func (p *Parser) Done() bool {
	if p.frame.Done() {
		if n := len(p.stack); n > 0 {
			p.frame = p.stack[n-1]
			p.stack = p.stack[:n-1]
		}
	}
	return p.frame.Done()
}

type prefixFunc func() (Statement, error)

type infixFunc func(Statement, func() bool) (Statement, error)

var operandMapping = map[rune]string{
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

type frame struct {
	scan *Scanner
	set  KeywordSet

	base string
	curr Token
	peek Token
}

func createFrame(r io.Reader, set KeywordSet) (*frame, error) {
	scan, err := Scan(r, set)
	if err != nil {
		return nil, err
	}
	f := frame{
		scan: scan,
		set:  set,
	}
	if n, ok := r.(interface{ Name() string }); ok {
		f.base = filepath.Dir(n.Name())
	}
	f.Next()
	f.Next()
	return &f, nil
}

func (f *frame) Curr() Token {
	return f.curr
}

func (f *frame) Peek() Token {
	return f.peek
}

func (f *frame) GetCurrLiteral() string {
	return f.curr.Literal
}

func (f *frame) GetPeekLiteral() string {
	return f.peek.Literal
}

func (f *frame) GetCurrType() rune {
	return f.curr.Type
}

func (f *frame) GetPeekType() rune {
	return f.peek.Type
}

func (f *frame) Next() {
	f.curr = f.peek
	f.peek = f.scan.Scan()
}

func (f *frame) Done() bool {
	return f.Is(EOF)
}

func (f *frame) Is(kind rune) bool {
	return f.curr.Type == kind
}

func (f *frame) peekIs(kind rune) bool {
	return f.peek.Type == kind
}
