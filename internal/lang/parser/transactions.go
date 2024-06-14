package parser

import (
	"github.com/midbel/sweet/internal/lang/ast"
	"github.com/midbel/sweet/internal/token"
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
	case p.Is(token.EOL):
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
	case p.Is(token.EOL):
	default:
		return nil, p.Unexpected("transaction")
	}
	if !p.Is(token.EOL) {
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
	if p.Is(token.Ident) {
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
	if !p.Is(token.Ident) {
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
	if !p.Is(token.Ident) {
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
