package lang

import (
	"fmt"
	"slices"
	"strings"
)

func Print(stmt Statement) error {
	return nil
}

type Statement interface {
	// Keyword() (string, error)
	// fmt.Stringer
}

type Expression interface {
	// Operand() string
	// fmt.Stringer
}

type relation interface {
	IsRelation() bool
}

func hasSimple(st Statement) bool {
	r, ok := st.(relation)
	return !ok || !r.IsRelation()
}

func isRelation(st Statement) bool {
	r, ok := st.(relation)
	return ok && r.IsRelation()
}

func wrapWithParens(st Statement) bool {
	switch st.(type) {
	case SelectStatement:
	case ValuesStatement:
	default:
		return false
	}
	return true
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

func (n Not) GetNames() []string {
	return getNamesFromStmt([]Statement{n.Statement})
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

var sqlAggregates = []string{
	"max",
	"min",
	"avg",
	"sum",
	"count",
}

var sqlBuiltins = []string{
	"max",
	"min",
	"avg",
	"sum",
	"count",
}

type Call struct {
	Distinct bool
	Ident    Statement
	Args     []Statement
	Filter   Statement
	Over     Statement
}

func (c Call) GetNames() []string {
	return getNamesFromStmt(c.Args)
}

func (c Call) GetIdent() string {
	n, ok := c.Ident.(Name)
	if !ok {
		return "?"
	}
	return n.Ident()
}

func (c Call) IsAggregate() bool {
	return slices.Contains(sqlAggregates, c.GetIdent())
}

func (c Call) BuiltinSql() bool {
	return slices.Contains(sqlBuiltins, c.GetIdent())
}

type Row struct {
	Values []Statement
}

func (r Row) Keyword() (string, error) {
	return "ROW", nil
}

type Unary struct {
	Right Statement
	Op    string
}

func (u Unary) GetNames() []string {
	return getNamesFromStmt([]Statement{u.Right})
}

type Binary struct {
	Left  Statement
	Right Statement
	Op    string
}

func (b Binary) GetNames() []string {
	var list []string
	list = append(list, getNamesFromStmt([]Statement{b.Left})...)
	list = append(list, getNamesFromStmt([]Statement{b.Right})...)
	return list
}

func (b Binary) IsRelation() bool {
	return b.Op == "AND" || b.Op == "OR"
}

type All struct {
	Statement
}

type Any struct {
	Statement
}

type Is struct {
	Ident Statement
	Value Statement
}

type In struct {
	Ident Statement
	Value Statement
}

func (i In) GetNames() []string {
	var list []string
	list = append(list, getNamesFromStmt([]Statement{i.Ident})...)
	list = append(list, getNamesFromStmt([]Statement{i.Value})...)
	return list
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

func (n Name) Schema() string {
	switch len(n.Parts) {
	case 2:
		return n.Parts[0]
	case 3:
		return n.Parts[1]
	default:
		return ""
	}
}

func (n Name) Name() string {
	if len(n.Parts) == 0 {
		return "*"
	}
	str := n.Parts[len(n.Parts)-1]
	if str == "" {
		str = "*"
	}
	return str
}

func (n Name) Ident() string {
	z := len(n.Parts)
	if z == 0 {
		return "*"
	}
	if n.Parts[z-1] == "" {
		n.Parts[z-1] = "*"
	}
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

func (s WithStatement) GetNames() []string {
	q, ok := s.Statement.(interface{ GetNames() []string })
	if !ok {
		return nil
	}
	return q.GetNames()
}

func (s WithStatement) Keyword() (string, error) {
	return "WITH", nil
}

type ValuesStatement struct {
	List   []Statement
	Orders []Statement
	Limit  Statement
}

func (s ValuesStatement) Keyword() (string, error) {
	return "VALUES", nil
}

type SelectStatement struct {
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

func (s SelectStatement) ColumnsCount() int {
	return -1
}

func (s SelectStatement) Keyword() (string, error) {
	return "SELECT", nil
}

func (s SelectStatement) GetNames() []string {
	var list []string
	for _, c := range s.Columns {
		switch c := c.(type) {
		case Alias:
			list = append(list, c.Alias)
		case Name:
			if len(c.Parts) == 0 {
				return nil
			}
			n := c.Parts[len(c.Parts)-1]
			if n == "" || n == "*" {
				return nil
			}
			list = append(list, n)
		default:
			return nil
		}
	}
	return list
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

type MatchStatement struct {
	Condition Statement
	Statement
}

type MergeStatement struct {
	Target  Statement
	Source  Statement
	Join    Statement
	Actions []Statement
}

func (s MergeStatement) Keyword() (string, error) {
	return "MERGE", nil
}

type Upsert struct {
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

type IdentityMode int

const (
	RestartIdentity IdentityMode = iota + 1
	ContinueIdentity
)

type TruncateStatement struct {
	Tables   []string
	Cascade  bool
	Identity IdentityMode
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

type DropViewStatement struct {
	Names   []Statement
	Exists  bool
	Cascade bool
}

func (s DropViewStatement) Keyword() (string, error) {
	if s.Exists {
		return "DROP VIEW IF EXISTS", nil
	}
	return "DROP VIEW", nil
}

type DropTableStatement struct {
	Names   []Statement
	Exists  bool
	Cascade bool
}

func (s DropTableStatement) Keyword() (string, error) {
	if s.Exists {
		return "DROP TABLE IF EXISTS", nil
	}
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
	Object     string
	Privileges []string
	Users      []string
}

func (s GrantStatement) Keyword() (string, error) {
	return "GRANT", nil
}

type RevokeStatement struct {
	Object     string
	Privileges []string
	Users      []string
}

func (s RevokeStatement) Keyword() (string, error) {
	return "REVOKE", nil
}

func getNamesFromStatments(cs []Statement) []string {
	var list []string
	for _, c := range cs {
		c, ok := c.(Name)
		if !ok {
			continue
		}
		n := c.Parts[len(c.Parts)-1]
		if n == "" || n == "*" {
			continue
		}
		list = append(list, n)
	}
	return list
}

func getSchemasFromStmt(all []Statement) []string {
	var list []string
	for _, c := range all {
		if c, ok := c.(interface{ Schema() string }); ok {
			schema := c.Schema()
			if schema == "" {
				continue
			}
			list = append(list, schema)
		}
	}
	return list
}

func getAliasFromStmt(all []Statement) []string {
	var list []string
	for _, c := range all {
		a, ok := c.(Alias)
		if !ok {
			continue
		}
		list = append(list, a.Alias)
	}
	return list
}

func getNamesFromStmt(all []Statement) []string {
	get := func(s Statement) []string {
		if n, ok := s.(Name); ok {
			if len(n.Parts) == 0 {
				return nil
			}
			return []string{n.Parts[len(n.Parts)-1]}
		}
		if g, ok := s.(interface{ GetNames() []string }); ok {
			return g.GetNames()
		}
		return nil
	}
	var list []string
	for _, s := range all {
		list = append(list, get(s)...)
	}
	return list

}

func substituteQueries(list []Statement, stmt Statement) Statement {
	queries := make(map[string]Statement)
	for _, q := range list {
		c, ok := q.(CteStatement)
		if !ok {
			continue
		}
		queries[c.Ident] = c.Statement
	}
	switch q := stmt.(type) {
	case SelectStatement:
		stmt = substituteSelect(q, queries)
	default:
	}
	return stmt
}

func substituteSelect(stmt SelectStatement, queries map[string]Statement) Statement {
	for i, t := range stmt.Tables {
		ident := getIdentFromStatement(t)
		if ident == "" {
			continue
		}
		q, ok := queries[ident]
		if ok {
			stmt.Tables[i] = Alias{
				Statement: q,
				Alias:     ident,
			}
		}
	}
	return stmt
}

func getIdentFromStatement(stmt Statement) string {
	switch s := stmt.(type) {
	case Name:
		return s.Ident()
	case Alias:
		return getIdentFromStatement(s.Statement)
	case Join:
		return getIdentFromStatement(s.Table)
	default:
		return ""
	}
}
