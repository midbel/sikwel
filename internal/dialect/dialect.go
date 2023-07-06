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
	Sqlite   = "sqlite"
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

func FormatSqlite(w io.Writer) Writer {
	return nil
}

func ParseSqlite(r io.Reader) (Parser, error) {
	return sqlite.NewParser(r)
}

func NewParser(r io.Reader, dialect string) (Parser, error) {
	switch dialect {
	case "", Ansi:
		return ParseAnsi(r)
	case Sqlite:
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
	case Sqlite:
	case Postgres:
	case Mysql:
	case Maria:
	case Db2:
	default:
		return nil, fmt.Errorf("dialect %q not yet supported", dialect)
	}
	return nil, fmt.Errorf("dialect %q not yet supported", dialect)
}
