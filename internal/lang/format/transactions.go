package format

import (
	"fmt"

	"github.com/midbel/sweet/internal/lang/ast"
)

func (w *Writer) FormatStartTransaction(stmt ast.StartTransaction) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	if stmt.Mode > 0 {
		w.WriteBlank()
		switch stmt.Mode {
		case ast.ModeReadWrite:
			w.WriteKeyword("READ WRITE")
		case ast.ModeReadOnly:
			w.WriteKeyword("READ ONLY")
		default:
			return fmt.Errorf("unknown transaction mode")
		}
	}
	if stmt.Body != nil {
		w.WriteNL()
		if err := w.FormatStatement(stmt.Body); err != nil {
			return err
		}
	}
	if stmt.End == nil {
		return nil
	}
	return w.FormatStatement(stmt.End)
}

func (w *Writer) FormatSetTransaction(stmt ast.SetTransaction) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	if stmt.Level > 0 {
		w.WriteBlank()
		w.WriteKeyword("ISOLATION LEVEL")
		w.WriteBlank()
	}
	if stmt.Mode > 0 {
		w.WriteBlank()
		switch stmt.Mode {
		case ast.ModeReadWrite:
			w.WriteKeyword("READ WRITE")
		case ast.ModeReadOnly:
			w.WriteKeyword("READ ONLY")
		default:
			return fmt.Errorf("unknown transaction mode")
		}
	}
	w.WriteBlank()
	return nil
}

func (w *Writer) FormatSavepoint(stmt ast.Savepoint) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	if stmt.Name != "" {
		w.WriteBlank()
		w.WriteString(stmt.Name)
	}
	return nil
}

func (w *Writer) FormatReleaseSavepoint(stmt ast.ReleaseSavepoint) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	if stmt.Name != "" {
		w.WriteBlank()
		w.WriteString(stmt.Name)
	}
	return nil
}

func (w *Writer) FormatRollbackSavepoint(stmt ast.RollbackSavepoint) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	if stmt.Name != "" {
		w.WriteBlank()
		w.WriteString(stmt.Name)
	}
	return nil
}

func (w *Writer) FormatCommit(stmt ast.Commit) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	return nil
}

func (w *Writer) FormatRollback(stmt ast.Rollback) error {
	kw, _ := stmt.Keyword()
	w.WriteKeyword(kw)
	return nil
}
