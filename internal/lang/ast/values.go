package ast

import (
	"slices"
	"strings"

	"github.com/midbel/sweet/internal/token"
)

func GetNamesFromWhere(where Statement, prefix string) []Statement {
	var (
		names []Statement
		walk  func(Statement)
		seen  = make(map[string]struct{})
	)

	walk = func(stmt Statement) {
		switch stmt := stmt.(type) {
		case Between:
			walk(stmt.Ident)
		case In:
			walk(stmt.Ident)
		case Is:
			walk(stmt.Ident)
		case Unary:
			walk(stmt.Right)
		case Binary:
			walk(stmt.Left)
			walk(stmt.Right)
		case Name:
			ident := stmt.Ident()
			if _, ok := seen[ident]; ok {
				break
			}
			if strings.HasPrefix(stmt.Ident(), prefix) {
				seen[ident] = struct{}{}
				names = append(names, stmt)
			}
		default:
		}
	}

	walk(where)
	return names
}

func ReplaceOp(b Binary) Binary {
	if b.Op == "!=" {
		b.Op = "<>"
	}
	return b
}

func ReplaceExpr(b Binary) Statement {
	v, ok := b.Right.(Value)
	if !ok {
		return b
	}
	if !v.Constant() {
		return b
	}
	x := Is{
		Ident: b.Left,
		Value: b.Right,
	}
	switch b.Op {
	case "=":
		return x
	case "<>":
		return Not{
			Statement: x,
		}
	default:
		return b
	}
}

type Group struct {
	Statement
}

type Cast struct {
	token.Position

	Ident Statement
	Type  Type
}

type Type struct {
	token.Position

	Name      string
	Length    int
	Precision int
}

type Not struct {
	token.Position
	Statement
}

func (n Not) GetNames() []string {
	return GetNamesFromStmt([]Statement{n.Statement})
}

type Collate struct {
	token.Position
	Statement
	Collation string
}

type Exists struct {
	token.Position
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
	Position token.Position
	Distinct bool
	Ident    Statement
	Args     []Statement
	Filter   Statement
	Over     Statement
}

func (c Call) GetNames() []string {
	return GetNamesFromStmt(c.Args)
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
	token.Position
	Values []Statement
}

func (r Row) Keyword() (string, error) {
	return "ROW", nil
}

type Unary struct {
	token.Position
	Right Statement
	Op    string
}

func (u Unary) GetNames() []string {
	return GetNamesFromStmt([]Statement{u.Right})
}

type Binary struct {
	token.Position
	Left  Statement
	Right Statement
	Op    string
}

func (b Binary) GetNames() []string {
	var list []string
	list = append(list, GetNamesFromStmt([]Statement{b.Left})...)
	list = append(list, GetNamesFromStmt([]Statement{b.Right})...)
	return list
}

func (b Binary) IsRelation() bool {
	return b.Op == "AND" || b.Op == "OR"
}

type All struct {
	token.Position
	Statement
}

type Any struct {
	token.Position
	Statement
}

type Is struct {
	token.Position
	Ident Statement
	Value Statement
}

type In struct {
	token.Position
	Ident Statement
	Value Statement
}

func (i In) GetNames() []string {
	var list []string
	list = append(list, GetNamesFromStmt([]Statement{i.Ident})...)
	list = append(list, GetNamesFromStmt([]Statement{i.Value})...)
	return list
}

type Between struct {
	token.Position
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

type Placeholder struct {
	token.Position
	Statement
}

type Value struct {
	token.Position
	Literal string
}

func (v Value) Constant() bool {
	return v.Null() || v.True() || v.False()
}

func (v Value) Null() bool {
	return v.Literal == "NULL"
}

func (v Value) True() bool {
	return v.Literal == "TRUE"
}

func (v Value) False() bool {
	return v.Literal == "FALSE"
}

type Alias struct {
	token.Position
	Statement
	Alias string
	As    bool
}

type Name struct {
	token.Position
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
