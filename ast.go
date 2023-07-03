package sweet

type Statement interface{}

type Commit struct{}

type Rollback struct{}

type Call struct {
	Ident Statement
	Args  []Statement
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

type Between struct {
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

type Value struct {
	Literal string
}

type Name struct {
	Prefix string
	Ident  string
}

type Alias struct {
	Statement
	Alias string
}

type Limit struct {
	Count  int
	Offset int
}

type Order struct {
	Statement
	Orient string
	Nulls  string
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

type CteStatement struct {
	Ident   string
	Columns []string
	Statement
}

type WithStatement struct {
	Queries []Statement
	Statement
}

type SelectStatement struct {
	All      bool
	Distinct bool
	Columns  []Statement
	Tables   []Statement
	Where    Statement
	Groups   []Statement
	Having   Statement
	Orders   []Statement
	Limit    Statement
}

type UnionStatement struct {
	Left     Statement
	Right    Statement
	All      bool
	Distinct bool
}

type IntersectStatement struct {
	Left     Statement
	Right    Statement
	All      bool
	Distinct bool
}

type ExceptStatement struct {
	Left     Statement
	Right    Statement
	All      bool
	Distinct bool
}

type UpsertStatement struct {
	Columns []string
	List    []Statement
	Where   Statement
}

type InsertStatement struct {
	Table   string
	Columns []string
	Values  Statement
	Upsert  Statement
	Return  Statement
}

type UpdateStatement struct {
	Table  Statement
	List   []Statement
	Tables []Statement
	Where  Statement
	Return Statement
}

type DeleteStatement struct {
	Table  string
	Where  Statement
	Return Statement
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
