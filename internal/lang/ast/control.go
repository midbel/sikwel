package ast

type Return struct {
	Statement
}

type While struct {
	Cdt  Statement
	Body Statement
}

type If struct {
	Cdt Statement
	Csq Statement
	Alt Statement
}

type Declare struct {
	Ident string
	Type  Type
	Value Statement
}

type Case struct {
	Cdt  Statement
	Body []Statement
	Else Statement
}

type When struct {
	Cdt  Statement
	Body Statement
}

type Set struct {
	Ident string
	Expr  Statement
}

type CallStatement struct {
	Ident Statement
	Names []string
	Args  []Statement
}

func (_ CallStatement) Keyword() (string, error) {
	return "CALL", nil
}
