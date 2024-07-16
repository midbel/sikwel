package ast

import (
	"fmt"
)

type Node struct {
	Statement
	Before []string
	After  string
}

func (n Node) Get() Statement {
	if len(n.Before) == 0 && n.After == "" {
		return n.Statement
	}
	return n
}

type Statement interface{}

type Limit struct {
	Count  int
	Offset int
}

type Offset struct {
	Limit
	Next bool
}

type OrderDir uint8

const (
	AscOrder OrderDir = 1 << iota
	DescOrder
)

type Order struct {
	Statement
	Dir   OrderDir
	Nulls string
}

type Join struct {
	Type  string
	Table Statement
	Where Statement
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

func (s WithStatement) Get() Statement {
	if len(s.Queries) == 0 {
		return s.Statement
	}
	return s
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
		case Call:
			list = append(list, c.Ident.(Name).Name())
		default:
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

type Assignment struct {
	Field Statement
	Value Statement
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
	Tables   []string
	Cascade  CascadeMode
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

func GetAliasFromStmt(all []Statement) []string {
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

func GetNamesFromStmt(all []Statement) []string {
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

func SubstituteQueries(list []Statement, stmt Statement) Statement {
	queries := make(map[string]Statement)
	for _, q := range list {
		c, ok := q.(CteStatement)
		if !ok {
			continue
		}
		queries[c.Ident] = c.Statement
	}
	for n, q := range queries {
		s, ok := q.(SelectStatement)
		if !ok {
			continue
		}
		queries[n] = substituteSelect(s, queries)
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
		ident, alias := GetIdentFromStatement(t)
		if ident == "" {
			continue
		}
		q, ok := queries[ident]
		if ok {
			stmt.Tables[i] = Alias{
				Statement: q,
				Alias:     alias,
			}
		}
	}
	return stmt
}

func GetIdentFromStatement(stmt Statement) (string, string) {
	switch s := stmt.(type) {
	case Name:
		return s.Ident(), s.Ident()
	case Alias:
		ident, _ := GetIdentFromStatement(s.Statement)
		return ident, s.Alias
	case Join:
		return GetIdentFromStatement(s.Table)
	default:
		return "", ""
	}
}
