package lang

import (
	"fmt"
	"strings"

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

func (w *Writer) FormatCreateView(stmt ast.CreateViewStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	if stmt.NotExists {
		w.WriteKeyword("IF NOT EXISTS")
		w.WriteBlank()
	}
	if err := w.FormatTableName(stmt.Name); err != nil {
		return err
	}

	if len(stmt.Columns) == 0 && w.UseNames {
		if q, ok := stmt.Select.(interface{ GetNames() []string }); ok {
			stmt.Columns = q.GetNames()
		}
	}
	if len(stmt.Columns) > 0 {
		w.WriteBlank()
		w.WriteString("(")
		for i, s := range stmt.Columns {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			if w.Upperize.Identifier() || w.Upperize.All() {
				s = strings.ToUpper(s)
			}
			if w.UseQuote {
				s = w.Quote(s)
			}
			w.WriteString(s)
		}
		w.WriteString(")")
	}

	w.WriteBlank()
	w.WriteStatement("AS")
	w.WriteNL()

	w.Leave()
	defer w.Enter()
	return w.FormatStatement(stmt.Select)
}

type CreateTableFormatter interface {
	FormatTableName(ast.Statement) error
	FormatColumnDef(ConstraintFormatter, ast.Statement, int) error
	ConstraintFormatter
}

type ConstraintFormatter interface {
	FormatConstraint(ast.Statement) error

	FormatPrimaryKeyConstraint(ast.PrimaryKeyConstraint) error
	FormatForeignKeyConstraint(ast.ForeignKeyConstraint) error
	FormatDefaultConstraint(ast.DefaultConstraint) error
	FormatNotNullConstraint(ast.NotNullConstraint) error
	FormatUniqueConstraint(ast.UniqueConstraint) error
	FormatCheckConstraint(ast.CheckConstraint) error
	FormatGeneratedConstraint(ast.GeneratedConstraint) error
}

func (w *Writer) FormatCreateTable(stmt ast.CreateTableStatement) error {
	return w.FormatCreateTableWithFormatter(w, stmt)
}

func (w *Writer) FormatCreateTableWithFormatter(ctf CreateTableFormatter, stmt ast.CreateTableStatement) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	if stmt.NotExists {
		w.WriteKeyword("IF NOT EXISTS")
		w.WriteBlank()
	}
	if err := ctf.FormatTableName(stmt.Name); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteString("(")
	w.WriteNL()

	w.Enter()
	defer w.Leave()
	var longest int
	if !w.Compact {
		for _, c := range stmt.Columns {
			d, ok := c.(ast.ColumnDef)
			if !ok {
				continue
			}
			if z := len(d.Name); z > longest {
				longest = z
			}
		}
	}
	for i, c := range stmt.Columns {
		if i > 0 {
			w.WriteString(",")
			w.WriteNL()
		}
		if err := ctf.FormatColumnDef(ctf, c, longest); err != nil {
			return err
		}
	}
	for _, c := range stmt.Constraints {
		w.WriteString(",")
		w.WriteNL()
		w.WritePrefix()
		if err := ctf.FormatConstraint(c); err != nil {
			return err
		}
	}
	w.WriteNL()
	w.WriteString(")")
	return nil
}

func (w *Writer) FormatTableName(stmt ast.Statement) error {
	return w.FormatExpr(stmt, false)
}

