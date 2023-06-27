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

type List struct {
	Values []Statement
}

func (i List) Len() int {
	return len(i.Values)
}

func (i List) AsStatement() Statement {
	if i.Len() == 1 {
		return i.Values[0]
	}
	return i
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
	Columns []Statement
	Tables  []Statement
	Where   Statement
	Groups  []Statement
	Having  Statement
	Orders  []Statement
	Limit   string
	Offset  string
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

type InsertStatement struct {
	Table   string
	Columns []string
	Values  Statement
	Return  Statement
}

type UpdateStatement struct {
	Table string
	List  []Statement
	Where Statement
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
