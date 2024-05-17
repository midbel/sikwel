package lang

import (
	"fmt"
	"strings"
)

type Statement interface {
	// Keyword() (string, error)
	// fmt.Stringer
}

type Expression interface {
	// Operand() string
	// fmt.Stringer
}

type Commented struct {
	Before []string
	After  string
	Statement
}

type TransactionMode int

const (
	ModeReadWrite TransactionMode = 1 << (iota + 1)
	ModeReadOnly
)

type TransactionLevel int

const (
	LevelReadRepeat TransactionLevel = 1 << (iota + 1)
	LevelReadCommit
	LevelReadUncommit
	LevelSerializable
)

type SetTransaction struct {
	Mode  TransactionMode
	Level TransactionLevel
}

func (_ SetTransaction) Keyword() (string, error) {
	return "SET TRANSACTION", nil
}

type StartTransaction struct {
	Mode TransactionMode
	Body Statement
	End  Statement
}

func (_ StartTransaction) Keyword() (string, error) {
	return "START TRANSACTION", nil
}

type Savepoint struct {
	Name string
}

func (_ Savepoint) Keyword() (string, error) {
	return "SAVEPOINT", nil
}

type ReleaseSavepoint struct {
	Name string
}

func (_ ReleaseSavepoint) Keyword() (string, error) {
	return "RELEASE SAVEPOINT", nil
}

type RollbackSavepoint struct {
	Name string
}

func (_ RollbackSavepoint) Keyword() (string, error) {
	return "ROLLBACK TO SAVEPOINT", nil
}

type Commit struct{}

func (_ Commit) Keyword() (string, error) {
	return "COMMIT", nil
}

type Rollback struct{}

func (_ Rollback) Keyword() (string, error) {
	return "ROLLBACK", nil
}

type Type struct {
	Name      string
	Length    int
	Precision int
}

type Declare struct {
	Ident string
	Type  Type
	Value Statement
}

type Not struct {
	Statement
}

type Collate struct {
	Statement
	Collation string
}

type Cast struct {
	Ident Statement
	Type  Type
}

type Exists struct {
	Statement
}

type Call struct {
	Distinct bool
	Ident    Statement
	Args     []Statement
	Filter   Statement
	Over     Statement
}

type Row struct {
	Values []Statement
}

type Unary struct {
	Right Statement
	Op    string
}

type Binary struct {
	Left  Statement
	Right Statement
	Op    string
}

type Is struct {
	Ident Statement
	Value Statement
}

type In struct {
	Not   bool
	Ident Statement
	List  []Statement
}

type Between struct {
	Not   bool
	Ident Statement
	Lower Statement
	Upper Statement
}

type List struct {
	Values []Statement
}

func (i List) Len() int {
	return len(i.Values)
}

type Return struct {
	Statement
}

type Value struct {
	Literal string
}

type Name struct {
	Parts []string
}

func (n Name) All() bool {
	return false
}

func (n Name) Ident() string {
	return strings.Join(n.Parts, ".")
}

type Alias struct {
	Statement
	Alias string
}

type Limit struct {
	Count  int
	Offset int
}

type Offset struct {
	Limit
	Next bool
}

type Order struct {
	Statement
	Dir   string
	Nulls string
}

type Join struct {
	Type  string
	Table Statement
	Where Statement
}

type Assignment struct {
	Field Statement
	Value Statement
}

type WindowDefinition struct {
	Ident  Statement
	Window Statement
}

type Window struct {
	Ident      Statement
	Partitions []Statement
	Orders     []Statement
	Spec       FrameSpec
}

type FrameRow int

const (
	RowCurrent FrameRow = 1 << iota
	RowPreceding
	RowFollowing
	RowUnbounded
)

type FrameExclude int

const (
	ExcludeCurrent FrameExclude = 1 << (iota + 1)
	ExcludeNoOthers
	ExcludeGroup
	ExcludeTies
)

type FrameSpec struct {
	Row  FrameRow
	Expr Statement
}

type BetweenFrameSpec struct {
	Left    FrameSpec
	Right   FrameSpec
	Exclude FrameExclude
}

type MaterializedMode int

const (
	MaterializedCte MaterializedMode = iota + 1
	NotMaterializedCte
)

type CteStatement struct {
	Ident        string
	Materialized MaterializedMode
	Columns      []string
	Statement
}

type WithStatement struct {
	Recursive bool
	Queries   []Statement
	Statement
}

func (s WithStatement) Keyword() (string, error) {
	return "WITH", nil
}

type ValuesStatement struct {
	List []Statement
}

func (s ValuesStatement) Keyword() (string, error) {
	return "VALUES", nil
}

type SelectStatement struct {
	All      bool
	Distinct bool
	Columns  []Statement
	Tables   []Statement
	Where    Statement
	Groups   []Statement
	Having   Statement
	Windows  []Statement
	Orders   []Statement
	Limit    Statement
}

func (s SelectStatement) Keyword() (string, error) {
	return "SELECT", nil
}