func (w *Writer) FormatColumnDef(ctf ConstraintFormatter, stmt ast.Statement, size int) error {
	def, ok := stmt.(ast.ColumnDef)
	if !ok {
		return w.CanNotUse("column", stmt)
	}
	w.WritePrefix()
	w.WriteString(def.Name)
	if z := len(def.Name); size > 0 && z < size {
		w.WriteString(strings.Repeat(" ", size-z))
	}
	w.WriteBlank()
	if err := w.FormatType(def.Type); err != nil {
		return err
	}

	for _, c := range def.Constraints {
		w.WriteBlank()
		if err := ctf.FormatConstraint(c); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatConstraint(stmt ast.Statement) error {
	return w.formatConstraint(stmt, "CONSTRAINT")
}

func (w *Writer) formatConstraint(stmt ast.Statement, keyword string) error {
	cst, ok := stmt.(ast.Constraint)
	if !ok {
		return w.CanNotUse("constraint", stmt)
	}
	if cst.Name != "" {
		w.WriteKeyword(keyword)
		w.WriteBlank()
		w.WriteString(cst.Name)
		w.WriteBlank()
	}
	switch stmt := cst.Statement.(type) {
	case ast.PrimaryKeyConstraint:
		return w.FormatPrimaryKeyConstraint(stmt)
	case ast.ForeignKeyConstraint:
		return w.FormatForeignKeyConstraint(stmt)
	case ast.NotNullConstraint:
		return w.FormatNotNullConstraint(stmt)
	case ast.UniqueConstraint:
		return w.FormatUniqueConstraint(stmt)
	case ast.CheckConstraint:
		return w.FormatCheckConstraint(stmt)
	case ast.DefaultConstraint:
		return w.FormatDefaultConstraint(stmt)
	case ast.GeneratedConstraint:
		return w.FormatGeneratedConstraint(stmt)
	default:
		return fmt.Errorf("%T: unsupported constraint type", cst.Statement)
	}
}

func (w *Writer) FormatPrimaryKeyConstraint(cst ast.PrimaryKeyConstraint) error {
	kw, _ := cst.Keyword()
	w.WriteKeyword(kw)
	if len(cst.Columns) == 0 {
		return nil
	}
	w.WriteBlank()
	w.WriteString("(")
	for i, c := range cst.Columns {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		w.WriteString(c)
	}
	w.WriteString(")")
	return nil
}

func (w *Writer) FormatForeignKeyConstraint(cst ast.ForeignKeyConstraint) error {
	if len(cst.Locals) > 0 {
		w.WriteKeyword("FOREIGN KEY")
		w.WriteBlank()
		w.WriteString("(")
		for i, c := range cst.Locals {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteString(c)
		}
		w.WriteString(")")
		w.WriteBlank()
	}
	if len(cst.Remotes) > 0 {
		w.WriteKeyword("REFERENCES")
		w.WriteBlank()
		w.WriteString(cst.Table)
		w.WriteString("(")
		for i, c := range cst.Remotes {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteString(c)
		}
		w.WriteString(")")
	}
	return nil
}

func (w *Writer) FormatNotNullConstraint(cst ast.NotNullConstraint) error {
	kw, _ := cst.Keyword()
	w.WriteKeyword(kw)
	return nil
}

func (w *Writer) FormatUniqueConstraint(cst ast.UniqueConstraint) error {
	kw, _ := cst.Keyword()
	w.WriteKeyword(kw)
	if len(cst.Columns) == 0 {
		return nil
	}
	w.WriteBlank()
	w.WriteString("(")
	for i, c := range cst.Columns {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		w.WriteString(c)
	}
	w.WriteString(")")
	return nil
}

func (w *Writer) FormatDefaultConstraint(cst ast.DefaultConstraint) error {
	kw, _ := cst.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()
	_, ok := cst.Expr.(ast.Value)
	if !ok {
		w.WriteString("(")
	}
	if err := w.FormatExpr(cst.Expr, false); err != nil {
		return err
	}
	if !ok {
		w.WriteString(")")
	}
	return nil
}

func (w *Writer) FormatCheckConstraint(cst ast.CheckConstraint) error {
	kw, _ := cst.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()
	if err := w.FormatExpr(cst.Expr, false); err != nil {
		return err
	}
	return nil
}

func (w *Writer) FormatGeneratedConstraint(cst ast.GeneratedConstraint) error {
	kw, _ := cst.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()
	w.WriteString("(")
	if err := w.FormatExpr(cst.Expr, false); err != nil {
		return err
	}
	w.WriteString(")")
	w.WriteBlank()
	w.WriteKeyword("STORED")
	return nil
}

func (w *Writer) FormatAlterTable(stmt ast.AlterTableStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Name, false); err != nil {
		return err
	}
	w.WriteBlank()
	switch action := stmt.Action.(type) {
	case ast.DropColumnAction:
		w.WriteKeyword("DROP COLUMN")
		if action.Exists {
			w.WriteBlank()
			w.WriteKeyword("IF EXISTS")
		}
		w.WriteBlank()
		w.WriteString(action.Name)
		if action.Cascade == ast.Cascade {
			w.WriteBlank()
			w.WriteKeyword("CASCADE")
		} else if action.Cascade == ast.Restrict {
			w.WriteBlank()
			w.WriteKeyword("RESTRICT")
		}
	case ast.AddColumnAction:
		w.WriteKeyword("ADD COLUMN")
		w.WriteBlank()

		def, ok := action.Def.(ast.ColumnDef)
		if !ok {
			return w.CanNotUse("add column", action.Def)
		}
		w.WriteString(def.Name)
		w.WriteBlank()
		if err := w.FormatType(def.Type); err != nil {
			return err
		}
		for _, c := range def.Constraints {
			w.WriteBlank()
			if err := w.FormatConstraint(c); err != nil {
				return err
			}
		}
		return nil
	case ast.AlterColumnAction:
	case ast.RenameColumnAction:
		w.WriteKeyword("RENAME COLUMN")
		w.WriteBlank()
		w.WriteString(action.Old)
		w.WriteBlank()
		w.WriteKeyword("TO")
		w.WriteBlank()
		w.WriteString(action.New)
	case ast.AddConstraintAction:
		return w.formatConstraint(action.Constraint, "ADD CONSTRAINT")
	case ast.DropConstraintAction:
		w.WriteKeyword("DROP CONSTRAINT")
		if action.Exists {
			w.WriteBlank()
			w.WriteKeyword("IF EXISTS")
		}
		w.WriteBlank()
		w.WriteString(action.Name)
		if action.Cascade == ast.Cascade {
			w.WriteBlank()
			w.WriteKeyword("CASCADE")
		} else if action.Cascade == ast.Restrict {
			w.WriteBlank()
			w.WriteKeyword("RESTRICT")
		}
	case ast.RenameConstraintAction:
		w.WriteKeyword("RENAME CONSTRAINT")
		w.WriteBlank()
		w.WriteString(action.Old)
		w.WriteBlank()
		w.WriteKeyword("TO")
		w.WriteBlank()
		w.WriteString(action.New)
	case ast.RenameTableAction:
		w.WriteKeyword("RENAME")
		w.WriteBlank()
		w.WriteKeyword("TO")
		w.WriteBlank()
		w.WriteString(action.Name)
	default:
		return w.CanNotUse("alter table", action)
	}
	return nil
}

func (w *Writer) FormatDropView(stmt ast.DropViewStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Exists {
		w.WriteBlank()
		w.WriteKeyword("IF EXISTS")
	}
	w.WriteBlank()
	for i, s := range stmt.Names {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		if err := w.FormatExpr(s, false); err != nil {
			return err
		}
	}
	switch stmt.Cascade {
	case ast.Cascade:
		w.WriteBlank()
		w.WriteKeyword("CASCADE")
	case ast.Restrict:
		w.WriteBlank()
		w.WriteKeyword("RESTRICT")
	default:
	}
	return nil
}

func (w *Writer) FormatDropTable(stmt ast.DropTableStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Exists {
		w.WriteBlank()
		w.WriteKeyword("IF EXISTS")
	}
	w.WriteBlank()
	for i, s := range stmt.Names {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		if err := w.FormatExpr(s, false); err != nil {
			return err
		}
	}
	switch stmt.Cascade {
	case ast.Cascade:
		w.WriteBlank()
		w.WriteKeyword("CASCADE")
	case ast.Restrict:
		w.WriteBlank()
		w.WriteKeyword("RESTRICT")
	default:
	}
	return nil
}
