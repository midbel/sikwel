package sqlite

import (
	"github.com/midbel/sweet/internal/lang"
)

const (
	CollateBinary = "BINARY"
	CollateNocase = "NOCASE"
	CollateTrim   = "RTRIM"
)

var keywords = lang.KeywordSet{
	{"collate"},
	{"replace", "into"},
	{"insert", "or", "abort", "into"},
	{"insert", "or", "fail", "into"},
	{"insert", "or", "ignore", "into"},
	{"insert", "or", "replace", "into"},
	{"insert", "or", "rollback", "into"},
	{"update", "or", "abort"},
	{"update", "or", "fail"},
	{"update", "or", "ignore"},
	{"update", "or", "replace"},
	{"update", "or", "rollback"},
	{"vacuum"},
	{"into"},
}
