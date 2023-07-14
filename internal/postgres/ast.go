package postgres

import (
	"github.com/midbel/sweet/internal/lang"
)

type TruncateStatement struct {
	Only bool
	Tables   []lang.Statement
	Identity string
	Cascade  bool
	Restrict bool
}

func (s TruncateStatement) Keyword() (string, error) {
	return "TRUNCATE", nil
}
