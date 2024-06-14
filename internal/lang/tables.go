package lang

import (
	"github.com/midbel/sweet/internal/lang/ast"
)

type CreateTableParser interface {
	ParseTableName() (ast.Statement, error)
	ParseConstraint(bool) (ast.Statement, error)
	ParseColumnDef(CreateTableParser) (ast.Statement, error)
}

func (p *Parser) ParseDropTable() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.DropTableStatement
		err  error
	)
	if p.IsKeyword("IF EXISTS") {
		stmt.Exists = true
		p.Next()
	}
	for !p.QueryEnds() && !p.Done() {
		n, err := p.ParseIdentifier()
		if err != nil {
			return nil, err
		}
		stmt.Names = append(stmt.Names, n)
		if !p.Is(Comma) {
			break
		}
		p.Next()

	}
	if p.IsKeyword("RESTRICT") {
		stmt.Cascade = ast.Restrict
		p.Next()
	} else if p.IsKeyword("CASCADE") {
		stmt.Cascade = ast.Cascade
		p.Next()
	}
	return stmt, err
}

func (p *Parser) ParseDropView() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.DropViewStatement
		err  error
	)
	if p.IsKeyword("IF EXISTS") {
		stmt.Exists = true
		p.Next()
	}
	for !p.QueryEnds() && !p.Done() {
		n, err := p.ParseIdentifier()
		if err != nil {
			return nil, err
		}
		stmt.Names = append(stmt.Names, n)
		if !p.Is(Comma) {
			break
		}
		p.Next()

	}
	if p.IsKeyword("RESTRICT") {
		stmt.Cascade = ast.Restrict
		p.Next()
	} else if p.IsKeyword("CASCADE") {
		stmt.Cascade = ast.Cascade
		p.Next()
	}
	return stmt, err
}

func (p *Parser) ParseAlterTable() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.AlterTableStatement
		err  error
	)
	stmt.Name, err = p.ParseIdentifier()
	if err != nil {
		return nil, err
	}
	switch {
	case p.IsKeyword("RENAME TO"):
		p.Next()
		stmt.Action = ast.RenameTableAction{
			Name: p.GetCurrLiteral(),
		}
		p.Next()
	case p.IsKeyword("RENAME CONSTRAINT"):
		p.Next()
		src := p.GetCurrLiteral()
		p.Next()
		if !p.IsKeyword("TO") {
			return nil, p.Unexpected("alter table")
		}
		p.Next()
		dst := p.GetCurrLiteral()
		stmt.Action = ast.RenameConstraintAction{
			Old: src,
			New: dst,
		}
		p.Next()
	case p.IsKeyword("RENAME") || p.IsKeyword("RENAME COLUMN"):
		p.Next()
		src := p.GetCurrLiteral()
		p.Next()
		if !p.IsKeyword("TO") {
			return nil, p.Unexpected("alter table")
		}
		p.Next()
		dst := p.GetCurrLiteral()
		stmt.Action = ast.RenameColumnAction{
			Old: src,
			New: dst,
		}
		p.Next()
	case p.IsKeyword("ADD") || p.IsKeyword("ADD COLUMN"):
		p.Next()
		var notExists bool
		if notExists = p.IsKeyword("IF NOT EXISTS"); notExists {
			p.Next()
		}
		def, err := p.ParseColumnDef(p)
		if err != nil {
			return nil, err
		}
		stmt.Action = ast.AddColumnAction{
			Def:       def,
			NotExists: notExists,
		}
	case p.IsKeyword("ADD CONSTRAINT"):
		cst, err := p.parseConstraintWithKeyword("ADD CONSTRAINT", true, false)
		if err != nil {
			return nil, err
		}
		stmt.Action = ast.AddConstraintAction{
			Constraint: cst,
		}
	case p.IsKeyword("ALTER") || p.IsKeyword("ALTER COLUMN"):
		p.Next()
		var (
			action ast.AlterColumnAction
			err    error
		)
		action.Name = p.GetCurrLiteral()
		p.Next()
		stmt.Action = action
		return nil, err
	case p.IsKeyword("DROP CONSTRAINT"):
		p.Next()
		var exists bool
		if exists = p.IsKeyword("IF EXISTS"); exists {
			p.Next()
		}
		action := ast.DropConstraintAction{
			Name:   p.GetCurrLiteral(),
			Exists: exists,
		}
		p.Next()
		if p.IsKeyword("CASCADE") {
			action.Cascade = ast.Cascade
			p.Next()
		} else if p.IsKeyword("RESTRICT") {
			action.Cascade = ast.Restrict
			p.Next()
		}
		stmt.Action = action
	case p.IsKeyword("DROP") || p.IsKeyword("DROP COLUMN"):
		p.Next()
		var exists bool
		if exists = p.IsKeyword("IF EXISTS"); exists {
			p.Next()
		}
		action := ast.DropColumnAction{
			Name:   p.GetCurrLiteral(),
			Exists: exists,
		}
		p.Next()
		if p.IsKeyword("CASCADE") {
			action.Cascade = ast.Cascade
			p.Next()
		} else if p.IsKeyword("RESTRICT") {
			action.Cascade = ast.Restrict
			p.Next()
		}
		stmt.Action = action
	default:
		return nil, p.Unexpected("alter table")
	}
	return stmt, nil
}

