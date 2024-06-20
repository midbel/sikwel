package format

import (
	"github.com/midbel/sweet/internal/lang/ast"
)

func (w *Writer) Rewrite(stmt ast.Statement) (ast.Statement, error) {
	var err error
	if w.Rules.ReplaceSubqueryWithCte() {
		stmt, err = w.replaceSubqueryWithCte(stmt)
	} else if w.Rules.ReplaceCteWithSubquery() {
		stmt, err = w.replaceCteWithSubquery(stmt)
	}
	if err != nil {
		return nil, err
	}
	return w.rewrite(stmt)
}

func (w *Writer) replaceSubqueryWithCte(stmt ast.Statement) (ast.Statement, error) {
	return stmt, nil
}

func (w *Writer) replaceCteWithSubquery(stmt ast.Statement) (ast.Statement, error) {
	return stmt, nil
}

func (w *Writer) rewrite(stmt ast.Statement) (ast.Statement, error) {
	if w.Rules.KeepAsIs() {
		return stmt, nil
	}
	switch st := stmt.(type) {
	case ast.SelectStatement:
		stmt, _ = w.rewriteSelect(st)
	case ast.UpdateStatement:
		stmt, _ = w.rewriteUpdate(st)
	case ast.DeleteStatement:
		stmt, _ = w.rewriteDelete(st)
	case ast.WithStatement:
		stmt, _ = w.rewriteWith(st)
	case ast.CteStatement:
		stmt, _ = w.rewriteCte(st)
	case ast.UnionStatement:
		stmt, _ = w.rewriteUnion(st)
	case ast.ExceptStatement:
		stmt, _ = w.rewriteExcept(st)
	case ast.IntersectStatement:
		stmt, _ = w.rewriteIntersect(st)
	case ast.Binary:
		stmt, _ = w.rewriteBinary(st)
	default:
	}
	return stmt, nil
}

func (w *Writer) rewriteBinary(stmt ast.Binary) (ast.Statement, error) {
	if stmt.IsRelation() {
		stmt.Left, _ = w.rewrite(stmt.Left)
		stmt.Right, _ = w.rewrite(stmt.Right)
		return stmt, nil
	}
	if w.Rules.UseStdOp() {
		return ast.ReplaceOp(stmt), nil
	}
	if w.Rules.UseStdExpr() {
		return ast.ReplaceExpr(stmt), nil
	}
	return stmt, nil
}

func (w *Writer) rewriteWith(stmt ast.WithStatement) (ast.Statement, error) {
	for i := range stmt.Queries {
		stmt.Queries[i], _ = w.rewrite(stmt.Queries[i])
	}
	stmt.Statement, _ = w.rewrite(stmt.Statement)
	return stmt, nil
}

func (w *Writer) rewriteCte(stmt ast.CteStatement) (ast.Statement, error) {
	if len(stmt.Columns) == 0 && w.Rules.SetMissingCteAlias() {
		if gn, ok := stmt.Statement.(interface{ GetNames() []string }); ok {
			stmt.Columns = gn.GetNames()
		}
	}
	stmt.Statement, _ = w.rewrite(stmt.Statement)
	return stmt, nil
}

func (w *Writer) rewriteUnion(stmt ast.UnionStatement) (ast.Statement, error) {
	stmt.Left, _ = w.rewrite(stmt.Left)
	stmt.Right, _ = w.rewrite(stmt.Right)
	return stmt, nil
}

func (w *Writer) rewriteExcept(stmt ast.ExceptStatement) (ast.Statement, error) {
	stmt.Left, _ = w.rewrite(stmt.Left)
	stmt.Right, _ = w.rewrite(stmt.Right)
	return stmt, nil
}

func (w *Writer) rewriteIntersect(stmt ast.IntersectStatement) (ast.Statement, error) {
	stmt.Left, _ = w.rewrite(stmt.Left)
	stmt.Right, _ = w.rewrite(stmt.Right)
	return stmt, nil
}

func (w *Writer) rewriteSelect(stmt ast.SelectStatement) (ast.Statement, error) {
	stmt.Where, _ = w.rewrite(stmt.Where)
	return stmt, nil
}

func (w *Writer) rewriteUpdate(stmt ast.UpdateStatement) (ast.Statement, error) {
	stmt.Where, _ = w.rewrite(stmt.Where)
	return stmt, nil
}

func (w *Writer) rewriteDelete(stmt ast.DeleteStatement) (ast.Statement, error) {
	stmt.Where, _ = w.rewrite(stmt.Where)
	return stmt, nil
}
