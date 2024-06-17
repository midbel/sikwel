package lang

import (
	"github.com/midbel/sweet/internal/lang/ast"
)

type Formatter interface {
	Quote(string) string
}

type Parser interface {
	Parse() (ast.Statement, error)
}
