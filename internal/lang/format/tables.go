package format

import (
	"fmt"
	"strings"

	"github.com/midbel/sweet/internal/lang/ast"
)

func (w *Writer) FormatCreateView(stmt ast.CreateViewStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()
	if stmt.NotExists {
		w.WriteKeyword("IF NOT EXISTS")
		w.WriteBlank()
	}
	if err := w.FormatTableName(stmt.Name); err != nil {
		return err
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
	w.WriteKeyword("AS")
	w.WriteNL()

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
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
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
	w.WriteKeyword(kw)
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
	w.WriteKeyword(kw)
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
	w.WriteKeyword(kw)
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
