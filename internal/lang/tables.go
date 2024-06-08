package lang

import (
	"fmt"
	"strings"
)

type CreateTableParser interface {
	ParseTableName() (Statement, error)
	ParseConstraint(bool) (Statement, error)
	ParseColumnDef(CreateTableParser) (Statement, error)
}

func (p *Parser) ParseDropTable() (Statement, error) {
	p.Next()
	var (
		stmt DropTableStatement
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
		stmt.Cascade = Restrict
		p.Next()
	} else if p.IsKeyword("CASCADE") {
		stmt.Cascade = Cascade
		p.Next()
	}
	return stmt, err
}

func (p *Parser) ParseDropView() (Statement, error) {
	p.Next()
	var (
		stmt DropViewStatement
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
		stmt.Cascade = Restrict
		p.Next()
	} else if p.IsKeyword("CASCADE") {
		stmt.Cascade = Cascade
		p.Next()
	}
	return stmt, err
}

func (p *Parser) ParseAlterTable() (Statement, error) {
	p.Next()
	var (
		stmt AlterTableStatement
		err  error
	)
	stmt.Name, err = p.ParseIdentifier()
	if err != nil {
		return nil, err
	}
	switch {
	case p.IsKeyword("RENAME TO"):
		p.Next()
		stmt.Action = RenameTableAction{
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
		stmt.Action = RenameConstraintAction{
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
		stmt.Action = RenameColumnAction{
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
		stmt.Action = AddColumnAction{
			Def:       def,
			NotExists: notExists,
		}
	case p.IsKeyword("ADD CONSTRAINT"):
		cst, err := p.parseConstraintWithKeyword("ADD CONSTRAINT", true, true)
		if err != nil {
			return nil, err
		}
		stmt.Action = AddConstraintAction{
			Constraint: cst,
		}
	case p.IsKeyword("ALTER") || p.IsKeyword("ALTER COLUMN"):
	case p.IsKeyword("DROP CONSTRAINT"):
		p.Next()
		var exists bool
		if exists = p.IsKeyword("IF EXISTS"); exists {
			p.Next()
		}
		action := DropConstraintAction{
			Name:   p.GetCurrLiteral(),
			Exists: exists,
		}
		p.Next()
		if p.IsKeyword("CASCADE") {
			action.Cascade = Cascade
			p.Next()
		} else if p.IsKeyword("RESTRICT") {
			action.Cascade = Restrict
			p.Next()
		}
		stmt.Action = action
	case p.IsKeyword("DROP") || p.IsKeyword("DROP COLUMN"):
		p.Next()
		var exists bool
		if exists = p.IsKeyword("IF EXISTS"); exists {
			p.Next()
		}
		action := DropColumnAction{
			Name:   p.GetCurrLiteral(),
			Exists: exists,
		}
		p.Next()
		if p.IsKeyword("CASCADE") {
			action.Cascade = Cascade
			p.Next()
		} else if p.IsKeyword("RESTRICT") {
			action.Cascade = Restrict
			p.Next()
		}
		stmt.Action = action
	default:
		return nil, p.Unexpected("alter table")
	}
	return stmt, nil
}

func (p *Parser) ParseCreateTable() (Statement, error) {
	return p.ParseCreateTableStatement(p)
}

func (p *Parser) ParseCreateTableStatement(ctp CreateTableParser) (Statement, error) {
	p.Next()
	var (
		stmt CreateTableStatement
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

func (p *Parser) ParseCreateView() (Statement, error) {
	p.Next()
	var (
		stmt CreateViewStatement
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

func (p *Parser) ParseTableName() (Statement, error) {
	return p.ParseIdentifier()
}

func (p *Parser) ParseColumnDef(ctp CreateTableParser) (Statement, error) {
	var (
		def ColumnDef
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

func (p *Parser) ParseConstraint(column bool) (Statement, error) {
	return p.parseConstraintWithKeyword("CONSTRAINT", false, column)
}

func (p *Parser) parseConstraintWithKeyword(keyword string, required, column bool) (Statement, error) {
	var (
		cst Constraint
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

func (p *Parser) ParsePrimaryKeyConstraint(short bool) (Statement, error) {
	p.Next()
	var cst PrimaryKeyConstraint
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

func (p *Parser) ParseForeignKeyConstraint(short bool) (Statement, error) {
	var cst ForeignKeyConstraint
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

func (p *Parser) ParseUniqueConstraint(short bool) (Statement, error) {
	p.Next()
	var cst UniqueConstraint
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

func (p *Parser) ParseNotNullConstraint() (Statement, error) {
	p.Next()
	var cst NotNullConstraint
	if !p.IsKeyword("NULL") {
		return nil, p.Unexpected("not null")
	}
	p.Next()
	return cst, nil
}

func (p *Parser) ParseCheckConstraint() (Statement, error) {
	p.Next()
	var (
		cst CheckConstraint
		err error
	)
	cst.Expr, err = p.StartExpression()
	return cst, err
}

func (p *Parser) ParseDefaultConstraint() (Statement, error) {
	p.Next()
	var (
		cst DefaultConstraint
		err error
	)
	cst.Expr, err = p.StartExpression()
	return cst, err
}

func (p *Parser) ParseGeneratedAlwaysConstraint() (Statement, error) {
	if p.IsKeyword("GENERATED ALWAYS") {
		p.Next()
		if !p.IsKeyword("AS") {
			return nil, p.Unexpected("generated always")
		}
	}
	p.Next()
	var (
		cst GeneratedConstraint
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

func (w *Writer) FormatCreateView(stmt CreateViewStatement) error {
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
			if w.Upperize {
				s = strings.ToUpper(s)
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
	FormatTableName(Statement) error
	FormatColumnDef(ConstraintFormatter, Statement, int) error
	ConstraintFormatter
}

type ConstraintFormatter interface {
	FormatConstraint(Statement) error

	FormatPrimaryKeyConstraint(PrimaryKeyConstraint) error
	FormatForeignKeyConstraint(ForeignKeyConstraint) error
	FormatDefaultConstraint(DefaultConstraint) error
	FormatNotNullConstraint(NotNullConstraint) error
	FormatUniqueConstraint(UniqueConstraint) error
	FormatCheckConstraint(CheckConstraint) error
	FormatGeneratedConstraint(GeneratedConstraint) error
}

func (w *Writer) FormatCreateTable(stmt CreateTableStatement) error {
	return w.FormatCreateTableWithFormatter(w, stmt)
}

func (w *Writer) FormatCreateTableWithFormatter(ctf CreateTableFormatter, stmt CreateTableStatement) error {
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
			d, ok := c.(ColumnDef)
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

func (w *Writer) FormatTableName(stmt Statement) error {
	return w.FormatExpr(stmt, false)
}

func (w *Writer) FormatColumnDef(ctf ConstraintFormatter, stmt Statement, size int) error {
	def, ok := stmt.(ColumnDef)
	if !ok {
		return fmt.Errorf("%T can not be used as column definition", stmt)
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

func (w *Writer) FormatConstraint(stmt Statement) error {
	cst, ok := stmt.(Constraint)
	if !ok {
		return fmt.Errorf("%T can not be used as constraint", stmt)
	}
	if cst.Name != "" {
		w.WriteKeyword("CONSTRAINT")
		w.WriteBlank()
		w.WriteString(cst.Name)
		w.WriteBlank()
	}
	switch stmt := cst.Statement.(type) {
	case PrimaryKeyConstraint:
		return w.FormatPrimaryKeyConstraint(stmt)
	case ForeignKeyConstraint:
		return w.FormatForeignKeyConstraint(stmt)
	case NotNullConstraint:
		return w.FormatNotNullConstraint(stmt)
	case UniqueConstraint:
		return w.FormatUniqueConstraint(stmt)
	case CheckConstraint:
		return w.FormatCheckConstraint(stmt)
	case DefaultConstraint:
		return w.FormatDefaultConstraint(stmt)
	case GeneratedConstraint:
		return w.FormatGeneratedConstraint(stmt)
	default:
		return fmt.Errorf("%T: unsupported constraint type", cst.Statement)
	}
}

func (w *Writer) FormatPrimaryKeyConstraint(cst PrimaryKeyConstraint) error {
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

func (w *Writer) FormatForeignKeyConstraint(cst ForeignKeyConstraint) error {
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

func (w *Writer) FormatNotNullConstraint(cst NotNullConstraint) error {
	kw, _ := cst.Keyword()
	w.WriteKeyword(kw)
	return nil
}

func (w *Writer) FormatUniqueConstraint(cst UniqueConstraint) error {
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

func (w *Writer) FormatDefaultConstraint(cst DefaultConstraint) error {
	kw, _ := cst.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()
	_, ok := cst.Expr.(Value)
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

func (w *Writer) FormatCheckConstraint(cst CheckConstraint) error {
	kw, _ := cst.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()
	w.WriteString("(")
	if err := w.FormatExpr(cst.Expr, false); err != nil {
		return err
	}
	w.WriteString(")")
	return nil
}

func (w *Writer) FormatGeneratedConstraint(cst GeneratedConstraint) error {
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

func (w *Writer) FormatAlterTable(stmt AlterTableStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Name, false); err != nil {
		return err
	}
	w.WriteBlank()
	switch action := stmt.Action.(type) {
	case DropColumnAction:
		w.WriteKeyword("DROP COLUMN")
		if action.Exists {
			w.WriteBlank()
			w.WriteKeyword("IF EXISTS")
		}
		w.WriteBlank()
		w.WriteString(action.Name)
		if action.Cascade == Cascade {
			w.WriteBlank()
			w.WriteKeyword("CASCADE")
		} else if action.Cascade == Restrict {
			w.WriteBlank()
			w.WriteKeyword("RESTRICT")
		}
	case AddColumnAction:
		w.WriteKeyword("ADD COLUMN")
		w.WriteBlank()
		_, ok := action.Def.(ColumnDef)
		if !ok {
			return w.CanNotUse("add column", action.Def)
		}
	case AlterColumnAction:
	case RenameColumnAction:
		w.WriteKeyword("RENAME COLUMN")
		w.WriteBlank()
		w.WriteString(action.Old)
		w.WriteBlank()
		w.WriteKeyword("TO")
		w.WriteBlank()
		w.WriteString(action.New)
	case AddConstraintAction:
	case DropConstraintAction:
	case RenameConstraintAction:
		w.WriteKeyword("RENAME CONSTRAINT")
		w.WriteBlank()
		w.WriteString(action.Old)
		w.WriteBlank()
		w.WriteKeyword("TO")
		w.WriteBlank()
		w.WriteString(action.New)
	case RenameTableAction:
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

func (w *Writer) FormatDropView(stmt DropViewStatement) error {
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
	case Cascade:
		w.WriteBlank()
		w.WriteKeyword("CASCADE")
	case Restrict:
		w.WriteBlank()
		w.WriteKeyword("RESTRICT")
	default:
	}
	return nil
}

func (w *Writer) FormatDropTable(stmt DropTableStatement) error {
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
	case Cascade:
		w.WriteBlank()
		w.WriteKeyword("CASCADE")
	case Restrict:
		w.WriteBlank()
		w.WriteKeyword("RESTRICT")
	default:
	}
	return nil
}
