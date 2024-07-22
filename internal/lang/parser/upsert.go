package parser

import (
	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/token"
)

func (p *Parser) ParseMerge() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.MergeStatement
		err  error
	)
	if stmt.Target, err = p.ParseIdent(); err != nil {
		return nil, err
	}
	if !p.IsKeyword("USING") {
		return nil, p.Unexpected("merge", keywordExpected("USING"))
	}
	p.Next()
	switch {
	case p.Is(token.Lparen):
	case p.Is(token.Ident):
		stmt.Source, err = p.ParseIdent()
	default:
		err = p.Unexpected("merge", defaultReason)
	}
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("ON") {
		return nil, p.Unexpected("merge", keywordExpected("ON"))
	}
	p.Next()
	if stmt.Join, err = p.StartExpression(); err != nil {
		return nil, err
	}
	for !p.QueryEnds() && !p.Done() {
		var (
			parseAction func(ast.Statement) (ast.Statement, error)
			cdt         ast.Statement
			err         error
		)
		switch {
		case p.IsKeyword("WHEN MATCHED"):
			parseAction = p.parseMergeMatched
		case p.IsKeyword("WHEN NOT MATCHED"):
			parseAction = p.parseMergeNotMatched
		default:
			return nil, p.Unexpected("merge", defaultReason)
		}
		p.Next()
		if p.IsKeyword("AND") {
			p.Next()
			if cdt, err = p.StartExpression(); err != nil {
				return nil, err
			}
		}
		if !p.IsKeyword("THEN") {
			return nil, p.Unexpected("merge", keywordExpected("THEN"))
		}
		p.Next()
		act, err := parseAction(cdt)
		if err != nil {
			return nil, err
		}
		stmt.Actions = append(stmt.Actions, act)
	}
	return stmt, nil
}

func (p *Parser) parseMergeMatched(cdt ast.Statement) (ast.Statement, error) {
	var (
		stmt ast.Statement
		err  error
	)
	switch {
	case p.IsKeyword("DELETE"):
		p.Next()
		stmt = ast.MatchStatement{
			Condition: cdt,
			Statement: ast.DeleteStatement{},
		}
	case p.IsKeyword("UPDATE"):
		p.Next()
		if !p.IsKeyword("SET") {
			return nil, p.Unexpected("matched", keywordExpected("SET"))
		}
		p.Next()
		var upd ast.UpdateStatement
		for !p.QueryEnds() && !p.IsKeyword("WHEN MATCHED") && !p.IsKeyword("WHEN NOT MATCHED") {
			s, err := p.parseAssignment()
			if err != nil {
				return nil, err
			}
			upd.List = append(upd.List, s)
		}
		stmt = ast.MatchStatement{
			Condition: cdt,
			Statement: upd,
		}
	default:
		err = p.Unexpected("matched", defaultReason)
	}
	return stmt, err
}

func (p *Parser) parseMergeNotMatched(cdt ast.Statement) (ast.Statement, error) {
	if !p.IsKeyword("INSERT") {
		return nil, p.Unexpected("match", keywordExpected("INSERT"))
	}
	p.Next()
	var (
		ins ast.InsertStatement
		err error
	)
	if p.Is(token.Lparen) {
		ins.Columns, err = p.parseColumnsList()
		if err != nil {
			return nil, err
		}
	}
	if !p.IsKeyword("VALUES") {
		return nil, p.Unexpected("not matched", keywordExpected("VALUES"))
	}
	ins.Values, err = p.ParseValues()
	if err != nil {
		return nil, err
	}
	stmt := ast.MatchStatement{
		Condition: cdt,
		Statement: ins,
	}
	return stmt, nil
}

func (p *Parser) ParseDelete() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.DeleteStatement
		err  error
	)
	if !p.Is(token.Ident) {
		return nil, p.Unexpected("delete", identExpected)
	}
	stmt.Table = p.GetCurrLiteral()
	p.Next()

	if stmt.Where, err = p.ParseWhere(); err != nil {
		return nil, err
	}
	if stmt.Return, err = p.ParseReturning(); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *Parser) ParseTruncate() (ast.Statement, error) {
	p.Next()
	var stmt ast.TruncateStatement
	if p.Is(token.Star) {
		p.Next()
		return stmt, nil
	} else {
		for !p.Is(token.EOL) && !p.Done() && !p.Is(token.Keyword) {
			if !p.Is(token.Ident) {
				return nil, p.Unexpected("truncate", identExpected)
			}
			stmt.Tables = append(stmt.Tables, p.GetCurrLiteral())
			p.Next()
			switch {
			case p.Is(token.EOL) || p.Is(token.Keyword):
			case p.Is(token.Comma):
				p.Next()
			default:
				return nil, p.Unexpected("truncate", defaultReason)
			}
		}
	}
	if p.IsKeyword("RESTART IDENTITY") || p.IsKeyword("CONTINUE IDENTITY") {
		stmt.Identity = ast.RestartIdentity
		if p.IsKeyword("CONTINUE IDENTITY") {
			stmt.Identity = ast.ContinueIdentity
		}
		p.Next()
	}
	if p.IsKeyword("RESTRICT") {
		stmt.Cascade = ast.Restrict
	} else if p.IsKeyword("CASCADE") {
		stmt.Cascade = ast.Restrict
	}
	if stmt.Cascade != 0 {
		p.Next()
	}
	return stmt, nil
}