func (p *Parser) ParseCreateTable() (ast.Statement, error) {
	return p.ParseCreateTableStatement(p)
}

func (p *Parser) ParseCreateTableStatement(ctp CreateTableParser) (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.CreateTableStatement
		err  error
	)
	if p.IsKeyword("IF NOT EXISTS") {
		p.Next()
		stmt.NotExists = true
	}
	if stmt.Name, err = ctp.ParseTableName(); err != nil {
		return nil, err
	}
	if err := p.Expect("create table", Lparen); err != nil {
		return nil, err
	}
	for !p.Done() && !p.Is(Rparen) && !p.Is(Keyword) {
		def, err := ctp.ParseColumnDef(ctp)
		if err != nil {
			return nil, err
		}
		stmt.Columns = append(stmt.Columns, def)
		if err = p.EnsureEnd("create table", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	for !p.Done() && !p.Is(Rparen) {
		cst, err := ctp.ParseConstraint(false)
		if err != nil {
			return nil, err
		}
		stmt.Constraints = append(stmt.Constraints, cst)
		if err = p.EnsureEnd("create table", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	return stmt, p.Expect("create table", Rparen)
}

func (p *Parser) ParseCreateView() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.CreateViewStatement
		err  error
	)
	if p.IsKeyword("IF NOT EXISTS") {
		p.Next()
		stmt.NotExists = true
	}
	if stmt.Name, err = p.ParseTableName(); err != nil {
		return nil, err
	}
	if p.Is(Lparen) {
		stmt.Columns, err = p.parseColumnsList()
		if err != nil {
			return nil, err
		}
	}
	if !p.IsKeyword("AS") {
		return nil, p.Unexpected("create view")
	}
	p.Next()

	stmt.Select, err = p.ParseStatement()
	return stmt, err
}

func (p *Parser) ParseTableName() (ast.Statement, error) {
	return p.ParseIdentifier()
}

func (p *Parser) ParseColumnDef(ctp CreateTableParser) (ast.Statement, error) {
	var (
		def ast.ColumnDef
		err error
	)
	def.Name = p.GetCurrLiteral()
	p.Next()
	if def.Type, err = p.ParseType(); err != nil {
		return nil, err
	}
	if p.Is(Comma) {
		return def, nil
	}
	for !p.QueryEnds() && !p.Done() && !p.Is(Comma) && !p.Is(Rparen) {
		cst, err := ctp.ParseConstraint(true)
		if err != nil {
			return nil, err
		}
		def.Constraints = append(def.Constraints, cst)
	}
	return def, err
}

func (p *Parser) ParseConstraint(column bool) (ast.Statement, error) {
	return p.parseConstraintWithKeyword("CONSTRAINT", false, column)
}

func (p *Parser) parseConstraintWithKeyword(keyword string, required, column bool) (ast.Statement, error) {
	var (
		cst ast.Constraint
		err error
	)
	if p.IsKeyword(keyword) {
		p.Next()
		cst.Name = p.GetCurrLiteral()
		p.Next()
	} else if required && !p.IsKeyword(keyword) {
		return nil, p.Unexpected("constraint")
	}
	switch {
	case p.IsKeyword("PRIMARY KEY"):
		cst.Statement, err = p.ParsePrimaryKeyConstraint(column)
	case p.IsKeyword("FOREIGN KEY") || p.IsKeyword("REFERENCES"):
		cst.Statement, err = p.ParseForeignKeyConstraint(column)
	case p.IsKeyword("UNIQUE"):
		cst.Statement, err = p.ParseUniqueConstraint(column)
	case p.IsKeyword("NOT"):
		if !column {
			return nil, p.Unexpected("constraint")
		}
		cst.Statement, err = p.ParseNotNullConstraint()
	case p.IsKeyword("CHECK"):
		cst.Statement, err = p.ParseCheckConstraint()
	case p.IsKeyword("DEFAULT"):
		if !column {
			return nil, p.Unexpected("constraint")
		}
		cst.Statement, err = p.ParseDefaultConstraint()
	case p.IsKeyword("GENERATED ALWAYS") || p.IsKeyword("AS"):
		cst.Statement, err = p.ParseGeneratedAlwaysConstraint()
	default:
		return nil, p.Unexpected("constraint")
	}
	return cst, err
}

func (p *Parser) ParsePrimaryKeyConstraint(short bool) (ast.Statement, error) {
	p.Next()
	var cst ast.PrimaryKeyConstraint
	if short {
		return cst, nil
	}
	if err := p.Expect("primary key", Lparen); err != nil {
		return nil, err
	}
	for !p.Done() && !p.Is(Rparen) {
		if !p.Is(Ident) {
			return nil, p.Unexpected("primary key")
		}
		cst.Columns = append(cst.Columns, p.GetCurrLiteral())
		p.Next()
		if err := p.EnsureEnd("primary key", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	return cst, p.Expect("primary key", Rparen)
}

func (p *Parser) ParseForeignKeyConstraint(short bool) (ast.Statement, error) {
	var cst ast.ForeignKeyConstraint
	if p.IsKeyword("FOREIGN KEY") {
		p.Next()
		if err := p.Expect("foreign key", Lparen); err != nil {
			return nil, err
		}
		for !p.Done() && !p.Is(Rparen) {
			if !p.Is(Ident) {
				return nil, p.Unexpected("foreign key")
			}
			cst.Locals = append(cst.Locals, p.GetCurrLiteral())
			p.Next()
			if err := p.EnsureEnd("foreign key", Comma, Rparen); err != nil {
				return nil, err
			}
		}
		if err := p.Expect("foreign key", Rparen); err != nil {
			return nil, err
		}
	}
	if !p.IsKeyword("REFERENCES") {
		return nil, p.Unexpected("foreign key")
	}
	p.Next()
	if !p.Is(Ident) {
		return nil, p.Unexpected("foreign key")
	}
	cst.Table = p.GetCurrLiteral()
	p.Next()
	if err := p.Expect("foreign key", Lparen); err != nil {
		return nil, err
	}
	for !p.Done() && !p.Is(Rparen) {
		if !p.Is(Ident) {
			return nil, p.Unexpected("foreign key")
		}
		cst.Remotes = append(cst.Remotes, p.GetCurrLiteral())
		p.Next()
		if err := p.EnsureEnd("foreign key", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	return cst, p.Expect("foreign key", Rparen)
}

func (p *Parser) ParseUniqueConstraint(short bool) (ast.Statement, error) {
	p.Next()
	var cst ast.UniqueConstraint
	if short {
		return cst, nil
	}
	if err := p.Expect("unique", Lparen); err != nil {
		return nil, err
	}
	for !p.Done() && !p.Is(Rparen) {
		if !p.Is(Ident) {
			return nil, p.Unexpected("unique")
		}
		cst.Columns = append(cst.Columns, p.GetCurrLiteral())
		p.Next()
		if err := p.EnsureEnd("unique", Comma, Rparen); err != nil {
			return nil, err
		}
	}
	return cst, p.Expect("unique", Rparen)
}

func (p *Parser) ParseNotNullConstraint() (ast.Statement, error) {
	p.Next()
	var cst ast.NotNullConstraint
	if !p.IsKeyword("NULL") {
		return nil, p.Unexpected("not null")
	}
	p.Next()
	return cst, nil
}

func (p *Parser) ParseCheckConstraint() (ast.Statement, error) {
	p.Next()
	var (
		cst ast.CheckConstraint
		err error
	)
	cst.Expr, err = p.StartExpression()
	return cst, err
}

func (p *Parser) ParseDefaultConstraint() (ast.Statement, error) {
	p.Next()
	var (
		cst ast.DefaultConstraint
		err error
	)
	cst.Expr, err = p.StartExpression()
	return cst, err
}

func (p *Parser) ParseGeneratedAlwaysConstraint() (ast.Statement, error) {
	if p.IsKeyword("GENERATED ALWAYS") {
		p.Next()
		if !p.IsKeyword("AS") {
			return nil, p.Unexpected("generated always")
		}
	}
	p.Next()
	var (
		cst ast.GeneratedConstraint
		err error
	)
	cst.Expr, err = p.StartExpression()
	if err != nil {
		return nil, err
	}
	if !p.IsKeyword("STORED") {
		return nil, p.Unexpected("generated always")
	}
	p.Next()
	return cst, nil
}
