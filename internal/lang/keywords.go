package lang

import (
	"sort"
	"strings"
)

type KeywordSet [][]string

var keywords = KeywordSet{
	{"create", "procedure"},
	{"create", "or", "replace", "procedure"},
	{"declare"},
	{"default"},
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
	{"like"},
	{"ilike"},
	{"delete", "from"},
	{"update"},
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
	{"conflict"},
	{"nothing"},
	{"while"},
	{"end", "while"},
	{"do"},
	{"if"},
	{"end", "if"},
	{"else"},
	{"elsif"},
	{"with"},
	{"return"},
	{"returning"},
	{"is"},
	{"not"},
	{"collate"},
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
}

func GetKeywords() KeywordSet {
	return keywords
}

func (ks KeywordSet) Merge(other KeywordSet) KeywordSet {
	return append(ks, other...)
}

func (ks KeywordSet) Len() int {
	return len(ks)
}

func (ks KeywordSet) Find(str string) int {
	return sort.Search(ks.Len(), func(i int) bool {
		return str <= ks[i][0]
	})
}

func (ks KeywordSet) Is(str []string) (string, bool) {
	var (
		n = ks.Len()
		s = strings.ToLower(str[0])
		i = ks.Find(s)
	)
	if i >= n || ks[i][0] != s {
		return "", false
	}
	if len(ks[i]) == 1 && len(str) == 1 && i+1 < n && ks[i+1][0] != s {
		return s, true
	}
	var (
		got  = strings.ToLower(strings.Join(str, " "))
		want string
	)
	for _, kw := range ks[i:] {
		if kw[0] != s {
			break
		}
		want = strings.Join(kw, " ")
		switch {
		case want == got:
			return got, true
		case strings.HasPrefix(want, got):
			return got, false
		default:
		}
	}
	return "", false
}

func (ks KeywordSet) prepare() {
	seen := make(map[string]struct{})
	for i := range ks {
		str := strings.Join(ks[i], "")
		if _, ok := seen[str]; ok {
			continue
		}
		seen[str] = struct{}{}
		for j := range ks[i] {
			ks[i][j] = strings.ToLower(ks[i][j])
		}
	}
	sort.Slice(ks, func(i, j int) bool {
		fst := strings.Join(ks[i], " ")
		lst := strings.Join(ks[j], " ")
		return fst < lst
	})
}
