package dialect

import (
	"fmt"
	"io"

	"github.com/midbel/sweet/internal/lang"
	"github.com/midbel/sweet/internal/sqlite"
)

const (
	Ansi     = "ansi"
	Postgres = "postgres"
	Maria    = "maria"
	Mysql    = "mysql"
	Db2      = "db2"
)

type Parser interface {
	Parse() (lang.Statement, error)
}

type Writer interface {
	Format(io.Reader) error
}

func ParseAnsi(r io.Reader) (Parser, error) {
	return lang.NewParser(r)
}

func FormatAnsi(w io.Writer) Writer {
	return lang.NewWriter(w)
}

func ParseSqlite(r io.Reader) (Parser, error) {
	return sqlite.NewParser(r)
}

func FormatSqlite(w io.Writer) Writer {
	return lang.NewWriter(w)
}

func NewParser(r io.Reader, dialect string) (Parser, error) {
	switch dialect {
	case "", Ansi:
		return ParseAnsi(r)
	case sqlite.Vendor:
		return ParseSqlite(r)
	case Postgres:
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
	case Postgres:
	case Mysql:
	case Maria:
	case Db2:
	default:
		return nil, fmt.Errorf("dialect %q not yet supported", dialect)
	}
	return nil, fmt.Errorf("dialect %q not yet supported", dialect)
}
