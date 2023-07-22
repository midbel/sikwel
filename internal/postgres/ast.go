package postgres

import (
	"github.com/midbel/sweet/internal/lang"
)

type TruncateStatement struct {
	Only     bool
	Tables   []lang.Statement
	Identity string
	Cascade  bool
	Restrict bool
}

func (s TruncateStatement) Keyword() (string, error) {
	return "TRUNCATE TABLE", nil
}

type MergeStatement struct {
}

func (_ MergeStatement) Keyword() (string, error) {
	return "MERGE", nil
}

type CopyStatement struct {
	Import  bool
	Table   lang.Statement
	Columns []string
	Query   lang.Statement
	Where   lang.Statement
	File    string
	Program bool
	Stdio   bool

	Format       string
	Delimiter    string
	Freeze       string
	Null         string
	Header       string
	Quote        string
	Escape       string
	ForceQuote   string
	ForceNotNull string
	ForceNull    string
	Encoding     string
}

func (_ CopyStatement) Keyword() (string, error) {
	return "COPY", nil
}

type Order struct {
	lang.Statement
	Orient string
	Nulls  string
}
