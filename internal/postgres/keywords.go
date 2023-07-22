package postgres

import (
	"github.com/midbel/sweet/internal/lang"
)

var keywords = lang.KeywordSet{
	{"truncate"},
	{"truncate", "table"},
	{"restart", "identity"},
	{"continue", "identity"},
	{"cascade"},
	{"restrict"},
	{"copy"},
	{"merge"},
}
