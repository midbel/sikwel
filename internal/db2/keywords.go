package db2

import (
	"github.com/midbel/sweet/lang"
)

var keywords = KeywordSet{
	{"prepare"},
	{"leave"},
	{"fetch"},
	{"open"},
	{"elseif"},
	{"values"},
}

func GetKeywords() KeywordSet {
	ansi := lang.GetKeywords()
	return ansi.Merge(db2)
}
