package lang

import (
	"strconv"
	"strings"
)

func (p *Parser) ParseLiteral() (Statement, error) {
	stmt := Value{
		Literal: p.GetCurrLiteral(),
	}
	p.Next()
	return stmt, nil
}

func (p *Parser) ParseConstant() (Statement, error) {
	if !p.Is(Keyword) {
		return nil, p.Unexpected("constant")
	}
	switch p.GetCurrLiteral() {
	case "TRUE", "FALSE", "UNKNOWN", "NULL", "DEFAULT":
	default:
		return nil, p.Unexpected("constant")
	}
	return p.ParseLiteral()
}

func (p *Parser) ParseIdentifier() (Statement, error) {
	var name Name
	for p.peekIs(Dot) {
		name.Parts = append(name.Parts, p.GetCurrLiteral())
		p.Next()
		p.Next()
	}
	if !p.Is(Ident) && !p.Is(Star) {
		return nil, p.Unexpected("identifier")
	}
	name.Parts = append(name.Parts, p.GetCurrLiteral())
	p.Next()
	return name, nil
}

func (p *Parser) ParseIdent() (Statement, error) {
	stmt, err := p.ParseIdentifier()
	if err == nil {
		stmt, err = p.ParseAlias(stmt)
	}
	return stmt, nil
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
			Alias:     p.GetCurrLiteral(),
		}
		p.Next()
	default:
		if mandatory {
			return nil, p.Unexpected("alias")
		}
	}
	return stmt, nil
}

