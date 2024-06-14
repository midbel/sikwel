package lang

import (
	"github.com/midbel/sweet/internal/keywords"
)

var ansi = keywords.Set{
	{"create", "procedure"},
	{"create", "or", "replace", "procedure"},
	{"create", "table"},
	{"create", "view"},
	{"create", "temp", "view"},
	{"create", "temporary", "view"},
	{"create", "temp", "table"},
	{"create", "temporary", "table"},
	{"if", "not", "exists"},
	{"if", "exists"},
	{"declare"},
	{"default"},
	{"exists"},
	{"null"},
	{"select"},
	{"from"},
	{"where"},
	{"having"},
	{"limit"},
	{"offset"},
	{"fetch"},
	{"row"},
	{"rows"},
	{"next"},
	{"only"},
	{"group", "by"},
	{"order", "by"},
	{"as"},
	{"in"},
	{"inout"},
	{"out"},
	{"join"},
	{"on"},
	{"full", "join"},
	{"full", "outer", "join"},
	{"outer", "join"},
	{"left", "join"},
	{"left", "outer", "join"},
	{"right", "join"},
	{"right", "outer", "join"},
	{"inner", "join"},
	{"union"},
	{"intersect"},
	{"except"},
	{"all"},
	{"distinct"},
	{"and"},
	{"or"},
	{"asc"},
	{"desc"},
	{"nulls"},
	{"first"},
	{"last"},
	{"similar"},
	{"like"},
	{"ilike"},
	{"delete", "from"},
	{"truncate"},
	{"truncate", "table"},
	{"update"},
	{"merge"},
	{"merge", "into"},
	{"when", "matched"},
	{"when", "not", "matched"},
	{"set"},
	{"insert", "into"},
	{"values"},
	{"case"},
	{"when"},
	{"then"},
	{"end"},
	{"using"},
	{"begin"},
	{"read", "write"},
	{"read", "only"},
	{"repeatable", "read"},
	{"read", "committed"},
	{"read", "uncommitted"},
	{"isolation", "level"},
	{"start", "transaction"},
	{"set", "transaction"},
	{"savepoint"},
	{"release"},
	{"release", "savepoint"},
	{"rollback", "to", "savepoint"},
	{"commit"},
	{"rollback"},
	{"on", "conflict"},
	{"nothing"},
	{"while"},
	{"end", "while"},
	{"do"},
	{"if"},
	{"end", "if"},
	{"else"},
	{"elsif"},
	{"with"},
	{"recursive"},
	{"materialized"},
	{"return"},
	{"returning"},
	{"is"},
	{"isnull"},
	{"notnull"},
	{"not"},
	{"collate"},
	{"between"},
	{"cast"},
	{"filter"},
	{"window"},
	{"over"},
	{"partition", "by"},
	{"range"},
	{"groups"},
	{"preceding"},
	{"following"},
	{"unbounded", "preceding"},
	{"unbounded", "following"},
	{"current", "row"},
	{"exclude", "no", "others"},
	{"exclude", "current", "row"},
	{"exclude", "group"},
	{"exclude", "ties"},
	{"call"},
	{"constraint"},
	{"primary", "key"},
	{"foreign", "key"},
	{"references"},
	{"autoincrement"},
	{"unique"},
	{"check"},
	{"generated", "always"},
	{"stored"},
	{"language"},
	{"alter", "table"},
	{"rename", "to"},
	{"rename", "column"},
	{"rename", "constraint"},
	{"alter"},
	{"alter", "column"},
	{"add"},
	{"add", "column"},
	{"add", "constraint"},
	{"drop"},
	{"drop", "table"},
	{"drop", "view"},
	{"drop", "column"},
	{"drop", "constraint"},
	{"to"},
	{"true"},
	{"false"},
	{"unknown"},
	{"cascade"},
	{"restrict"},
	{"restart", "identity"},
	{"continue", "identity"},
	{"grant"},
	{"revoke"},
	{"all", "privileges"},
}

func GetKeywords() keywords.Set {
	return ansi
}
