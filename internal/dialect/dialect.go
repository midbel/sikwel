package dialect

import (
	"fmt"
	"io"

	"github.com/midbel/sweet/internal/lang"
	"github.com/midbel/sweet/internal/postgres"
	"github.com/midbel/sweet/internal/sqlite"
)

const (
	Ansi  = "ansi"
	Maria = "maria"
	Mysql = "mysql"
	Db2   = "db2"
)

type Parser interface {
	Parse() (lang.Statement, error)
}

type Writer interface {
	Format(io.Reader) error
	SetIndent(string)
	SetCompact(bool)
	SetKeywordUppercase(bool)
	SetFunctionUppercase(bool)
	SetKeepComments(bool)
}

func ParseAnsi(r io.Reader) (Parser, error) {
	return lang.NewParser(r)
}

func ParseSqlite(r io.Reader) (Parser, error) {
	return sqlite.NewParser(r)
}

func ParsePostgres(r io.Reader) (Parser, error) {
	return postgres.NewParser(r)
}

func FormatAnsi(w io.Writer) Writer {
	return lang.NewWriter(w)
}

func FormatSqlite(w io.Writer) Writer {
	return sqlite.NewWriter(w)
}

func FormatPostgres(w io.Writer) Writer {
	return postgres.NewWriter(w)
}

func NewParser(r io.Reader, dialect string) (Parser, error) {
	switch dialect {
	case "", Ansi:
		return ParseAnsi(r)
	case sqlite.Vendor:
		return ParseSqlite(r)
	case postgres.Vendor:
		return ParsePostgres(r)
	case Mysql:
	case Maria:
	case Db2:
	default:
		return nil, fmt.Errorf("dialect %q not yet supported", dialect)
	}
	return nil, fmt.Errorf("dialect %q not yet supported", dialect)
}

func NewWriter(w io.Writer, dialect string) (Writer, error) {
	switch dialect {
	case "", Ansi:
		return FormatAnsi(w), nil
	case sqlite.Vendor:
		return FormatSqlite(w), nil
	case postgres.Vendor:
		return FormatPostgres(w), nil
	case Mysql:
	case Maria:
	case Db2:
	default:
		return nil, fmt.Errorf("dialect %q not yet supported", dialect)
	}
	return nil, fmt.Errorf("dialect %q not yet supported", dialect)
}
