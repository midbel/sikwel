package lang

import (
	"fmt"
)

func (p *Parser) ParseBegin() (Statement, error) {
	p.Next()
	stmt, err := p.ParseBody(func() bool {
		return p.Done() || p.IsKeyword("END")
	})
	if err == nil {
		p.Next()
	}
	return stmt, err
}

func (p *Parser) parseSetTransaction() (Statement, error) {
	p.Next()
	var (
		stmt SetTransaction
		err  error
	)
	if p.IsKeyword("ISOLATION LEVEL") {
		p.Next()
		switch {
		case p.IsKeyword("REPEATABLE READ"):
			stmt.Level = LevelReadRepeat
		case p.IsKeyword("READ COMMITTED"):
			stmt.Level = LevelReadCommit
		case p.IsKeyword("READ UNCOMMITTED"):
			stmt.Level = LevelReadUncommit
		case p.IsKeyword("SERIALIZABLE"):
			stmt.Level = LevelSerializable
		default:
			return nil, p.Unexpected("transaction")
		}
		p.Next()
	}
	switch {
	case p.IsKeyword("READ ONLY"):
		stmt.Mode = ModeReadOnly
		p.Next()
	case p.IsKeyword("READ WRITE"):
		stmt.Mode = ModeReadWrite
		p.Next()
	case p.Is(EOL):
	default:
		return nil, p.Unexpected("transaction")
	}
	return stmt, err
}

func (p *Parser) parseStartTransaction() (Statement, error) {
	defer func() {
		p.UnregisterParseFunc("SAVEPOINT")
		p.UnregisterParseFunc("COMMIT")
		p.UnregisterParseFunc("ROLLBACK")
		p.UnregisterParseFunc("RELEASE")
		p.UnregisterParseFunc("RELEASE SAVEPOINT")
		p.UnregisterParseFunc("ROLLBACK TO SAVEPOINT")
		p.UnregisterParseFunc("SET TRANSACTION")
	}()
	p.Next()

	var (
		stmt StartTransaction
		err  error
	)
	switch {
	case p.IsKeyword("READ ONLY"):
		stmt.Mode = ModeReadOnly
		p.Next()
	case p.IsKeyword("READ WRITE"):
		stmt.Mode = ModeReadWrite
		p.Next()
	case p.Is(EOL):
	default:
		return nil, p.Unexpected("transaction")
	}
	if !p.Is(EOL) {
		return nil, p.Unexpected("transaction")
	}
	p.Next()

	p.RegisterParseFunc("SAVEPOINT", p.parseSavepoint)
	p.RegisterParseFunc("RELEASE", p.parseReleaseSavepoint)
	p.RegisterParseFunc("RELEASE SAVEPOINT", p.parseReleaseSavepoint)
	p.RegisterParseFunc("ROLLBACK TO SAVEPOINT", p.parseRollbackSavepoint)
	p.RegisterParseFunc("COMMIT", p.parseCommit)
	p.RegisterParseFunc("ROLLBACK", p.parseRollback)
	p.RegisterParseFunc("SET TRANSACTION", p.parseSetTransaction)

	stmt.Body, err = p.ParseBody(p.KwCheck("END", "COMMIT", "ROLLBACK"))
	if err != nil {
		return nil, err
	}
	switch {
	case p.IsKeyword("END") || p.IsKeyword("COMMIT"):
		stmt.End = Commit{}
	case p.IsKeyword("ROLLBACK"):
		stmt.End = Rollback{}
	default:
		return nil, p.Unexpected("transaction")
	}
	p.Next()
	return stmt, err
}

func (p *Parser) parseSavepoint() (Statement, error) {
	p.Next()
	var (
		stmt Savepoint
		err  error
	)
	if p.Is(Ident) {
		stmt.Name = p.GetCurrLiteral()
		p.Next()
	}
	return stmt, err
}

func (p *Parser) parseReleaseSavepoint() (Statement, error) {
	p.Next()
	var (
		stmt ReleaseSavepoint
		err  error
	)
	if !p.Is(Ident) {
		return nil, p.Unexpected("release savepoint")
	}
	stmt.Name = p.GetCurrLiteral()
	p.Next()
	return stmt, err
}

func (p *Parser) parseRollbackSavepoint() (Statement, error) {
	p.Next()
	var (
		stmt RollbackSavepoint
		err  error
	)
	if !p.Is(Ident) {
		return nil, p.Unexpected("rollback savepoint")
	}
	stmt.Name = p.GetCurrLiteral()
	p.Next()
	return stmt, err
}

func (p *Parser) parseCommit() (Statement, error) {
	p.Next()
	return Commit{}, nil
}

func (p *Parser) parseRollback() (Statement, error) {
	p.Next()
	return Rollback{}, nil
}

func (w *Writer) FormatStartTransaction(stmt StartTransaction) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	if stmt.Mode > 0 {
		w.WriteBlank()
		switch stmt.Mode {
		case ModeReadWrite:
			w.WriteKeyword("READ WRITE")
		case ModeReadOnly:
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

func (w *Writer) FormatSetTransaction(stmt SetTransaction) error {
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
		case ModeReadWrite:
			w.WriteKeyword("READ WRITE")
		case ModeReadOnly:
			w.WriteKeyword("READ ONLY")
		default:
			return fmt.Errorf("unknown transaction mode")
		}
	}
	w.WriteBlank()
	return nil
}

func (w *Writer) FormatSavepoint(stmt Savepoint) error {
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

func (w *Writer) FormatReleaseSavepoint(stmt ReleaseSavepoint) error {
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

func (w *Writer) FormatRollbackSavepoint(stmt RollbackSavepoint) error {
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

func (w *Writer) FormatCommit(stmt Commit) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	return nil
}

func (w *Writer) FormatRollback(stmt Rollback) error {
	kw, _ := stmt.Keyword()
	w.WriteStatement(kw)
	return nil
}
