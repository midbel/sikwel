package format

import (
	"github.com/midbel/sweet/internal/lang/ast"
)

func (w *Writer) FormatGrant(stmt ast.GrantStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	if len(stmt.Privileges) == 0 {
		w.WriteStatement("ALL PRIVILEGES")
	} else {
		for i, p := range stmt.Privileges {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteKeyword(p)
		}
	}
	w.WriteBlank()
	w.WriteKeyword("TO")
	w.WriteBlank()
	for i, u := range stmt.Users {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		w.WriteString(u)
	}
	return nil
}

func (w *Writer) FormatRevoke(stmt ast.RevokeStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	w.WriteBlank()
	if len(stmt.Privileges) == 0 {
		w.WriteStatement("ALL")
	} else {
		for i, p := range stmt.Privileges {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteKeyword(p)
		}
	}
	w.WriteBlank()
	w.WriteKeyword("FROM")
	w.WriteBlank()
	for i, u := range stmt.Users {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		w.WriteString(u)
	}
	return nil
}