func getCompoundKeyword(kw string, all, distinct bool) (string, error) {
	var suffix string
	switch {
	default:
		return kw, nil
	case all:
		suffix = "ALL"
	case distinct:
		suffix = "DISTINCT"
	case all && distinct:
		return "", fmt.Errorf("%s: all and distinct can not be set at the same time", kw)
	}
	return fmt.Sprintf("%s %s", kw, suffix), nil
}

type UnionStatement struct {
	Left     Statement
	Right    Statement
	All      bool
	Distinct bool
}

func (s UnionStatement) Keyword() (string, error) {
	return getCompoundKeyword("UNION", s.All, s.Distinct)
}

type IntersectStatement struct {
	Left     Statement
	Right    Statement
	All      bool
	Distinct bool
}

func (s IntersectStatement) Keyword() (string, error) {
	return getCompoundKeyword("INTERSECT", s.All, s.Distinct)
}

type ExceptStatement struct {
	Left     Statement
	Right    Statement
	All      bool
	Distinct bool
}

func (s ExceptStatement) Keyword() (string, error) {
	return getCompoundKeyword("EXCEPT", s.All, s.Distinct)
}

type UpsertStatement struct {
	Columns []string
	List    []Statement
	Where   Statement
}

type InsertStatement struct {
	Table   Statement
	Columns []string
	Values  Statement
	Upsert  Statement
	Return  Statement
}

func (s InsertStatement) Keyword() (string, error) {
	return "INSERT INTO", nil
}

type UpdateStatement struct {
	Table  Statement
	List   []Statement
	Tables []Statement
	Where  Statement
	Return Statement
}

func (s UpdateStatement) Keyword() (string, error) {
	return "UPDATE", nil
}

type TruncateStatement struct {
	Tables []string
}

func (s TruncateStatement) Keyword() (string, error) {
	return "TRUNCATE", nil
}

type DeleteStatement struct {
	Table  string
	Where  Statement
	Return Statement
}

func (s DeleteStatement) Keyword() (string, error) {
	return "DELETE FROM", nil
}

type CallStatement struct {
	Ident Statement
	Names []string
	Args  []Statement
}

func (_ CallStatement) Keyword() (string, error) {
	return "CALL", nil
}

type WhileStatement struct {
	Cdt  Statement
	Body Statement
}

type IfStatement struct {
	Cdt Statement
	Csq Statement
	Alt Statement
}

type CaseStatement struct {
	Cdt  Statement
	Body []Statement
	Else Statement
}

type WhenStatement struct {
	Cdt  Statement
	Body Statement
}

type SetStatement struct {
	Ident string
	Expr  Statement
}

type PrimaryKeyConstraint struct {
	Columns []string
}

func (_ PrimaryKeyConstraint) Keyword() (string, error) {
	return "PRIMARY KEY", nil
}

type ForeignKeyConstraint struct {
	Locals   []string
	Remotes  []string
	Table    string
	OnDelete Statement
	OnUpdate Statement
}

func (c ForeignKeyConstraint) Keyword() (string, error) {
	if len(c.Locals) == 0 {
		return "REFERENCES", nil
	}
	return "FOREIGN KEY", nil
}

type NotNullConstraint struct {
	Column string
}

func (_ NotNullConstraint) Keyword() (string, error) {
	return "NOT NULL", nil
}

type UniqueConstraint struct {
	Columns []string
}

func (_ UniqueConstraint) Keyword() (string, error) {
	return "UNIQUE", nil
}

type CheckConstraint struct {
	Expr Statement
}

func (_ CheckConstraint) Keyword() (string, error) {
	return "CHECK", nil
}

type DefaultConstraint struct {
	Expr Statement
}

func (_ DefaultConstraint) Keyword() (string, error) {
	return "DEFAULT", nil
}

type GeneratedConstraint struct {
	Expr Statement
}

func (_ GeneratedConstraint) Keyword() (string, error) {
	return "GENERATED ALWAYS AS", nil
}

type Constraint struct {
	Name string
	Statement
}

type ColumnDef struct {
	Name        string
	Type        Type
	Constraints []Statement
}

type RenameTableAction struct {
	Name string
}

type RenameColumnAction struct {
	Src string
	Dst string
}

type AddColumnAction struct {
	Def       Statement
	NotExists bool
}

type DropColumnAction struct {
	Name   string
	Exists bool
}

type AlterTableStatement struct {
	Name   Statement
	Action Statement
}

type DropTableStatement struct {
	Name   Statement
	Exists bool
}

func (s DropTableStatement) Keyword() (string, error) {
	if s.Exists {
		return "DROP TABLE IF EXISTS", nil
	}
	return "DROP TABLE", nil
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

type ParameterMode int

const (
	ModeIn ParameterMode = 1 << (iota + 1)
	ModeOut
	ModeInOut
)

type ProcedureParameter struct {
	Mode    ParameterMode
	Name    string
	Type    Type
	Default Statement
}

type CreateProcedureStatement struct {
	Replace    bool
	Name       string
	Parameters []Statement
	Language   string
	Body       Statement
}

func (s CreateProcedureStatement) Keyword() (string, error) {
	if s.Replace {
		return "CREATE OR REPLACE PROCEDURE", nil
	}
	return "CREATE PROCEDURE", nil
}

type GrantStatement struct {
}

type RevokeStatement struct {
}
