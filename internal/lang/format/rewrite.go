package format

import (
	"fmt"

	"github.com/midbel/sweet/internal/lang/ast"
)

func (w *Writer) Rewrite(stmt ast.Statement) (ast.Statement, error) {
	if w.Rules.KeepAsIs() {
		return stmt, nil
	}
	var err error
	if w.Rules.ReplaceSubqueryWithCte() || w.Rules.All() {
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
	var with ast.WithStatement
	if w, ok := stmt.(ast.WithStatement); ok {
		with = w
	} else {
		with.Statement = stmt
	}

	if q, ok := with.Statement.(ast.SelectStatement); ok {
		q, qs, err := w.replaceSubqueries(q)
		if err != nil {
			return nil, err
		}
		with.Statement = q
		with.Queries = append(with.Queries, qs...)
	}
	return with, nil
}

func (w *Writer) replaceSubqueries(stmt ast.SelectStatement) (ast.Statement, []ast.Statement, error) {
	var qs []ast.Statement

	if !w.Rules.SetMissingCteAlias() {
		rules := w.Rules
		w.Rules |= RewriteMissCteAlias
		defer func() {
			w.Rules = rules
		}()
	}

	for i, q := range stmt.Tables {
		j, ok := q.(ast.Join)
		if !ok {
			continue
		}
		n := fmt.Sprintf("q%03d", i+1)
		if a, ok := j.Table.(ast.Alias); ok {
			n = a.Alias
			q = a.Statement
		} else {
			q = j.Table

		}
		q, ok := q.(ast.SelectStatement)
		if !ok {
			continue
		}
		cte := ast.CteStatement{
			Ident:     n,
			Statement: q,
		}
		c, err := w.rewriteCte(cte)
		if err != nil {
			return nil, nil, err
		}
		qs = append(qs, c)

		j.Table = ast.Name{
			Parts: []string{n},
		}
		stmt.Tables[i] = j
	}
	return stmt, qs, nil
}

func (w *Writer) replaceCteWithSubquery(stmt ast.Statement) (ast.Statement, error) {
	if _, ok := stmt.(ast.WithStatement); !ok {
		return stmt, nil
	}
	return stmt, nil
}

func (w *Writer) rewrite(stmt ast.Statement) (ast.Statement, error) {
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
	if w.Rules.UseStdOp() || w.Rules.All() {
		stmt = ast.ReplaceOp(stmt)
	}
	if w.Rules.UseStdExpr() || w.Rules.All() {
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
