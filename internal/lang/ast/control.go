package ast

import (
	"github.com/midbel/sweet/internal/token"
)

type Return struct {
	token.Position
	Statement
}

type While struct {
	token.Position
	Cdt  Statement
	Body Statement
}

type If struct {
	token.Position
	Cdt Statement
	Csq Statement
	Alt Statement
}

type Declare struct {
	token.Position
	Ident string
	Type  Type
	Value Statement
}

type Case struct {
	token.Position
	Cdt  Statement
	Body []Statement
	Else Statement
}

type When struct {
	token.Position
	Cdt  Statement
	Body Statement
}

type Set struct {
	token.Position
	Ident string
	Expr  Statement
}

type CallStatement struct {
	token.Position
	Ident Statement
	Names []string
	Args  []Statement
}

func (_ CallStatement) Keyword() (string, error) {
	return "CALL", nil
}
