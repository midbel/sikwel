package lang

import (
	"errors"
	"io"
	"slices"

	"github.com/midbel/sweet/internal/lang/ast"
)

func Complexity(r io.Reader) (int, error) {
	p, err := NewParser(r)
	if err != nil {
		return 0, err
	}
	var total int
	for {
		stmt, err := p.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return total, nil
		}
		total += measureQuery(stmt)
	}
	return total, nil
}

func measureQuery(stmt ast.Statement) int {
	var total int
	switch stmt := stmt.(type) {
	case ast.TruncateStatement:
		total++
	case ast.DeleteStatement:
	case ast.UpdateStatement:
	case ast.InsertStatement:
	case ast.MergeStatement:
	case ast.ValuesStatement:
	case ast.CallStatement:
	case ast.UnionStatement:
	case ast.ExceptStatement:
	case ast.IntersectStatement:
	case ast.SelectStatement:
		total = measureSelect(stmt)
	case ast.WithStatement:
		total = measureWith(stmt)
	case ast.CteStatement:
		total = measureCte(stmt)
	case ast.Set:
	case ast.If:
	case ast.While:
	case ast.Return:
	case ast.Alias:
		total = measureQuery(stmt.Statement)
	case ast.Join:
		total = measureJoin(stmt)
	case ast.Case:
		total = measureCase(stmt)
	case ast.When:
		total = measureWhen(stmt)
	case ast.Group:
		total = measureQuery(stmt.Statement)
	case ast.Binary:
		total = measureBinary(stmt)
	case ast.Unary:
		total = measureQuery(stmt.Right)
	case ast.Exists:
	case ast.Between:
	case ast.All:
	case ast.Any:
	case ast.Is:
	default:
		// pass
	}
	return total
}

func measureBinary(stmt ast.Binary) int {
	var total int
	if stmt.IsRelation() {
		total++
	}
	total += measureQuery(stmt.Left)
	total += measureQuery(stmt.Right)
	return total

}

func measureCase(stmt ast.Case) int {
	var total int
	for _, q := range stmt.Body {
		total++
		total += measureQuery(q)
	}
	if stmt.Else != nil {
		total++
		total += measureQuery(stmt.Else)
	}
	return total
}

func measureWhen(stmt ast.When) int {
	return measureQuery(stmt.Cdt) + measureQuery(stmt.Body)
}

func measureJoin(stmt ast.Join) int {
	var total int
	if a, ok := stmt.Table.(ast.Alias); ok {
		stmt.Table = a.Statement
		return measureJoin(stmt)
	} else {
		total += measureQuery(stmt.Table)
	}
	return total + measureQuery(stmt.Where)
}

func measureSelect(stmt ast.SelectStatement) int {
	var (
		total int
		list  []ast.Statement
	)
	list = slices.Concat(list, stmt.Tables, stmt.Columns)
	for _, q := range list {
		total += measureQuery(q)
	}
	return total + 1
}

func measureWith(stmt ast.WithStatement) int {
	return 0
}

func measureCte(stmt ast.CteStatement) int {
	return 0
}
