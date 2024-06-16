package db2

import (
	"github.com/midbel/sweet/internal/keywords"
	"github.com/midbel/sweet/internal/lang"
)

var kw = keywords.Set{
	{"label", "on"},
	{"set", "option"},
	{"read", "sql", "data"},
	{"modifies", "sql", "data"},
}

func GetKeywords() keywords.Set {
	return kw.Merge(lang.GetKeywords())
}
