package rules

import (
	"github.com/midbel/sweet/internal/token"
)

type LintMessage[T any] struct {
	Severity Level
	Rule     string
	Message  string
	Position token.Position
	Body     T
}

type LintInfo struct {
	Rule    string
	Enabled bool
}