func (p *Parser) ParseCase() (Statement, error) {
	p.Next()
	var (
		stmt CaseStatement
		err  error
	)
	if !p.IsKeyword("WHEN") {
		stmt.Cdt, err = p.StartExpression()
		if err = wrapError("case", err); err != nil {
			return nil, err
		}
	}
	for p.IsKeyword("WHEN") {
		var when WhenStatement
		p.Next()
		when.Cdt, err = p.StartExpression()
		if err = wrapError("when", err); err != nil {
			return nil, err
		}
		p.Next()
		when.Body, err = p.StartExpression()
		if err = wrapError("then", err); err != nil {
			return nil, err
		}
		stmt.Body = append(stmt.Body, when)
	}
	if p.IsKeyword("ELSE") {
		p.Next()
		stmt.Else, err = p.StartExpression()
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

func (p *Parser) ParseCast() (Statement, error) {
	p.Next()
	if !p.Is(Lparen) {
		return nil, p.Unexpected("cast")
	}
	p.Next()
	var (
		cast Cast
		err  error
	)
	cast.Ident, err = p.ParseIdentifier()
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("AS") {
		return nil, p.Unexpected("cast")
	}
	p.Next()
	if cast.Type, err = p.ParseType(); err != nil {
		return nil, err
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("cast")
	}
	p.Next()
	return cast, nil
}

func (p *Parser) ParseType() (Type, error) {
	var t Type
	if !p.Is(Ident) {
		return t, p.Unexpected("type")
	}
	t.Name = p.GetCurrLiteral()
	p.Next()
	if p.Is(Lparen) {
		p.Next()
		size, err := strconv.Atoi(p.GetCurrLiteral())
		if err != nil {
			return t, err
		}
		t.Length = size
		p.Next()
		if p.Is(Comma) {
			p.Next()
			size, err = strconv.Atoi(p.GetCurrLiteral())
			if err != nil {
				return t, err
			}
			t.Precision = size
			p.Next()
		}
		if !p.Is(Rparen) {
			return t, p.Unexpected("type")
		}
		p.Next()
	}
	return t, nil
}

func (p *Parser) ParseRow() (Statement, error) {
	p.Next()
	if !p.Is(Lparen) {
		return nil, p.Unexpected("row")
	}
	p.Next()

	p.setDefaultFuncSet()
	defer p.unsetFuncSet()

	var row Row
	for !p.Done() && !p.Is(Rparen) {
		expr, err := p.StartExpression()
		if err != nil {
			return nil, err
		}
		row.Values = append(row.Values, expr)
		if err = p.EnsureEnd("row", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	if !p.Is(Rparen) {
		return nil, p.Unexpected("row")
	}
	p.Next()
	return row, nil
}

func (w *Writer) FormatName(name Name) {
	for i := range name.Parts {
		if i > 0 {
			w.WriteString(".")
		}
		str := name.Parts[i]
		if w.Upperize {
			str = strings.ToUpper(str)
		}
		if w.UseQuote && str != "*" {
			str = w.Quote(str)
			// str = fmt.Sprintf("\"%s\"", str)
		}
		w.WriteString(str)
	}
}

func (w *Writer) FormatAlias(alias Alias) error {
	var err error
	if wrapWithParens(alias.Statement) {
		w.WriteString("(")
		if !w.Compact {
			w.WriteNL()
		}
		if w.getCurrDepth() <= 1 {
			w.Enter()
			defer w.Leave()
		}
		err = w.FormatStatement(alias.Statement)
		if err == nil {
			w.WriteNL()
			w.WritePrefix()
			w.WriteString(")")
		}
	} else {
		err = w.FormatExpr(alias.Statement, false)
	}
	if err != nil {
		return err
	}
	w.WriteBlank()
	if w.UseAs {
		w.WriteKeyword("AS")
		w.WriteBlank()
	}
	str := alias.Alias
	if w.Upperize {
		str = strings.ToUpper(str)
	}
	if w.UseQuote {
		str = w.Quote(str)
	}
	w.WriteString(str)
	return nil
}

func (w *Writer) FormatLiteral(literal string) {
	if literal == "NULL" || literal == "DEFAULT" || literal == "TRUE" || literal == "FALSE" || literal == "*" {
		if w.withColor() {
			w.WriteString(keywordColor)
		}
		w.WriteKeyword(literal)
		if w.withColor() {
			w.WriteString(resetCode)
		}
		return
	}
	if _, err := strconv.Atoi(literal); err == nil {
		if w.withColor() {
			w.WriteString(numberColor)
		}
		w.WriteString(literal)
		if w.withColor() {
			w.WriteString(resetCode)
		}
		return
	}
	if _, err := strconv.ParseFloat(literal, 64); err == nil {
		if w.withColor() {
			w.WriteString(numberColor)
		}
		w.WriteString(literal)
		if w.withColor() {
			w.WriteString(resetCode)
		}
		return
	}
	w.WriteQuoted(literal)
}

func (w *Writer) FormatRow(stmt Row, nl bool) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	w.WriteString("(")
	for i, v := range stmt.Values {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		if nl {
			w.WriteNL()
			w.WritePrefix()
		}
		if err := w.FormatExpr(v, false); err != nil {
			return err
		}
	}
	if nl {
		w.WriteNL()
	}
	w.WriteString(")")
	return nil
}

func (w *Writer) FormatCase(stmt CaseStatement) error {
	w.WriteKeyword("CASE")
	if stmt.Cdt != nil {
		w.WriteBlank()
		w.FormatExpr(stmt.Cdt, false)
	}
	w.WriteBlank()
	w.Enter()
	for _, s := range stmt.Body {
		w.WriteNL()
		if err := w.FormatExpr(s, false); err != nil {
			return err
		}
	}
	if stmt.Else != nil {
		w.WriteNL()
		w.WriteStatement("ELSE")
		w.WriteBlank()
		if err := w.FormatExpr(stmt.Else, false); err != nil {
			return err
		}
	}
	w.Leave()
	w.WriteNL()
	w.WriteStatement("END")
	return nil
}

func (w *Writer) FormatWhen(stmt WhenStatement) error {
	w.WriteStatement("WHEN")
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Cdt, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("THEN")
	w.WriteBlank()
	return w.FormatExpr(stmt.Body, false)
}

func (w *Writer) FormatCast(stmt Cast, _ bool) error {
	w.WriteKeyword("CAST")
	w.WriteString("(")
	if err := w.FormatExpr(stmt.Ident, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("AS")
	w.WriteBlank()
	if err := w.FormatType(stmt.Type); err != nil {
		return err
	}
	w.WriteString(")")
	return nil
}

func (w *Writer) FormatType(dt Type) error {
	w.WriteString(dt.Name)
	if dt.Length <= 0 {
		return nil
	}
	w.WriteString("(")
	w.WriteString(strconv.Itoa(dt.Length))
	if dt.Precision > 0 {
		w.WriteString(",")
		w.WriteBlank()
		w.WriteString(strconv.Itoa(dt.Precision))
	}
	w.WriteString(")")
	return nil
}