func (p *Parser) ParseReturning() (ast.Statement, error) {
	if !p.IsKeyword("RETURNING") {
		return nil, nil
	}
	p.Next()
	if p.Is(token.Star) {
		var stmt ast.Name
		p.Next()
		if !p.QueryEnds() {
			return nil, p.Unexpected("returning", missingEol)
		}
		return stmt, nil
	}
	var list ast.List
	for !p.Done() && !p.Is(token.EOL) {
		stmt, err := p.StartExpression()
		if err != nil {
			return nil, err
		}
		list.Values = append(list.Values, stmt)
		if err = p.EnsureEnd("returning", token.Comma, token.EOL); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (p *Parser) ParseUpdate() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.UpdateStatement
		err  error
	)
	stmt.Table, err = p.ParseIdent()
	if err != nil {
		return nil, err
	}

	if !p.IsKeyword("SET") {
		return nil, p.Unexpected("update", keywordExpected("SET"))
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
	return nil, err
}

func (p *Parser) ParseUpdateList() ([]ast.Statement, error) {
	var list []ast.Statement
	for !p.Done() && !p.Is(token.EOL) && !p.IsKeyword("WHERE") && !p.IsKeyword("FROM") && !p.IsKeyword("RETURNING") {
		stmt, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}
		if p.Is(token.EOL) {
			break
		}
		if err := p.EnsureEnd("update", token.Comma, token.Keyword); err != nil {
			return nil, err
		}
		list = append(list, stmt)
	}
	return list, nil
}

func (p *Parser) parseAssignment() (ast.Statement, error) {
	var (
		ass ast.Assignment
		err error
	)
	switch {
	case p.Is(token.Ident):
		ass.Field, err = p.ParseIdentifier()
		if err != nil {
			return nil, err
		}
	case p.Is(token.Lparen):
		p.Next()
		var list ast.List
		for !p.Done() && !p.Is(token.Rparen) {
			stmt, err := p.ParseIdentifier()
			if err != nil {
				return nil, err
			}
			list.Values = append(list.Values, stmt)
			if err = p.EnsureEnd("update", token.Comma, token.Rparen); err != nil {
				return nil, err
			}
		}
		if !p.Is(token.Rparen) {
			return nil, err
		}
		p.Next()
		ass.Field = list
	default:
		return nil, p.Unexpected("update", defaultReason)
	}
	if !p.Is(token.Eq) {
		return nil, p.Unexpected("update", "equal operator expected")
	}
	p.Next()
	if p.Is(token.Lparen) {
		p.Next()
		var list ast.List
		for !p.Done() && !p.Is(token.Rparen) {
			expr, err := p.StartExpression()
			if err != nil {
				return nil, err
			}
			if err = p.EnsureEnd("update", token.Comma, token.Rparen); err != nil {
				return nil, err
			}
			list.Values = append(list.Values, expr)
		}
		if !p.Is(token.Rparen) {
			return nil, p.Unexpected("update", missingCloseParen)
		}
		p.Next()
	} else {
		ass.Value, err = p.StartExpression()
		if err != nil {
			return nil, err
		}
	}
	return ass, nil
}

func (p *Parser) ParseInsert() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.InsertStatement
		err  error
	)
	stmt.Table, err = p.ParseIdentifier()
	if err != nil {
		return nil, err
	}

	stmt.Columns, err = p.parseColumnsList()
	if err != nil {
		return nil, err
	}

	switch {
	case p.IsKeyword("SELECT") || p.IsKeyword("WITH"):
		stmt.Values, err = p.ParseStatement()
	case p.IsKeyword("VALUES"):
		stmt.Values, err = p.ParseValues()
	default:
		return nil, p.Unexpected("insert", defaultReason)
	}
	if err != nil {
		return nil, err
	}
	if stmt.Upsert, err = p.ParseUpsert(); err != nil {
		return nil, err
	}
	stmt.Return, err = p.ParseReturning()
	return stmt, err
}

func (p *Parser) ParseUpsert() (ast.Statement, error) {
	if !p.IsKeyword("ON CONFLICT") {
		return nil, nil
	}
	p.Next()

	var (
		stmt ast.Upsert
		err  error
	)

	if !p.IsKeyword("DO") {
		stmt.Columns, err = p.parseColumnsList()
		if err != nil {
			return nil, err
		}
	}
	if !p.IsKeyword("DO") {
		return nil, p.Unexpected("upsert", keywordExpected("DO"))
	}
	p.Next()
	if p.IsKeyword("NOTHING") {
		p.Next()
		return stmt, nil
	}
	if !p.IsKeyword("UPDATE") {
		return nil, p.Unexpected("upsert", keywordExpected("UPDATE"))
	}
	p.Next()
	if !p.IsKeyword("SET") {
		return nil, p.Unexpected("upsert", keywordExpected("SET"))
	}
	p.Next()
	if stmt.List, err = p.ParseUpsertList(); err != nil {
		return nil, err
	}
	stmt.Where, err = p.ParseWhere()
	return stmt, err
}

func (p *Parser) ParseUpsertList() ([]ast.Statement, error) {
	var list []ast.Statement
	for !p.Done() && !p.Is(token.EOL) && !p.IsKeyword("WHERE") && !p.IsKeyword("RETURNING") {
		stmt, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}
		if p.Is(token.EOL) {
			break
		}
		if err := p.EnsureEnd("update", token.Comma, token.Keyword); err != nil {
			return nil, err
		}
		list = append(list, stmt)
	}
	return list, nil
}
