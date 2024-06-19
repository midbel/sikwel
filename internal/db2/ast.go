package db2

import (
	"github.com/midbel/sweet/internal/lang/ast"
)

type SqlStatementSpec int8

const (
	ContainsSql SqlStatementSpec = iota + 1
	ModifiesSql
	ReadsSql
)

type CreateProcedureStatement struct {
	ast.CreateProcedureStatement
	Specific      string
	Deterministic bool
	NullInput     bool
	Options       ast.Statement
	StmtSpec      SqlStatementSpec
}

type HandlerType int8

const (
	ExitHandler HandlerType = iota + 1
	ContinueHandler
	UndoHandler
)

type Handler struct {
	Type      HandlerType
	Condition ast.Statement
	ast.Statement
}

func (h Handler) Keyword() (string, error) {
	return "DECLARE", nil
}
