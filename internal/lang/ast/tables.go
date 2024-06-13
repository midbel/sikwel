package ast

type CascadeMode int

const (
	Cascade CascadeMode = iota + 1
	Restrict
)

type IdentityMode int

const (
	RestartIdentity IdentityMode = iota + 1
	ContinueIdentity
)

type ColumnDef struct {
	Name        string
	Type        Type
	Constraints []Statement
}

type RenameTableAction struct {
	Name string
}

type RenameColumnAction struct {
	Old string
	New string
}

type AddColumnAction struct {
	Def       Statement
	NotExists bool
}

type AlterColumnAction struct {
	Name string
}

type DropColumnAction struct {
	Name    string
	Exists  bool
	Cascade CascadeMode
}

type AddConstraintAction struct {
	Constraint Statement
}

type DropConstraintAction struct {
	Name    string
	Exists  bool
	Cascade CascadeMode
}

type RenameConstraintAction struct {
	Old string
	New string
}

type AlterTableStatement struct {
	Name   Statement
	Action Statement
}

func (s AlterTableStatement) Keyword() (string, error) {
	return "ALTER TABLE", nil
}

type DropViewStatement struct {
	Names   []Statement
	Exists  bool
	Cascade CascadeMode
}

func (s DropViewStatement) Keyword() (string, error) {
	return "DROP VIEW", nil
}

type DropTableStatement struct {
	Names   []Statement
	Exists  bool
	Cascade CascadeMode
}

func (s DropTableStatement) Keyword() (string, error) {
	return "DROP TABLE", nil
}

type CreateViewStatement struct {
	Temp      bool
	Name      Statement
	NotExists bool
	Columns   []string
	Select    Statement
}

func (s CreateViewStatement) Keyword() (string, error) {
	if s.Temp {
		return "CREATE TEMPORARY VIEW", nil
	}
	return "CREATE VIEW", nil
}

type CreateTableStatement struct {
	Temp        bool
	Name        Statement
	NotExists   bool
	Columns     []Statement
	Constraints []Statement
}

func (s CreateTableStatement) Keyword() (string, error) {
	if s.Temp {
		return "CREATE TEMPORARY TABLE", nil
	}
	return "CREATE TABLE", nil
}
