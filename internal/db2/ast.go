package db2

import (
	"github.com/midbel/sweet/internal/lang/ast"
)

type SqlStatementSpec int8

const (
	ContainsSql SqlStatementSpec = iota
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
