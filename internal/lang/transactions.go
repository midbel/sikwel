package lang

import (
	"fmt"

	"github.com/midbel/sweet/internal/lang/ast"
)

func (p *Parser) ParseBegin() (ast.Statement, error) {
	p.Next()
	stmt, err := p.ParseBody(func() bool {
		return p.Done() || p.IsKeyword("END")
	})
	if err == nil {
		p.Next()
	}
	return stmt, err
}

func (p *Parser) parseSetTransaction() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.SetTransaction
		err  error
	)
	if p.IsKeyword("ISOLATION LEVEL") {
		p.Next()
		switch {
		case p.IsKeyword("REPEATABLE READ"):
			stmt.Level = ast.LevelReadRepeat
		case p.IsKeyword("READ COMMITTED"):
			stmt.Level = ast.LevelReadCommit
		case p.IsKeyword("READ UNCOMMITTED"):
			stmt.Level = ast.LevelReadUncommit
		case p.IsKeyword("SERIALIZABLE"):
			stmt.Level = ast.LevelSerializable
		default:
			return nil, p.Unexpected("transaction")
		}
		p.Next()
	}
	switch {
	case p.IsKeyword("READ ONLY"):
		stmt.Mode = ast.ModeReadOnly
		p.Next()
	case p.IsKeyword("READ WRITE"):
		stmt.Mode = ast.ModeReadWrite
		p.Next()
	case p.Is(EOL):
	default:
		return nil, p.Unexpected("transaction")
	}
	return stmt, err
}

func (p *Parser) parseStartTransaction() (ast.Statement, error) {
	p.Next()

	var (
		stmt ast.StartTransaction
		err  error
	)
	switch {
	case p.IsKeyword("READ ONLY"):
		stmt.Mode = ast.ModeReadOnly
		p.Next()
	case p.IsKeyword("READ WRITE"):
		stmt.Mode = ast.ModeReadWrite
		p.Next()
	case p.Is(EOL):
	default:
		return nil, p.Unexpected("transaction")
	}
	if !p.Is(EOL) {
		return nil, p.Unexpected("transaction")
	}
	p.Next()

	stmt.Body, err = p.ParseBody(p.KwCheck("END", "COMMIT", "ROLLBACK"))
	if err != nil {
		return nil, err
	}
	switch {
	case p.IsKeyword("END") || p.IsKeyword("COMMIT"):
		stmt.End = ast.Commit{}
	case p.IsKeyword("ROLLBACK"):
		stmt.End = ast.Rollback{}
	default:
		return nil, p.Unexpected("transaction")
	}
	p.Next()
	return stmt, err
}

func (p *Parser) parseSavepoint() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.Savepoint
		err  error
	)
	if p.Is(Ident) {
		stmt.Name = p.GetCurrLiteral()
		p.Next()
	}
	return stmt, err
}

func (p *Parser) parseReleaseSavepoint() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.ReleaseSavepoint
		err  error
	)
	if !p.Is(Ident) {
		return nil, p.Unexpected("release savepoint")
	}
	stmt.Name = p.GetCurrLiteral()
	p.Next()
	return stmt, err
}

func (p *Parser) parseRollbackSavepoint() (ast.Statement, error) {
	p.Next()
	var (
		stmt ast.RollbackSavepoint
		err  error
	)
	if !p.Is(Ident) {
		return nil, p.Unexpected("rollback savepoint")
	}
	stmt.Name = p.GetCurrLiteral()
	p.Next()
	return stmt, err
}

func (p *Parser) parseCommit() (ast.Statement, error) {
	p.Next()
	return ast.Commit{}, nil
}

func (p *Parser) parseRollback() (ast.Statement, error) {
	p.Next()
	return ast.Rollback{}, nil
}

func (w *Writer) FormatStartTransaction(stmt ast.StartTransaction) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
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
	w.WriteStatement(kw)
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
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Name != "" {
		w.WriteBlank()
		w.WriteString(stmt.Name)
	}
	return nil
}

func (w *Writer) FormatReleaseSavepoint(stmt ast.ReleaseSavepoint) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Name != "" {
		w.WriteBlank()
		w.WriteString(stmt.Name)
	}
	return nil
}

func (w *Writer) FormatRollbackSavepoint(stmt ast.RollbackSavepoint) error {
	w.Enter()
	defer w.Leave()

	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Name != "" {
		w.WriteBlank()
		w.WriteString(stmt.Name)
	}
	return nil
}

func (w *Writer) FormatCommit(stmt ast.Commit) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	return nil
}

func (w *Writer) FormatRollback(stmt ast.Rollback) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	return nil
}
