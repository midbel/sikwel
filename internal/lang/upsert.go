package lang

import (
	"github.com/midbel/sweet/internal/lang/ast"
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
		return nil, p.Unexpected("merge")
	}
	p.Next()
	switch {
	case p.Is(Lparen):
	case p.Is(Ident):
		stmt.Source, err = p.ParseIdent()
	default:
		err = p.Unexpected("merge")
	}
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("ON") {
		return nil, p.Unexpected("merge")
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
			return nil, p.Unexpected("merge")
		}
		p.Next()
		if p.IsKeyword("AND") {
			p.Next()
			if cdt, err = p.StartExpression(); err != nil {
				return nil, err
			}
		}
		if !p.IsKeyword("THEN") {
			return nil, p.Unexpected("merge")
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
			return nil, p.Unexpected("matched")
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
		err = p.Unexpected("matched")
	}
	return stmt, err
}

func (p *Parser) parseMergeNotMatched(cdt ast.Statement) (ast.Statement, error) {
	if !p.IsKeyword("INSERT") {
		return nil, p.Unexpected("match")
	}
	p.Next()
	var (
		ins ast.InsertStatement
		err error
	)
	if p.Is(Lparen) {
		ins.Columns, err = p.parseColumnsList()
		if err != nil {
			return nil, err
		}
	}
	if !p.IsKeyword("VALUES") {
		return nil, p.Unexpected("not matched")
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
	if !p.Is(Ident) {
		return nil, p.Unexpected("delete")
	}
	stmt.Table = p.GetCurrLiteral()
	p.Next()

	if stmt.Where, err = p.ParseWhere(); err != nil {
		return nil, wrapError("delete", err)
	}
	if stmt.Return, err = p.ParseReturning(); err != nil {
		return nil, wrapError("delete", err)
	}
	return stmt, nil
}

func (p *Parser) ParseTruncate() (ast.Statement, error) {
	p.Next()
	var stmt ast.TruncateStatement
	if p.Is(Star) {
		p.Next()
		return stmt, nil
	} else {
		for !p.Is(EOL) && !p.Done() && !p.Is(Keyword) {
			if !p.Is(Ident) {
				return nil, p.Unexpected("truncate")
			}
			stmt.Tables = append(stmt.Tables, p.GetCurrLiteral())
			p.Next()
			switch {
			case p.Is(EOL) || p.Is(Keyword):
			case p.Is(Comma):
				p.Next()
			default:
				return nil, p.Unexpected("truncate")
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
	if p.Is(Star) {
		var stmt ast.Name
		p.Next()
		if !p.QueryEnds() {
			return nil, p.Unexpected("returning")
		}
		return stmt, nil
	}
	var list ast.List
	for !p.Done() && !p.Is(EOL) {
		stmt, err := p.StartExpression()
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

func (p *Parser) ParseUpdateList() ([]ast.Statement, error) {
	var list []ast.Statement
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

func (p *Parser) parseAssignment() (ast.Statement, error) {
	var (
		ass ast.Assignment
		err error
	)
	switch {
	case p.Is(Ident):
		ass.Field, err = p.ParseIdentifier()
		if err != nil {
			return nil, err
		}
	case p.Is(Lparen):
		p.Next()
		var list ast.List
		for !p.Done() && !p.Is(Rparen) {
			stmt, err := p.ParseIdentifier()
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
		var list ast.List
		for !p.Done() && !p.Is(Rparen) {
			expr, err := p.StartExpression()
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
		ass.Value, err = p.StartExpression()
		if err != nil {
			return nil, p.Unexpected("update")
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
	if err = wrapError("insert", err); err != nil {
		return nil, err
	}

	switch {
	case p.IsKeyword("SELECT") || p.IsKeyword("WITH"):
		stmt.Values, err = p.ParseStatement()
	case p.IsKeyword("VALUES"):
		stmt.Values, err = p.ParseValues()
	default:
		return nil, p.Unexpected("insert")
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

func (p *Parser) ParseUpsertList() ([]ast.Statement, error) {
	var list []ast.Statement
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
