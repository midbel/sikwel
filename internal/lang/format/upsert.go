package format

import "github.com/midbel/sweet/internal/lang/ast"

func (w *Writer) FormatMerge(stmt ast.MergeStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()
	w.WriteKeyword("INTO")
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Target, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("USING")
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Source, false); err != nil {
		return err
	}
	w.WriteBlank()
	w.WriteKeyword("ON")
	w.WriteBlank()
	if err := w.FormatExpr(stmt.Join, false); err != nil {
		return err
	}
	for _, a := range stmt.Actions {
		m, ok := a.(ast.MatchStatement)
		if !ok {
			return w.CanNotUse("merge", a)
		}
		w.WriteNL()
		w.WritePrefix()
		if err := w.FormatMatch(m); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatMatch(stmt ast.MatchStatement) error {
	w.WriteKeyword("WHEN")
	w.WriteBlank()
	switch stmt.Statement.(type) {
	case ast.DeleteStatement:
		w.WriteKeyword("MATCHED")
	case ast.UpdateStatement:
		w.WriteKeyword("MATCHED")
	case ast.InsertStatement:
		w.WriteKeyword("NOT MATCHED")
	default:
		return w.CanNotUse("merge", stmt.Statement)
	}
	if stmt.Condition != nil {
		w.WriteBlank()
		w.WriteKeyword("AND")
		w.WriteBlank()
		if err := w.FormatExpr(stmt.Condition, false); err != nil {
			return err
		}
	}
	w.WriteBlank()
	w.WriteKeyword("THEN")
	w.WriteNL()
	w.WritePrefix()

	switch stmt := stmt.Statement.(type) {
	case ast.DeleteStatement:
		w.WriteKeyword("DELETE")
	case ast.UpdateStatement:
		w.WriteKeyword("UPDATE")
		w.WriteBlank()
		w.WriteKeyword("SET")
		w.WriteBlank()

		compact := w.Compact
		w.Compact = true
		defer func() {
			w.Compact = compact
		}()
		if err := w.FormatAssignment(stmt.List); err != nil {
			return err
		}
	case ast.InsertStatement:
		w.WriteKeyword("INSERT")
		w.WriteBlank()
		if len(stmt.Columns) > 0 {
			w.WriteString("(")
			for i := range stmt.Columns {
				if i > 0 {
					w.WriteString(",")
					w.WriteBlank()
				}
				w.WriteString(stmt.Columns[i])
			}
			w.WriteString(")")
			w.WriteBlank()
		}
		values, ok := stmt.Values.(ast.ValuesStatement)
		if !ok {
			return w.CanNotUse("merge", stmt.Values)
		}
		compact := w.Compact
		w.Compact = true
		defer func() {
			w.Compact = compact
		}()
		if err := w.FormatValues(values); err != nil {
			return err
		}
	default:
		return w.CanNotUse("merge", stmt)
	}
	return nil
}

func (w *Writer) FormatTruncate(stmt ast.TruncateStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()
	if len(stmt.Tables) == 0 {
		w.WriteString("*")
		return nil
	}
	for i := range stmt.Tables {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		w.WriteString(stmt.Tables[i])
	}
	return nil
}

func (w *Writer) FormatDelete(stmt ast.DeleteStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()
	w.WriteString(stmt.Table)
	if stmt.Where != nil {
		w.WriteNL()
		if err := w.FormatWhere(stmt.Where); err != nil {
			return err
		}
	}
	if stmt.Return != nil {
		w.WriteNL()
		if err := w.FormatReturning(stmt.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatUpdate(stmt ast.UpdateStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()

	switch stmt := stmt.Table.(type) {
	case ast.Name:
		w.FormatName(stmt)
	case ast.Alias:
		if err := w.FormatAlias(stmt); err != nil {
			return err
		}
	default:
		return w.CanNotUse("update", stmt)
	}
	w.WriteBlank()
	w.WriteKeyword("SET")
	w.WriteNL()

	if err := w.FormatAssignment(stmt.List); err != nil {
		return err
	}

	if len(stmt.Tables) > 0 {
		w.WriteNL()
		if err := w.FormatFrom(stmt.Tables); err != nil {
			return err
		}
	}
	if stmt.Where != nil {
		w.WriteNL()
		if err := w.FormatWhere(stmt.Where); err != nil {
			return err
		}
	}
	if stmt.Return != nil {
		w.WriteNL()
		if err := w.FormatReturning(stmt.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatInsert(stmt ast.InsertStatement) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	w.WriteBlank()

	if err := w.FormatExpr(stmt.Table, false); err != nil {
		return err
	}
	if len(stmt.Columns) > 0 {
		w.WriteBlank()
		w.WriteString("(")
		for i, c := range stmt.Columns {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteString(c)
		}
		w.WriteString(")")
	}
	w.WriteBlank()
	if err := w.FormatInsertValues(stmt.Values); err != nil {
		return err
	}
	if stmt.Upsert != nil {
		w.WriteNL()
		if err := w.FormatUpsert(stmt.Upsert); err != nil {
			return err
		}
	}
	if stmt.Return != nil {
		w.WriteNL()
		if err := w.FormatReturning(stmt.Return); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) FormatInsertValues(values ast.Statement) error {
	if values == nil {
		return nil
	}
	var err error
	switch stmt := values.(type) {
	case ast.ValuesStatement:
		err = w.formatValues(stmt, true)
	case ast.SelectStatement:
		w.WriteNL()
		err = w.FormatSelect(stmt)
	default:
		err = w.CanNotUse("values", values)
	}
	return err
}

func (w *Writer) FormatUpsert(stmt ast.Statement) error {
	if stmt == nil {
		return nil
	}
	upsert, ok := stmt.(ast.Upsert)
	if !ok {
		return w.CanNotUse("insert(upsert)", stmt)
	}
	w.WriteKeyword("ON CONFLICT")
	w.WriteBlank()

	if len(upsert.Columns) > 0 {
		w.WriteString("(")
		for i, s := range upsert.Columns {
			if i > 0 {
				w.WriteString(",")
				w.WriteBlank()
			}
			w.WriteString(s)
		}
		w.WriteString(")")
	}
	w.WriteBlank()
	if len(upsert.List) == 0 {
		w.WriteKeyword("DO NOTHING")
		return nil
	}
	w.WriteKeyword("UPDATE SET")
	w.WriteNL()
	if err := w.FormatAssignment(upsert.List); err != nil {
		return err
	}
	return w.FormatWhere(upsert.Where)
}

func (w *Writer) FormatAssignment(list []ast.Statement) error {
	var err error
	for i, s := range list {
		if i > 0 {
			w.WriteString(",")
			w.WriteBlank()
		}
		ass, ok := s.(ast.Assignment)
		if !ok {
			return w.CanNotUse("assignment", s)
		}
		w.WritePrefix()
		switch field := ass.Field.(type) {
		case ast.Name:
			w.FormatName(field)
		case ast.List:
			err = w.formatList(field)
		default:
			return w.CanNotUse("assignment", s)
		}
		if err != nil {
			return err
		}
		w.WriteString("=")
		switch value := ass.Value.(type) {
		case ast.List:
			err = w.formatList(value)
		default:
			err = w.FormatExpr(value, false)
		}
		if err != nil {
			return err
		}
	}
	return err
}

func (w *Writer) FormatReturning(stmt ast.Statement) error {
	if stmt == nil {
		return nil
	}
	w.WriteKeyword("RETURNING")
	w.WriteBlank()

	list, ok := stmt.(ast.List)
	if !ok {
		return w.FormatExpr(stmt, false)
	}
	return w.formatStmtSlice(list.Values)
}
