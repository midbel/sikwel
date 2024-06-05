package lang

import (
	"fmt"
)

func (p *Parser) ParseMerge() (Statement, error) {
	p.Next()
	var (
		stmt MergeStatement
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
			parseAction func(Statement) (Statement, error)
			cdt         Statement
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

func (p *Parser) parseMergeMatched(cdt Statement) (Statement, error) {
	var (
		stmt Statement
		err  error
	)
	switch {
	case p.IsKeyword("DELETE"):
		p.Next()
		stmt = MatchStatement{
			Condition: cdt,
			Statement: DeleteStatement{},
		}
	case p.IsKeyword("UPDATE"):
		p.Next()
		if !p.IsKeyword("SET") {
			return nil, p.Unexpected("matched")
		}
		p.Next()
		var upd UpdateStatement
		for !p.QueryEnds() && !p.IsKeyword("WHEN MATCHED") && !p.IsKeyword("WHEN NOT MATCHED") {
			s, err := p.parseAssignment()
			if err != nil {
				return nil, err
			}
			upd.List = append(upd.List, s)
		}
		stmt = MatchStatement{
			Condition: cdt,
			Statement: upd,
		}
	default:
		err = p.Unexpected("matched")
	}
	return stmt, err
}

func (p *Parser) parseMergeNotMatched(cdt Statement) (Statement, error) {
	if !p.IsKeyword("INSERT") {
		return nil, p.Unexpected("match")
	}
	p.Next()
	var (
		ins InsertStatement
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
	stmt := MatchStatement{
		Condition: cdt,
		Statement: ins,
	}
	return stmt, nil
}

func (p *Parser) ParseDelete() (Statement, error) {
	p.Next()
	var (
		stmt DeleteStatement
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

func (p *Parser) ParseTruncate() (Statement, error) {
	p.Next()
	var stmt TruncateStatement
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
		stmt.Identity = RestartIdentity
		if p.IsKeyword("CONTINUE IDENTITY") {
			stmt.Identity = ContinueIdentity
		}
		p.Next()
	}
	if p.IsKeyword("RESTRICT") || p.IsKeyword("CASCADE") {
		stmt.Cascade = p.IsKeyword("CASCADE")
		p.Next()
	}
	return stmt, nil
}

func (p *Parser) ParseReturning() (Statement, error) {
	if !p.IsKeyword("RETURNING") {
		return nil, nil
	}
	p.Next()
	if p.Is(Star) {
		var stmt Name
		p.Next()
		if !p.QueryEnds() {
			return nil, p.Unexpected("returning")
		}
		return stmt, nil
	}
	var list List
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
		ass.Field, err = p.ParseIdentifier()
		if err != nil {
			return nil, err
		}
	case p.Is(Lparen):
		p.Next()
		var list List
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
		var list List
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

func (p *Parser) ParseInsert() (Statement, error) {
	p.Next()
	var (
		stmt InsertStatement
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

func (p *Parser) ParseUpsert() (Statement, error) {
	if !p.IsKeyword("ON CONFLICT") {
		return nil, nil
	}
	p.Next()

	var (
		stmt Upsert
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

func (w *Writer) FormatMerge(stmt MergeStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	w.WriteKeyword("INTO")
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Target, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("USING")
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Source, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("ON")
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Join, false); err != nil {
		return err
	}
	for _, a := range stmt.Actions {
		m, ok := a.(MatchStatement)
		if !ok {
			return w.CanNotUse("merge", a)
		}
		w.WriteNL()
		w.WritePrefix()
		if err := w.FormatMatch(m); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatMatch(stmt MatchStatement) error {
	w.WriteKeyword("WHEN")
	w.WriteBlank()
	switch stmt.Statement.(type) {
	case DeleteStatement:
		w.WriteKeyword("MATCHED")
	case UpdateStatement:
		w.WriteKeyword("MATCHED")
	case InsertStatement:
		w.WriteKeyword("NOT MATCHED")
	default:
		return w.CanNotUse("merge", stmt.Statement)
	}
	if stmt.Condition != nil {
		w.WriteBlank()
		w.WriteKeyword("AND")
		w.WriteBlank()
		if err := w.FormatExpr(stmt.Condition, false); err != nil {
			return err
		}
	}
	w.WriteBlank()
	w.WriteKeyword("THEN")
	w.WriteNL()
	w.Enter()
	defer w.Leave()

	w.WritePrefix()

	switch stmt := stmt.Statement.(type) {
	case DeleteStatement:
		w.WriteKeyword("DELETE")
	case UpdateStatement:
		w.WriteKeyword("UPDATE")
		w.WriteBlank()
		w.WriteKeyword("SET")
		w.WriteBlank()

		compact := w.Compact
		w.Compact = true
		defer func() {
			w.Compact = compact
		}()
		if err := w.FormatAssignment(stmt.List); err != nil {
			return err
		}
	case InsertStatement:
		w.WriteKeyword("INSERT")
		w.WriteBlank()
		if len(stmt.Columns) > 0 {
			w.WriteString("(")
			for i := range stmt.Columns {
				if i > 0 {
					w.WriteString(",")
					w.WriteBlank()
				}
				w.WriteString(stmt.Columns[i])
			}
			w.WriteString(")")
			w.WriteBlank()
		}
		values, ok := stmt.Values.(ValuesStatement)
		if !ok {
			return w.CanNotUse("merge", stmt.Values)
		}
		compact := w.Compact
		w.Compact = true
		defer func() {
			w.Compact = compact
		}()
		if err := w.FormatValues(values); err != nil {
			return err
		}
	default:
		return w.CanNotUse("merge", stmt)
	}
	return nil
}

func (w *Writer) FormatTruncate(stmt TruncateStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	if len(stmt.Tables) == 0 {
		w.WriteString("*")
		return nil
	}
	for i := range stmt.Tables {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		w.WriteString(stmt.Tables[i])
	}
	return nil
}

func (w *Writer) FormatDelete(stmt DeleteStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	w.WriteString(stmt.Table)
	if stmt.Where != nil {
		w.WriteNL()
		if err := w.FormatWhere(stmt.Where); err != nil {
			return err
		}
	}
	if stmt.Return != nil {
		w.WriteNL()
		if err := w.FormatReturning(stmt.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatUpdate(stmt UpdateStatement) error {
	kw, _ := stmt.Keyword()
	return w.FormatUpdateWithKeyword(kw, stmt)
}

func (w *Writer) FormatUpdateWithKeyword(kw string, stmt UpdateStatement) error {
	w.Enter()
	defer w.Leave()

	w.WriteStatement(kw)
	w.WriteBlank()

	switch stmt := stmt.Table.(type) {
	case Name:
		w.FormatName(stmt)
	case Alias:
		if err := w.FormatAlias(stmt); err != nil {
			return err
		}
	default:
		return w.CanNotUse("update", stmt)
	}
	w.WriteBlank()
	w.WriteKeyword("SET")
	w.WriteNL()

	if err := w.FormatAssignment(stmt.List); err != nil {
		return err
	}

	if len(stmt.Tables) > 0 {
		w.WriteNL()
		if err := w.FormatFrom(stmt.Tables); err != nil {
			return err
		}
	}
	if stmt.Where != nil {
		w.WriteNL()
		if err := w.FormatWhere(stmt.Where); err != nil {
			return err
		}
	}
	if stmt.Return != nil {
		w.WriteNL()
		if err := w.FormatReturning(stmt.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatInsert(stmt InsertStatement) error {
	kw, _ := stmt.Keyword()
	return w.FormatInsertWithKeyword(kw, stmt)
}

func (w *Writer) FormatInsertWithKeyword(kw string, stmt InsertStatement) error {
	w.Enter()
	defer w.Leave()

	w.WriteStatement(kw)
	w.WriteBlank()

	if err := w.FormatExpr(stmt.Table, false); err != nil {
		return err
	}
	if len(stmt.Columns) > 0 {
		w.WriteBlank()
		w.WriteString("(")
		for i, c := range stmt.Columns {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteString(c)
		}
		w.WriteString(")")
	}
	w.WriteBlank()
	if err := w.FormatInsertValues(stmt.Values); err != nil {
		return err
	}
	if stmt.Upsert != nil {
		w.WriteNL()
		if err := w.FormatUpsert(stmt.Upsert); err != nil {
			return err
		}
	}
	if stmt.Return != nil {
		w.WriteNL()
		if err := w.FormatReturning(stmt.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatInsertValues(values Statement) error {
	if values == nil {
		return nil
	}
	var err error
	switch stmt := values.(type) {
	case ValuesStatement:
		err = w.FormatValues(stmt)
	case SelectStatement:
		w.WriteNL()
		err = w.FormatSelect(stmt)
	default:
		err = fmt.Errorf("values: unexpected statement type(%T)", values)
	}
	return err
}

func (w *Writer) FormatUpsert(stmt Statement) error {
	if stmt == nil {
		return nil
	}
	upsert, ok := stmt.(Upsert)
	if !ok {
		return w.CanNotUse("insert(upsert)", stmt)
	}
	w.WriteStatement("ON CONFLICT")
	w.WriteBlank()

	if len(upsert.Columns) > 0 {
		w.WriteString("(")
		for i, s := range upsert.Columns {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteString(s)
		}
		w.WriteString(")")
	}
	w.WriteBlank()
	if len(upsert.List) == 0 {
		w.WriteKeyword("DO NOTHING")
		return nil
	}
	w.WriteKeyword("UPDATE SET")
	w.WriteNL()
	if err := w.FormatAssignment(upsert.List); err != nil {
		return err
	}
	return w.FormatWhere(upsert.Where)
}

func (w *Writer) FormatAssignment(list []Statement) error {
	w.Enter()
	defer w.Leave()

	var err error
	for i, s := range list {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		ass, ok := s.(Assignment)
		if !ok {
			return w.CanNotUse("assignment", s)
		}
		w.WritePrefix()
		switch field := ass.Field.(type) {
		case Name:
			w.FormatName(field)
		case List:
			err = w.formatList(field)
		default:
			return w.CanNotUse("assignment", s)
		}
		if err != nil {
			return err
		}
		w.WriteString("=")
		switch value := ass.Value.(type) {
		case List:
			err = w.formatList(value)
		default:
			err = w.FormatExpr(value, false)
		}
		if err != nil {
			return err
		}
	}
	return err
}

func (w *Writer) FormatReturning(stmt Statement) error {
	if stmt == nil {
		return nil
	}
	w.WriteStatement("RETURNING")
	w.WriteBlank()

	list, ok := stmt.(List)
	if !ok {
		return w.FormatExpr(stmt, false)
	}
	return w.formatStmtSlice(list.Values)
}
