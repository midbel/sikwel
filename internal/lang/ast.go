package lang

type Statement interface{}

type Commit struct{}

type Rollback struct{}

type Type struct {
	Name   string
	Length int
}

type Declare struct {
	Ident string
	Type  Type
	Value Statement
}

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

type Return struct {
	Statement
}

type Value struct {
	Literal string
}

// type Value[T string | float64 | int64 | bool] struct {
// 	Literal T
// }

// func createString(s string) Value[string] {
// 	return Value[string]{
// 		Literal: s,
// 	}
// }

// func createBool(b bool) Value[bool] {
// 	return Value[bool]{
// 		Literal: b,
// 	}
// }

// func createInt(i int64) Value[int64] {
// 	return Value[int64]{
// 		Literal: i,
// 	}
// }

// func createFloat(f float64) Value[float64] {
// 	return Value[float64]{
// 		Literal: f,
// 	}
// }

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

type ValuesStatement struct {
	List []Statement
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
	Table   Statement
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

type SetStatement struct {
	Ident string
	Expr  Statement
}
